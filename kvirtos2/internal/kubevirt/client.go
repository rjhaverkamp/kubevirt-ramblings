package kubevirt

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/kubevirt/kvirtos2/internal/models"
)

// Client wraps KubeVirt and Kubernetes clients with helper components
type Client struct {
	dynamicClient  dynamic.Interface
	k8sClient      kubernetes.Interface
	namespace      string
	vmBuilder      *VMBuilder
	statusMapper   *StatusMapper
	networkHandler *NetworkHandler
}

// NewClient creates a new KubeVirt client with all necessary components
func NewClient(kubeconfig, namespace string) (*Client, error) {
	config, err := buildKubernetesConfig(kubeconfig)
	if err != nil {
		return nil, NewConfigurationError("kubernetes", "failed to build config", err)
	}

	dynamicClient, k8sClient, err := createClients(config)
	if err != nil {
		return nil, NewConfigurationError("kubernetes", "failed to create clients", err)
	}

	if namespace == "" {
		namespace = DefaultNamespace
	}

	return &Client{
		dynamicClient:  dynamicClient,
		k8sClient:      k8sClient,
		namespace:      namespace,
		vmBuilder:      NewVMBuilder(namespace),
		statusMapper:   NewStatusMapper(),
		networkHandler: NewNetworkHandler(),
	}, nil
}

// CreateVM creates a new virtual machine
func (c *Client) CreateVM(ctx context.Context, req models.CreateVMParams) (*models.InternalVM, error) {
	// Validate the request
	if err := c.validateCreateRequest(req); err != nil {
		return nil, NewValidationError("request", "", err.Error())
	}

	// Generate unique ID for the VM
	vmID := uuid.New().String()

	// Build VM specification
	vmSpec := c.vmBuilder.BuildVMSpecFromInternal(req, vmID)

	// Create the VM in Kubernetes
	createdVM, err := c.dynamicClient.Resource(VirtualMachineGVR).
		Namespace(c.namespace).
		Create(ctx, vmSpec, metav1.CreateOptions{})
	if err != nil {
		return nil, NewKubernetesError("create", "virtualmachine", c.namespace, req.Name, err)
	}

	// Convert to internal format and return
	return c.convertVMToInternal(createdVM, nil), nil
}

// GetVM retrieves a virtual machine by ID
func (c *Client) GetVM(ctx context.Context, serverID string) (*models.InternalVM, error) {
	if err := validateServerID(serverID); err != nil {
		return nil, NewValidationError("serverID", serverID, err.Error())
	}

	vm, err := c.getVMByID(ctx, serverID)
	if err != nil {
		return nil, NewVMError("get", "", serverID, err)
	}

	// Get VMI if VM is running
	var vmi *unstructured.Unstructured
	if c.isVMRunning(vm) {
		vmi, _ = c.getVMI(ctx, vm.GetName())
	}

	return c.convertVMToInternal(vm, vmi), nil
}

// ListVMs lists all virtual machines managed by kvirtos2
func (c *Client) ListVMs(ctx context.Context) ([]models.InternalVM, error) {
	vmList, err := c.dynamicClient.Resource(VirtualMachineGVR).
		Namespace(c.namespace).
		List(ctx, metav1.ListOptions{
			LabelSelector: fmt.Sprintf("%s=%s", AppLabel, AppLabelValue),
		})
	if err != nil {
		return nil, NewKubernetesError("list", "virtualmachines", c.namespace, "", err)
	}

	vms := make([]models.InternalVM, 0, len(vmList.Items))
	for _, vm := range vmList.Items {
		var vmi *unstructured.Unstructured
		if c.isVMRunning(&vm) {
			vmi, _ = c.getVMI(ctx, vm.GetName())
		}
		vms = append(vms, *c.convertVMToInternal(&vm, vmi))
	}

	return vms, nil
}

// DeleteVM deletes a virtual machine
func (c *Client) DeleteVM(ctx context.Context, serverID string) error {
	if err := validateServerID(serverID); err != nil {
		return NewValidationError("serverID", serverID, err.Error())
	}

	vm, err := c.getVMByID(ctx, serverID)
	if err != nil {
		return NewVMError("delete", "", serverID, err)
	}

	err = c.dynamicClient.Resource(VirtualMachineGVR).
		Namespace(c.namespace).
		Delete(ctx, vm.GetName(), metav1.DeleteOptions{})
	if err != nil {
		return NewKubernetesError("delete", "virtualmachine", c.namespace, vm.GetName(), err)
	}

	return nil
}

// StartVM starts a virtual machine
func (c *Client) StartVM(ctx context.Context, serverID string) error {
	if err := validateServerID(serverID); err != nil {
		return NewValidationError("serverID", serverID, err.Error())
	}

	vm, err := c.getVMByID(ctx, serverID)
	if err != nil {
		return NewVMError("start", "", serverID, err)
	}

	// Check if already running
	if c.isVMRunning(vm) {
		return nil // Already running, no-op
	}

	// Update VM to set running=true
	updatedVM := c.vmBuilder.BuildVMUpdateSpec(vm, true)
	_, err = c.dynamicClient.Resource(VirtualMachineGVR).
		Namespace(c.namespace).
		Update(ctx, updatedVM, metav1.UpdateOptions{})
	if err != nil {
		return NewKubernetesError("update", "virtualmachine", c.namespace, vm.GetName(), err)
	}

	return nil
}

// StopVM stops a virtual machine
func (c *Client) StopVM(ctx context.Context, serverID string) error {
	if err := validateServerID(serverID); err != nil {
		return NewValidationError("serverID", serverID, err.Error())
	}

	vm, err := c.getVMByID(ctx, serverID)
	if err != nil {
		return NewVMError("stop", "", serverID, err)
	}

	// Check if already stopped
	if !c.isVMRunning(vm) {
		return nil // Already stopped, no-op
	}

	// Update VM to set running=false
	updatedVM := c.vmBuilder.BuildVMUpdateSpec(vm, false)
	_, err = c.dynamicClient.Resource(VirtualMachineGVR).
		Namespace(c.namespace).
		Update(ctx, updatedVM, metav1.UpdateOptions{})
	if err != nil {
		return NewKubernetesError("update", "virtualmachine", c.namespace, vm.GetName(), err)
	}

	return nil
}

// Helper methods

// buildKubernetesConfig creates a Kubernetes REST config
func buildKubernetesConfig(kubeconfig string) (*rest.Config, error) {
	if kubeconfig != "" {
		return clientcmd.BuildConfigFromFlags("", kubeconfig)
	}
	return rest.InClusterConfig()
}

// createClients creates the dynamic and standard Kubernetes clients
func createClients(config *rest.Config) (dynamic.Interface, kubernetes.Interface, error) {
	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create dynamic client: %w", err)
	}

	k8sClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create kubernetes client: %w", err)
	}

	return dynamicClient, k8sClient, nil
}

// getVMByID finds a VM by its server ID (vm-id label)
func (c *Client) getVMByID(ctx context.Context, serverID string) (*unstructured.Unstructured, error) {
	vmList, err := c.dynamicClient.Resource(VirtualMachineGVR).
		Namespace(c.namespace).
		List(ctx, metav1.ListOptions{
			LabelSelector: fmt.Sprintf("%s=%s,%s=%s", AppLabel, AppLabelValue, VMIDLabel, serverID),
		})
	if err != nil {
		return nil, fmt.Errorf("failed to search for virtual machine: %w", err)
	}

	if len(vmList.Items) == 0 {
		return nil, ErrVMNotFound
	}

	return &vmList.Items[0], nil
}

// getVMI retrieves a VirtualMachineInstance by name
func (c *Client) getVMI(ctx context.Context, vmiName string) (*unstructured.Unstructured, error) {
	vmi, err := c.dynamicClient.Resource(VirtualMachineInstanceGVR).
		Namespace(c.namespace).
		Get(ctx, vmiName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	return vmi, nil
}

// isVMRunning checks if a VM is currently running
func (c *Client) isVMRunning(vm *unstructured.Unstructured) bool {
	running, _, _ := unstructured.NestedBool(vm.Object, "spec", "running")
	return running
}

// convertVMToInternal converts a KubeVirt VM to internal VM format
func (c *Client) convertVMToInternal(vm *unstructured.Unstructured, vmi *unstructured.Unstructured) *models.InternalVM {
	labels := vm.GetLabels()
	vmID := labels[VMIDLabel]
	if vmID == "" {
		vmID = string(vm.GetUID())
	}

	// Use status mapper to determine states
	status := c.statusMapper.GetInternalVMStatus(vm, vmi)

	// Extract metadata from annotations
	metadata := c.extractMetadata(vm.GetAnnotations())

	// Extract network addresses
	addresses := c.networkHandler.ExtractInternalAddresses(vmi)

	// Extract node information
	node := c.extractNodeName(vmi)

	// Extract resources
	resources := c.extractInternalResources(labels)

	return &models.InternalVM{
		ID:        vmID,
		Name:      vm.GetName(),
		Status:    status,
		Created:   vm.GetCreationTimestamp().Time,
		Updated:   vm.GetCreationTimestamp().Time,
		Image:     labels[ImageRefLabel],
		Flavor:    labels[FlavorRefLabel],
		Metadata:  metadata,
		Addresses: addresses,
		Node:      node,
		Resources: resources,
	}
}

// extractMetadata extracts user metadata from VM annotations
func (c *Client) extractMetadata(annotations map[string]string) map[string]string {
	metadata := make(map[string]string)
	if annotations == nil {
		return metadata
	}

	for k, v := range annotations {
		if strings.HasPrefix(k, MetadataAnnotationPrefix) {
			key := strings.TrimPrefix(k, MetadataAnnotationPrefix)
			metadata[key] = v
		}
	}
	return metadata
}

// validateCreateRequest validates the VM creation request
func (c *Client) validateCreateRequest(req models.CreateVMParams) error {
	if err := validateVMName(req.Name); err != nil {
		return err
	}

	if req.Image == "" {
		return fmt.Errorf("image is required")
	}

	if req.Flavor == "" {
		return fmt.Errorf("flavor is required")
	}

	if !IsValidImage(req.Image) {
		return fmt.Errorf("unknown image: %s", req.Image)
	}

	if !IsValidFlavor(req.Flavor) {
		return fmt.Errorf("unknown flavor: %s", req.Flavor)
	}

	return nil
}

// extractNodeName extracts the node name from VMI
func (c *Client) extractNodeName(vmi *unstructured.Unstructured) string {
	if vmi == nil {
		return ""
	}
	
	nodeName, _, _ := unstructured.NestedString(vmi.Object, "status", "nodeName")
	return nodeName
}

// extractInternalResources extracts resource information
func (c *Client) extractInternalResources(labels map[string]string) *models.InternalResources {
	flavorRef := labels[FlavorRefLabel]
	if flavorRef == "" {
		return nil
	}
	
	flavorConfig := GetFlavorConfig(flavorRef)
	return &models.InternalResources{
		CPU:    fmt.Sprintf("%dm", flavorConfig.CPU),
		Memory: fmt.Sprintf("%dMi", flavorConfig.Memory),
	}
}