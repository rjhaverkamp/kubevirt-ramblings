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

// BuildVMSpec builds a complete VirtualMachine specification from a creation request
func (b *VMBuilder) BuildVMSpec(req models.CreateServerParams, vmID string) *unstructured.Unstructured {
	flavorConfig := GetFlavorConfig(req.FlavorRef)
	imageConfig := GetImageConfig(req.ImageRef)
	
	return &unstructured.Unstructured{
		Object: map[string]interface{}{
			"apiVersion": KubeVirtAPIVersion,
			"kind":       VMKind,
			"metadata":   b.buildMetadata(req, vmID),
			"spec":       b.buildSpec(req, vmID, flavorConfig, imageConfig),
		},
	}
}

// buildMetadata constructs the metadata section of the VM
func (b *VMBuilder) buildMetadata(req models.CreateServerParams, vmID string) map[string]interface{} {
	metadata := map[string]interface{}{
		"name":      req.Name,
		"namespace": b.namespace,
		"labels":    b.buildLabels(req, vmID),
	}
	
	// Add annotations for metadata if provided
	if req.Metadata != nil && len(req.Metadata) > 0 {
		metadata["annotations"] = b.buildAnnotations(req.Metadata)
	}
	
	return metadata
}

// buildLabels constructs the labels for the VM
func (b *VMBuilder) buildLabels(req models.CreateServerParams, vmID string) map[string]interface{} {
	return map[string]interface{}{
		AppLabel:       AppLabelValue,
		VMIDLabel:      vmID,
		ImageRefLabel:  req.ImageRef,
		FlavorRefLabel: req.FlavorRef,
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

// buildSpec constructs the spec section of the VM
func (b *VMBuilder) buildSpec(req models.CreateServerParams, vmID string, flavorConfig FlavorConfig, imageConfig ImageConfig) map[string]interface{} {
	return map[string]interface{}{
		"running":  false, // Always start VMs in stopped state
		"template": b.buildTemplate(req, vmID, flavorConfig, imageConfig),
	}
}

// buildTemplate constructs the template section of the VM spec
func (b *VMBuilder) buildTemplate(req models.CreateServerParams, vmID string, flavorConfig FlavorConfig, imageConfig ImageConfig) map[string]interface{} {
	return map[string]interface{}{
		"metadata": b.buildTemplateMetadata(vmID),
		"spec":     b.buildVMISpec(req, flavorConfig, imageConfig),
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

// buildVMISpec constructs the VirtualMachineInstance specification
func (b *VMBuilder) buildVMISpec(req models.CreateServerParams, flavorConfig FlavorConfig, imageConfig ImageConfig) map[string]interface{} {
	return map[string]interface{}{
		"domain":   b.buildDomainSpec(flavorConfig, req.Networks),
		"networks": b.buildNetworksSpec(req.Networks),
		"volumes":  b.buildVolumesSpec(imageConfig),
	}
}

// buildDomainSpec constructs the domain specification
func (b *VMBuilder) buildDomainSpec(flavorConfig FlavorConfig, networks []models.Network) map[string]interface{} {
	return map[string]interface{}{
		"resources": b.buildResourcesSpec(flavorConfig),
		"devices":   b.buildDevicesSpec(networks),
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

// buildDevicesSpec constructs the devices specification
func (b *VMBuilder) buildDevicesSpec(networks []models.Network) map[string]interface{} {
	return map[string]interface{}{
		"disks":      b.buildDisksSpec(),
		"interfaces": b.buildInterfacesSpec(networks),
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

// buildInterfacesSpec constructs the interfaces specification
func (b *VMBuilder) buildInterfacesSpec(networks []models.Network) []map[string]interface{} {
	return b.networkHandler.BuildInterfacesForVM(networks)
}

// buildNetworksSpec constructs the networks specification
func (b *VMBuilder) buildNetworksSpec(networks []models.Network) []map[string]interface{} {
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

// ValidateVMSpec validates a VM specification before creation
func (b *VMBuilder) ValidateVMSpec(req models.CreateServerParams) error {
	// Validate flavor
	if !IsValidFlavor(req.FlavorRef) {
		return NewValidationError("flavorRef", req.FlavorRef, "unknown flavor")
	}
	
	// Validate image
	if !IsValidImage(req.ImageRef) {
		return NewValidationError("imageRef", req.ImageRef, "unknown image")
	}
	
	// Validate networks if specified
	if len(req.Networks) > 0 {
		if err := b.networkHandler.ValidateNetworkConfiguration(req.Networks); err != nil {
			return NewValidationError("networks", "", err.Error())
		}
	}
	
	return nil
}

// GetVMSpecSummary returns a summary of the VM specification
func (b *VMBuilder) GetVMSpecSummary(req models.CreateServerParams) *VMSpecSummary {
	flavorConfig := GetFlavorConfig(req.FlavorRef)
	imageConfig := GetImageConfig(req.ImageRef)
	
	return &VMSpecSummary{
		Name:        req.Name,
		Flavor:      req.FlavorRef,
		Image:       req.ImageRef,
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