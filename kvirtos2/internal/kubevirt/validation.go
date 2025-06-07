package kubevirt

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/kubevirt/kvirtos2/internal/models"
)

var (
	// nameRegex defines valid VM name pattern (DNS-1123 compliant)
	nameRegex = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)
)

// validateCreateRequest validates the VM creation request
func validateCreateRequest(req models.CreateVMParams) error {
	if err := validateVMName(req.Name); err != nil {
		return err
	}

	if err := validateImageRef(req.Image); err != nil {
		return err
	}

	if err := validateFlavorRef(req.Flavor); err != nil {
		return err
	}

	if err := validateMetadata(req.Metadata); err != nil {
		return err
	}

	return nil
}

// validateVMName validates the VM name
func validateVMName(name string) error {
	if name == "" {
		return fmt.Errorf("VM name is required")
	}

	if len(name) > 63 {
		return fmt.Errorf("VM name must be 63 characters or less")
	}

	if !nameRegex.MatchString(name) {
		return fmt.Errorf("VM name must be a valid DNS-1123 label: %s", name)
	}

	return nil
}

// validateImageRef validates the image reference
func validateImageRef(imageRef string) error {
	if imageRef == "" {
		return fmt.Errorf("image reference is required")
	}

	if !IsValidImage(imageRef) {
		availableImages := make([]string, 0, len(Images))
		for img := range Images {
			availableImages = append(availableImages, img)
		}
		return fmt.Errorf("unknown image: %s. Available images: %s", 
			imageRef, strings.Join(availableImages, ", "))
	}

	return nil
}

// validateFlavorRef validates the flavor reference
func validateFlavorRef(flavorRef string) error {
	if flavorRef == "" {
		return fmt.Errorf("flavor reference is required")
	}

	if !IsValidFlavor(flavorRef) {
		availableFlavors := make([]string, 0, len(Flavors))
		for flavor := range Flavors {
			availableFlavors = append(availableFlavors, flavor)
		}
		return fmt.Errorf("unknown flavor: %s. Available flavors: %s", 
			flavorRef, strings.Join(availableFlavors, ", "))
	}

	return nil
}

// validateMetadata validates the metadata map
func validateMetadata(metadata map[string]string) error {
	if metadata == nil {
		return nil
	}

	for key, value := range metadata {
		if err := validateMetadataKey(key); err != nil {
			return err
		}
		if err := validateMetadataValue(value); err != nil {
			return err
		}
	}

	return nil
}

// validateMetadataKey validates a metadata key
func validateMetadataKey(key string) error {
	if key == "" {
		return fmt.Errorf("metadata key cannot be empty")
	}

	if len(key) > 63 {
		return fmt.Errorf("metadata key must be 63 characters or less: %s", key)
	}

	// Check for valid characters (alphanumeric, dash, underscore)
	validKeyRegex := regexp.MustCompile(`^[a-zA-Z0-9_-]+$`)
	if !validKeyRegex.MatchString(key) {
		return fmt.Errorf("metadata key contains invalid characters: %s", key)
	}

	return nil
}

// validateMetadataValue validates a metadata value
func validateMetadataValue(value string) error {
	if len(value) > 255 {
		return fmt.Errorf("metadata value must be 255 characters or less")
	}

	return nil
}

// validateServerID validates a server ID for operations
func validateServerID(serverID string) error {
	if serverID == "" {
		return fmt.Errorf("server ID is required")
	}

	// Check if it's a valid UUID format
	uuidRegex := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)
	if !uuidRegex.MatchString(serverID) {
		return fmt.Errorf("server ID must be a valid UUID: %s", serverID)
	}

	return nil
}