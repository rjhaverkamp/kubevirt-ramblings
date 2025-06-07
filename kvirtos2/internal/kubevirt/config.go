package kubevirt

// FlavorConfig defines CPU and memory resources for VM flavors
type FlavorConfig struct {
	CPU    int // CPU in millicores (1000m = 1 CPU)
	Memory int // Memory in MiB
}

// ImageConfig defines container image configuration
type ImageConfig struct {
	ContainerImage string
	Description    string
}

// Flavors defines the available VM flavors
var Flavors = map[string]FlavorConfig{
	"m1.tiny": {
		CPU:    500, // 0.5 CPU
		Memory: 512, // 512MB RAM
	},
	"m1.small": {
		CPU:    1000, // 1 CPU
		Memory: 2048, // 2GB RAM
	},
	"m1.medium": {
		CPU:    2000, // 2 CPU
		Memory: 4096, // 4GB RAM
	},
	"m1.large": {
		CPU:    4000, // 4 CPU
		Memory: 8192, // 8GB RAM
	},
}

// Images defines the available VM images
var Images = map[string]ImageConfig{
	"ubuntu-24": {
		ContainerImage: "ghcr.io/rjhaverkamp/ubuntu-2404:latest",
		Description:    "Ubuntu 24 Cloud Image",
	},
	"ubuntu-20.04": {
		ContainerImage: "quay.io/kubevirt/ubuntu-cloud-container-disk-demo",
		Description:    "Ubuntu 20.04 Cloud Image",
	},
	"fedora-cloud": {
		ContainerImage: "quay.io/kubevirt/fedora-cloud-container-disk-demo",
		Description:    "Fedora Cloud Image",
	},
	"cirros": {
		ContainerImage: "quay.io/kubevirt/cirros-container-disk-demo",
		Description:    "CirrOS Test Image",
	},
}

// GetFlavorConfig returns the flavor configuration for the given flavor reference
// Returns default configuration if flavor is not found
func GetFlavorConfig(flavorRef string) FlavorConfig {
	if flavor, exists := Flavors[flavorRef]; exists {
		return flavor
	}
	// Return default configuration
	return FlavorConfig{CPU: 1000, Memory: 1024}
}

// GetImageConfig returns the image configuration for the given image reference
// Returns default configuration if image is not found
func GetImageConfig(imageRef string) ImageConfig {
	if image, exists := Images[imageRef]; exists {
		return image
	}
	// Return default configuration
	return Images["fedora-cloud"]
}

// ListFlavors returns all available flavors
func ListFlavors() map[string]FlavorConfig {
	return Flavors
}

// ListImages returns all available images
func ListImages() map[string]ImageConfig {
	return Images
}

// IsValidFlavor checks if the given flavor reference is valid
func IsValidFlavor(flavorRef string) bool {
	_, exists := Flavors[flavorRef]
	return exists
}

// IsValidImage checks if the given image reference is valid
func IsValidImage(imageRef string) bool {
	_, exists := Images[imageRef]
	return exists
}
