package kubevirt

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/kubevirt/kvirtos2/internal/models"
)

// Client wraps KubeVirt and Kubernetes clients
type Client struct {
	dynamicClient dynamic.Interface
	k8sClient     kubernetes.Interface
	namespace     string
}

var (
	// KubeVirt GVRs
	vmGVR = schema.GroupVersionResource{
		Group:    "kubevirt.io",
		Version:  "v1",
		Resource: "virtualmachines",
	}
	vmiGVR = schema.GroupVersionResource{
		Group:    "kubevirt.io",
		Version:  "v1",
		Resource: "virtualmachineinstances",
	}
)

// NewClient creates a new KubeVirt client
func NewClient(kubeconfig, namespace string) (*Client, error) {
	var config *rest.Config
	var err error

	if kubeconfig != "" {
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	} else {
		config, err = rest.InClusterConfig()
	}
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes config: %v", err)
	}

	dynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create dynamic client: %v", err)
	}

	k8sClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("failed to create kubernetes client: %v", err)
	}

	if namespace == "" {
		namespace = "default"
	}

	return &Client{
		dynamicClient: dynamicClient,
		k8sClient:     k8sClient,
		namespace:     namespace,
	}, nil
}

// CreateVM creates a new virtual machine
func (c *Client) CreateVM(ctx context.Context, req models.CreateServerParams) (*models.Server, error) {
	vmName := req.Name
	if vmName == "" {
		return nil, fmt.Errorf("VM name is required")
	}

	// Generate UUID for the VM
	vmID := uuid.New().String()

	// Parse flavor (basic CPU/Memory allocation)
	cpu, memory := c.parseFlavorRef(req.FlavorRef)

	// Create VirtualMachine resource as unstructured
	vm := &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": "kubevirt.io/v1",
			"kind":       "VirtualMachine",
			"metadata": map[string]interface{}{
				"name":      vmName,
				"namespace": c.namespace,
				"labels": map[string]interface{}{
					"app":        "kvirtos2",
					"vm-id":      vmID,
					"image-ref":  req.ImageRef,
					"flavor-ref": req.FlavorRef,
				},
			},
			"spec": map[string]interface{}{
				"running": false,
				"template": map[string]interface{}{
					"metadata": map[string]interface{}{
						"labels": map[string]interface{}{
							"app":   "kvirtos2",
							"vm-id": vmID,
						},
					},
					"spec": map[string]interface{}{
						"domain": map[string]interface{}{
							"resources": map[string]interface{}{
								"requests": map[string]interface{}{
									"cpu":    fmt.Sprintf("%dm", cpu),
									"memory": fmt.Sprintf("%dMi", memory),
								},
							},
							"devices": map[string]interface{}{
								"disks": []map[string]interface{}{
									{
										"name": "disk0",
										"disk": map[string]interface{}{
											"bus": "virtio",
										},
									},
								},
								"interfaces": []map[string]interface{}{
									{
										"name":      "default",
										"masquerade": map[string]interface{}{},
									},
								},
							},
						},
						"networks": []map[string]interface{}{
							{
								"name": "default",
								"pod":  map[string]interface{}{},
							},
						},
						"volumes": []map[string]interface{}{
							{
								"name": "disk0",
								"containerDisk": map[string]interface{}{
									"image": c.parseImageRef(req.ImageRef),
								},
							},
						},
					},
				},
			},
		},
	}

	// Add metadata if provided
	if req.Metadata != nil {
		annotations := make(map[string]interface{})
		for k, v := range req.Metadata {
			annotations[fmt.Sprintf("kvirtos2.io/metadata.%s", k)] = v
		}
		vm.Object["metadata"].(map[string]interface{})["annotations"] = annotations
	}

	// Create the VM
	createdVM, err := c.dynamicClient.Resource(vmGVR).Namespace(c.namespace).Create(ctx, vm, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create virtual machine: %v", err)
	}

	return c.convertVMToServer(createdVM, nil), nil
}

// GetVM retrieves a virtual machine by ID
func (c *Client) GetVM(ctx context.Context, serverID string) (*models.Server, error) {
	vm, err := c.getVMByID(ctx, serverID)
	if err != nil {
		return nil, err
	}

	var vmi *unstructured.Unstructured
	running, _, _ := unstructured.NestedBool(vm.Object, "spec", "running")
	if running {
		vmi, _ = c.dynamicClient.Resource(vmiGVR).Namespace(c.namespace).Get(ctx, vm.GetName(), metav1.GetOptions{})
	}

	return c.convertVMToServer(vm, vmi), nil
}

// ListVMs lists all virtual machines
func (c *Client) ListVMs(ctx context.Context) ([]models.Server, error) {
	vmList, err := c.dynamicClient.Resource(vmGVR).Namespace(c.namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "app=kvirtos2",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list virtual machines: %v", err)
	}

	servers := make([]models.Server, 0, len(vmList.Items))
	for _, vm := range vmList.Items {
		var vmi *unstructured.Unstructured
		running, _, _ := unstructured.NestedBool(vm.Object, "spec", "running")
		if running {
			vmi, _ = c.dynamicClient.Resource(vmiGVR).Namespace(c.namespace).Get(ctx, vm.GetName(), metav1.GetOptions{})
		}
		servers = append(servers, *c.convertVMToServer(&vm, vmi))
	}

	return servers, nil
}

// DeleteVM deletes a virtual machine
func (c *Client) DeleteVM(ctx context.Context, serverID string) error {
	vm, err := c.getVMByID(ctx, serverID)
	if err != nil {
		return err
	}

	err = c.dynamicClient.Resource(vmGVR).Namespace(c.namespace).Delete(ctx, vm.GetName(), metav1.DeleteOptions{})
	if err != nil {
		return fmt.Errorf("failed to delete virtual machine: %v", err)
	}

	return nil
}

// StartVM starts a virtual machine
func (c *Client) StartVM(ctx context.Context, serverID string) error {
	vm, err := c.getVMByID(ctx, serverID)
	if err != nil {
		return err
	}

	running, _, _ := unstructured.NestedBool(vm.Object, "spec", "running")
	if running {
		return nil // Already running
	}

	unstructured.SetNestedField(vm.Object, true, "spec", "running")
	_, err = c.dynamicClient.Resource(vmGVR).Namespace(c.namespace).Update(ctx, vm, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to start virtual machine: %v", err)
	}

	return nil
}

// StopVM stops a virtual machine
func (c *Client) StopVM(ctx context.Context, serverID string) error {
	vm, err := c.getVMByID(ctx, serverID)
	if err != nil {
		return err
	}

	running, _, _ := unstructured.NestedBool(vm.Object, "spec", "running")
	if !running {
		return nil // Already stopped
	}

	unstructured.SetNestedField(vm.Object, false, "spec", "running")
	_, err = c.dynamicClient.Resource(vmGVR).Namespace(c.namespace).Update(ctx, vm, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to stop virtual machine: %v", err)
	}

	return nil
}

// Helper methods

func (c *Client) getVMByID(ctx context.Context, serverID string) (*unstructured.Unstructured, error) {
	vmList, err := c.dynamicClient.Resource(vmGVR).Namespace(c.namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app=kvirtos2,vm-id=%s", serverID),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to find virtual machine: %v", err)
	}

	if len(vmList.Items) == 0 {
		return nil, fmt.Errorf("virtual machine not found")
	}

	return &vmList.Items[0], nil
}

func (c *Client) convertVMToServer(vm *unstructured.Unstructured, vmi *unstructured.Unstructured) *models.Server {
	labels := vm.GetLabels()
	vmID := labels["vm-id"]
	if vmID == "" {
		vmID = string(vm.GetUID())
	}

	status := c.getVMStatus(vm, vmi)
	powerState := c.getPowerState(vm, vmi)
	vmState := c.getVMState(status)

	// Extract metadata
	metadata := make(map[string]string)
	annotations := vm.GetAnnotations()
	for k, v := range annotations {
		if strings.HasPrefix(k, "kvirtos2.io/metadata.") {
			key := strings.TrimPrefix(k, "kvirtos2.io/metadata.")
			metadata[key] = v
		}
	}

	// Get addresses
	addresses := make(map[string][]models.Address)
	if vmi != nil {
		interfaces, found, _ := unstructured.NestedSlice(vmi.Object, "status", "interfaces")
		if found {
			defaultAddresses := make([]models.Address, 0)
			for _, iface := range interfaces {
				ifaceMap, ok := iface.(map[string]interface{})
				if !ok {
					continue
				}
				if ip, found := ifaceMap["ipAddress"].(string); found && ip != "" {
					defaultAddresses = append(defaultAddresses, models.Address{
						Version: 4,
						Addr:    ip,
						Type:    "fixed",
					})
				}
			}
			if len(defaultAddresses) > 0 {
				addresses["default"] = defaultAddresses
			}
		}
	}

	return &models.Server{
		ID:         vmID,
		Name:       vm.GetName(),
		Status:     status,
		PowerState: powerState,
		VMState:    vmState,
		Created:    vm.GetCreationTimestamp().Time,
		Updated:    vm.GetCreationTimestamp().Time,
		Flavor: models.Flavor{
			ID: labels["flavor-ref"],
		},
		Image: models.Image{
			ID: labels["image-ref"],
		},
		Metadata:  metadata,
		Addresses: addresses,
		TenantID:  c.namespace,
		UserID:    "kvirtos2",
		HostID:    "",
		Progress:  100,
	}
}

func (c *Client) getVMStatus(vm *unstructured.Unstructured, vmi *unstructured.Unstructured) string {
	running, _, _ := unstructured.NestedBool(vm.Object, "spec", "running")
	if !running {
		return models.StatusShutoff
	}

	if vmi == nil {
		return models.StatusBuild
	}

	phase, found, _ := unstructured.NestedString(vmi.Object, "status", "phase")
	if !found {
		return models.StatusBuild
	}

	switch phase {
	case "Running":
		return models.StatusActive
	case "Pending", "Scheduling", "Scheduled":
		return models.StatusBuild
	case "Failed":
		return models.StatusError
	case "Succeeded":
		return models.StatusShutoff
	default:
		return models.StatusUnknown
	}
}

func (c *Client) getPowerState(vm *unstructured.Unstructured, vmi *unstructured.Unstructured) int {
	running, _, _ := unstructured.NestedBool(vm.Object, "spec", "running")
	if !running {
		return models.PowerStateShutdown
	}

	if vmi == nil {
		return models.PowerStateNoState
	}

	phase, found, _ := unstructured.NestedString(vmi.Object, "status", "phase")
	if !found {
		return models.PowerStateNoState
	}

	switch phase {
	case "Running":
		return models.PowerStateRunning
	case "Failed":
		return models.PowerStateCrashed
	default:
		return models.PowerStateNoState
	}
}

func (c *Client) getVMState(status string) string {
	switch status {
	case models.StatusActive:
		return models.VMStateActive
	case models.StatusBuild:
		return models.VMStateBuilding
	case models.StatusShutoff:
		return models.VMStateStopped
	case models.StatusError:
		return models.VMStateError
	case models.StatusDeleted:
		return models.VMStateDeleted
	default:
		return models.VMStateActive
	}
}

func (c *Client) parseFlavorRef(flavorRef string) (cpu int, memory int) {
	// Simple flavor parsing - in a real implementation, this would
	// look up flavor definitions from a database or config
	switch flavorRef {
	case "m1.tiny":
		return 500, 512 // 0.5 CPU, 512MB RAM
	case "m1.small":
		return 1000, 2048 // 1 CPU, 2GB RAM
	case "m1.medium":
		return 2000, 4096 // 2 CPU, 4GB RAM
	case "m1.large":
		return 4000, 8192 // 4 CPU, 8GB RAM
	default:
		return 1000, 1024 // Default: 1 CPU, 1GB RAM
	}
}

func (c *Client) parseImageRef(imageRef string) string {
	// Simple image parsing - in a real implementation, this would
	// map image IDs to container images
	switch imageRef {
	case "ubuntu-20.04":
		return "quay.io/kubevirt/fedora-cloud-container-disk-demo"
	case "fedora-cloud":
		return "quay.io/kubevirt/fedora-cloud-container-disk-demo"
	case "cirros":
		return "quay.io/kubevirt/cirros-container-disk-demo"
	default:
		// Default to a working container disk image
		return "quay.io/kubevirt/fedora-cloud-container-disk-demo"
	}
}