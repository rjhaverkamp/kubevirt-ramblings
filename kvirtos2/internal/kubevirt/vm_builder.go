package kubevirt

import (
	"fmt"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"github.com/kubevirt/kvirtos2/internal/models"
)

// VMBuilder handles the construction of VirtualMachine specifications
type VMBuilder struct {
	namespace      string
	networkHandler *NetworkHandler
}

// NewVMBuilder creates a new VMBuilder instance
func NewVMBuilder(namespace string) *VMBuilder {
	return &VMBuilder{
		namespace:      namespace,
		networkHandler: NewNetworkHandler(),
	}
}

// BuildVMSpecFromInternal builds a complete VirtualMachine specification from an internal creation request
func (b *VMBuilder) BuildVMSpecFromInternal(req models.CreateVMParams, vmID string) *unstructured.Unstructured {
	flavorConfig := GetFlavorConfig(req.Flavor)
	imageConfig := GetImageConfig(req.Image)
	
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": KubeVirtAPIVersion,
			"kind":       VMKind,
			"metadata":   b.buildMetadataFromInternal(req, vmID),
			"spec":       b.buildSpecFromInternal(req, vmID, flavorConfig, imageConfig),
		},
	}
}

// buildMetadataFromInternal constructs the metadata section of the VM from internal request
func (b *VMBuilder) buildMetadataFromInternal(req models.CreateVMParams, vmID string) map[string]interface{} {
	metadata := map[string]interface{}{
		"name":      req.Name,
		"namespace": b.namespace,
		"labels":    b.buildLabelsFromInternal(req, vmID),
	}
	
	// Add annotations for metadata if provided
	if req.Metadata != nil && len(req.Metadata) > 0 {
		metadata["annotations"] = b.buildAnnotations(req.Metadata)
	}
	
	return metadata
}

// buildLabelsFromInternal constructs the labels for the VM from internal request
func (b *VMBuilder) buildLabelsFromInternal(req models.CreateVMParams, vmID string) map[string]interface{} {
	return map[string]interface{}{
		AppLabel:       AppLabelValue,
		VMIDLabel:      vmID,
		ImageRefLabel:  req.Image,
		FlavorRefLabel: req.Flavor,
	}
}

// buildAnnotations constructs annotations from request metadata
func (b *VMBuilder) buildAnnotations(metadata map[string]string) map[string]interface{} {
	annotations := make(map[string]interface{})
	for k, v := range metadata {
		annotations[fmt.Sprintf("%s%s", MetadataAnnotationPrefix, k)] = v
	}
	return annotations
}

// buildSpecFromInternal constructs the spec section of the VM from internal request
func (b *VMBuilder) buildSpecFromInternal(req models.CreateVMParams, vmID string, flavorConfig FlavorConfig, imageConfig ImageConfig) map[string]interface{} {
	return map[string]interface{}{
		"running":  req.AutoStart, // Use autoStart from request
		"template": b.buildTemplateFromInternal(req, vmID, flavorConfig, imageConfig),
	}
}

// buildTemplateFromInternal constructs the template section of the VM spec from internal request
func (b *VMBuilder) buildTemplateFromInternal(req models.CreateVMParams, vmID string, flavorConfig FlavorConfig, imageConfig ImageConfig) map[string]interface{} {
	return map[string]interface{}{
		"metadata": b.buildTemplateMetadata(vmID),
		"spec":     b.buildVMISpecFromInternal(req, flavorConfig, imageConfig),
	}
}

// buildTemplateMetadata constructs the metadata for the VMI template
func (b *VMBuilder) buildTemplateMetadata(vmID string) map[string]interface{} {
	return map[string]interface{}{
		"labels": map[string]interface{}{
			AppLabel:  AppLabelValue,
			VMIDLabel: vmID,
		},
	}
}

// buildVMISpecFromInternal constructs the VirtualMachineInstance specification from internal request
func (b *VMBuilder) buildVMISpecFromInternal(req models.CreateVMParams, flavorConfig FlavorConfig, imageConfig ImageConfig) map[string]interface{} {
	return map[string]interface{}{
		"domain":   b.buildDomainSpecFromInternal(flavorConfig, req.Networks),
		"networks": b.buildNetworksSpecFromInternal(req.Networks),
		"volumes":  b.buildVolumesSpec(imageConfig),
	}
}

// buildDomainSpecFromInternal constructs the domain specification from internal request
func (b *VMBuilder) buildDomainSpecFromInternal(flavorConfig FlavorConfig, networks []models.InternalNetwork) map[string]interface{} {
	return map[string]interface{}{
		"resources": b.buildResourcesSpec(flavorConfig),
		"devices":   b.buildDevicesSpecFromInternal(networks),
	}
}

// buildResourcesSpec constructs the resources specification
func (b *VMBuilder) buildResourcesSpec(flavorConfig FlavorConfig) map[string]interface{} {
	return map[string]interface{}{
		"requests": map[string]interface{}{
			"cpu":    fmt.Sprintf("%dm", flavorConfig.CPU),
			"memory": fmt.Sprintf("%dMi", flavorConfig.Memory),
		},
	}
}

// buildDevicesSpecFromInternal constructs the devices specification from internal request
func (b *VMBuilder) buildDevicesSpecFromInternal(networks []models.InternalNetwork) map[string]interface{} {
	return map[string]interface{}{
		"disks":      b.buildDisksSpec(),
		"interfaces": b.buildInterfacesSpecFromInternal(networks),
	}
}

// buildDisksSpec constructs the disks specification
func (b *VMBuilder) buildDisksSpec() []map[string]interface{} {
	return []map[string]interface{}{
		{
			"name": DefaultDiskName,
			"disk": map[string]interface{}{
				"bus": DefaultDiskBus,
			},
		},
	}
}

// buildInterfacesSpecFromInternal constructs the interfaces specification from internal request
func (b *VMBuilder) buildInterfacesSpecFromInternal(networks []models.InternalNetwork) []map[string]interface{} {
	return b.networkHandler.BuildInterfacesForVM(networks)
}

// buildNetworksSpecFromInternal constructs the networks specification from internal request
func (b *VMBuilder) buildNetworksSpecFromInternal(networks []models.InternalNetwork) []map[string]interface{} {
	return b.networkHandler.BuildNetworksForVM(networks)
}

// buildVolumesSpec constructs the volumes specification
func (b *VMBuilder) buildVolumesSpec(imageConfig ImageConfig) []map[string]interface{} {
	return []map[string]interface{}{
		{
			"name": DefaultDiskName,
			"containerDisk": map[string]interface{}{
				"image": imageConfig.ContainerImage,
			},
		},
	}
}

// BuildVMUpdateSpec builds a specification for updating VM running state
func (b *VMBuilder) BuildVMUpdateSpec(vm *unstructured.Unstructured, running bool) *unstructured.Unstructured {
	// Create a copy of the VM
	updated := vm.DeepCopy()
	
	// Update the running field
	err := unstructured.SetNestedField(updated.Object, running, "spec", "running")
	if err != nil {
		// If we can't set the field, return the original VM
		// The caller should handle this error case
		return vm
	}
	
	return updated
}

// ValidateVMSpecFromInternal validates a VM specification from internal request before creation
func (b *VMBuilder) ValidateVMSpecFromInternal(req models.CreateVMParams) error {
	// Validate flavor
	if !IsValidFlavor(req.Flavor) {
		return NewValidationError("flavor", req.Flavor, "unknown flavor")
	}
	
	// Validate image
	if !IsValidImage(req.Image) {
		return NewValidationError("image", req.Image, "unknown image")
	}
	
	// Validate networks if specified
	if len(req.Networks) > 0 {
		if err := b.networkHandler.ValidateNetworkConfiguration(req.Networks); err != nil {
			return NewValidationError("networks", "", err.Error())
		}
	}
	
	return nil
}

// GetVMSpecSummaryFromInternal returns a summary of the VM specification from internal request
func (b *VMBuilder) GetVMSpecSummaryFromInternal(req models.CreateVMParams) *VMSpecSummary {
	flavorConfig := GetFlavorConfig(req.Flavor)
	imageConfig := GetImageConfig(req.Image)
	
	return &VMSpecSummary{
		Name:        req.Name,
		Flavor:      req.Flavor,
		Image:       req.Image,
		CPU:         flavorConfig.CPU,
		Memory:      flavorConfig.Memory,
		ImagePath:   imageConfig.ContainerImage,
		Networks:    len(req.Networks) + 1, // +1 for default network
		HasMetadata: len(req.Metadata) > 0,
	}
}

// VMSpecSummary provides a summary of VM specification
type VMSpecSummary struct {
	Name        string `json:"name"`
	Flavor      string `json:"flavor"`
	Image       string `json:"image"`
	CPU         int    `json:"cpu"`
	Memory      int    `json:"memory"`
	ImagePath   string `json:"imagePath"`
	Networks    int    `json:"networks"`
	HasMetadata bool   `json:"hasMetadata"`
}