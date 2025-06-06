package kubevirt

import (
	"testing"
)

func TestGetFlavorConfig(t *testing.T) {
	tests := []struct {
		name        string
		flavorRef   string
		expectedCPU int
		expectedMem int
	}{
		{
			name:        "m1.tiny flavor",
			flavorRef:   "m1.tiny",
			expectedCPU: 500,
			expectedMem: 512,
		},
		{
			name:        "m1.small flavor",
			flavorRef:   "m1.small",
			expectedCPU: 1000,
			expectedMem: 2048,
		},
		{
			name:        "m1.medium flavor",
			flavorRef:   "m1.medium",
			expectedCPU: 2000,
			expectedMem: 4096,
		},
		{
			name:        "m1.large flavor",
			flavorRef:   "m1.large",
			expectedCPU: 4000,
			expectedMem: 8192,
		},
		{
			name:        "unknown flavor returns default",
			flavorRef:   "unknown",
			expectedCPU: 1000,
			expectedMem: 1024,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := GetFlavorConfig(tt.flavorRef)
			if config.CPU != tt.expectedCPU {
				t.Errorf("GetFlavorConfig(%s).CPU = %d, want %d", tt.flavorRef, config.CPU, tt.expectedCPU)
			}
			if config.Memory != tt.expectedMem {
				t.Errorf("GetFlavorConfig(%s).Memory = %d, want %d", tt.flavorRef, config.Memory, tt.expectedMem)
			}
		})
	}
}

func TestGetImageConfig(t *testing.T) {
	tests := []struct {
		name           string
		imageRef       string
		expectedImage  string
		expectedDesc   string
	}{
		{
			name:          "ubuntu-20.04 image",
			imageRef:      "ubuntu-20.04",
			expectedImage: "quay.io/kubevirt/ubuntu-cloud-container-disk-demo",
			expectedDesc:  "Ubuntu 20.04 Cloud Image",
		},
		{
			name:          "fedora-cloud image",
			imageRef:      "fedora-cloud",
			expectedImage: "quay.io/kubevirt/fedora-cloud-container-disk-demo",
			expectedDesc:  "Fedora Cloud Image",
		},
		{
			name:          "cirros image",
			imageRef:      "cirros",
			expectedImage: "quay.io/kubevirt/cirros-container-disk-demo",
			expectedDesc:  "CirrOS Test Image",
		},
		{
			name:          "unknown image returns default",
			imageRef:      "unknown",
			expectedImage: "quay.io/kubevirt/fedora-cloud-container-disk-demo",
			expectedDesc:  "Fedora Cloud Image",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config := GetImageConfig(tt.imageRef)
			if config.ContainerImage != tt.expectedImage {
				t.Errorf("GetImageConfig(%s).ContainerImage = %s, want %s", tt.imageRef, config.ContainerImage, tt.expectedImage)
			}
			if config.Description != tt.expectedDesc {
				t.Errorf("GetImageConfig(%s).Description = %s, want %s", tt.imageRef, config.Description, tt.expectedDesc)
			}
		})
	}
}

func TestIsValidFlavor(t *testing.T) {
	tests := []struct {
		name      string
		flavorRef string
		expected  bool
	}{
		{"valid m1.tiny", "m1.tiny", true},
		{"valid m1.small", "m1.small", true},
		{"valid m1.medium", "m1.medium", true},
		{"valid m1.large", "m1.large", true},
		{"invalid flavor", "invalid", false},
		{"empty flavor", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidFlavor(tt.flavorRef)
			if result != tt.expected {
				t.Errorf("IsValidFlavor(%s) = %v, want %v", tt.flavorRef, result, tt.expected)
			}
		})
	}
}

func TestIsValidImage(t *testing.T) {
	tests := []struct {
		name     string
		imageRef string
		expected bool
	}{
		{"valid ubuntu-20.04", "ubuntu-20.04", true},
		{"valid fedora-cloud", "fedora-cloud", true},
		{"valid cirros", "cirros", true},
		{"invalid image", "invalid", false},
		{"empty image", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsValidImage(tt.imageRef)
			if result != tt.expected {
				t.Errorf("IsValidImage(%s) = %v, want %v", tt.imageRef, result, tt.expected)
			}
		})
	}
}

func TestListFlavors(t *testing.T) {
	flavors := ListFlavors()
	
	// Check that we have the expected number of flavors
	expectedCount := 4
	if len(flavors) != expectedCount {
		t.Errorf("ListFlavors() returned %d flavors, want %d", len(flavors), expectedCount)
	}
	
	// Check that specific flavors exist
	expectedFlavors := []string{"m1.tiny", "m1.small", "m1.medium", "m1.large"}
	for _, flavor := range expectedFlavors {
		if _, exists := flavors[flavor]; !exists {
			t.Errorf("ListFlavors() missing expected flavor: %s", flavor)
		}
	}
}

func TestListImages(t *testing.T) {
	images := ListImages()
	
	// Check that we have the expected number of images
	expectedCount := 3
	if len(images) != expectedCount {
		t.Errorf("ListImages() returned %d images, want %d", len(images), expectedCount)
	}
	
	// Check that specific images exist
	expectedImages := []string{"ubuntu-20.04", "fedora-cloud", "cirros"}
	for _, image := range expectedImages {
		if _, exists := images[image]; !exists {
			t.Errorf("ListImages() missing expected image: %s", image)
		}
	}
}

func TestFlavorConfigValues(t *testing.T) {
	// Test specific flavor configurations
	tiny := GetFlavorConfig("m1.tiny")
	if tiny.CPU < 100 || tiny.CPU > 1000 {
		t.Errorf("m1.tiny CPU should be reasonable, got %d", tiny.CPU)
	}
	if tiny.Memory < 256 || tiny.Memory > 1024 {
		t.Errorf("m1.tiny Memory should be reasonable, got %d", tiny.Memory)
	}
	
	large := GetFlavorConfig("m1.large")
	if large.CPU <= tiny.CPU {
		t.Errorf("m1.large CPU (%d) should be greater than m1.tiny CPU (%d)", large.CPU, tiny.CPU)
	}
	if large.Memory <= tiny.Memory {
		t.Errorf("m1.large Memory (%d) should be greater than m1.tiny Memory (%d)", large.Memory, tiny.Memory)
	}
}

func TestImageConfigValues(t *testing.T) {
	// Test that all images have valid container image URLs
	for imageRef := range Images {
		config := GetImageConfig(imageRef)
		if config.ContainerImage == "" {
			t.Errorf("Image %s has empty ContainerImage", imageRef)
		}
		if config.Description == "" {
			t.Errorf("Image %s has empty Description", imageRef)
		}
		
		// Basic validation that container image looks like a valid URL
		if len(config.ContainerImage) < 10 {
			t.Errorf("Image %s has suspiciously short ContainerImage: %s", imageRef, config.ContainerImage)
		}
	}
}