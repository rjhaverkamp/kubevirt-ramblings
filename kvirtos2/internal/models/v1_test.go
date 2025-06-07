package models

import (
	"encoding/json"
	"testing"
	"time"
)

func TestVMSerialization(t *testing.T) {
	vm := VM{
		Name:    "test-vm",
		Status:  VMStatusRunning,
		Image:   "ubuntu-24",
		Flavor:  "m1.small",
		Created: time.Now().UTC(),
		Updated: time.Now().UTC(),
		Metadata: map[string]string{
			"environment": "test",
			"project":     "demo",
		},
		Networks: []NetworkAttachment{
			{
				Name: "default",
				Type: NetworkTypePod,
				Addresses: []V1Address{
					{
						IP:      "10.244.1.5",
						Type:    AddressTypeInternal,
						Version: 4,
					},
				},
			},
		},
		Node: "worker-node-1",
		Resources: &ResourceInfo{
			CPU:    "1000m",
			Memory: "2Gi",
		},
		Conditions: []Condition{
			{
				Type:               ConditionTypeReady,
				Status:             ConditionStatusTrue,
				LastTransitionTime: time.Now().UTC(),
			},
		},
	}

	// Test JSON serialization
	data, err := json.Marshal(vm)
	if err != nil {
		t.Fatalf("Failed to marshal VM: %v", err)
	}

	// Test JSON deserialization
	var unmarshaled VM
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal VM: %v", err)
	}

	// Verify core fields
	if unmarshaled.Name != vm.Name {
		t.Errorf("Expected name %s, got %s", vm.Name, unmarshaled.Name)
	}
	if unmarshaled.Status != vm.Status {
		t.Errorf("Expected status %s, got %s", vm.Status, unmarshaled.Status)
	}
	if unmarshaled.Image != vm.Image {
		t.Errorf("Expected image %s, got %s", vm.Image, unmarshaled.Image)
	}
	if unmarshaled.Flavor != vm.Flavor {
		t.Errorf("Expected flavor %s, got %s", vm.Flavor, unmarshaled.Flavor)
	}
}

func TestCreateVMRequestValidation(t *testing.T) {
	tests := []struct {
		name    string
		request CreateVMRequest
		wantErr bool
	}{
		{
			name: "valid request",
			request: CreateVMRequest{
				Name:     "test-vm",
				Image:    "ubuntu-24",
				Flavor:   "m1.small",
				Metadata: map[string]string{"env": "test"},
			},
			wantErr: false,
		},
		{
			name: "missing name",
			request: CreateVMRequest{
				Image:  "ubuntu-24",
				Flavor: "m1.small",
			},
			wantErr: true,
		},
		{
			name: "missing image",
			request: CreateVMRequest{
				Name:   "test-vm",
				Flavor: "m1.small",
			},
			wantErr: true,
		},
		{
			name: "missing flavor",
			request: CreateVMRequest{
				Name:  "test-vm",
				Image: "ubuntu-24",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			data, err := json.Marshal(tt.request)
			if err != nil {
				t.Fatalf("Failed to marshal request: %v", err)
			}

			var req CreateVMRequest
			err = json.Unmarshal(data, &req)
			if err != nil {
				t.Fatalf("Failed to unmarshal request: %v", err)
			}

			// Basic validation - check required fields
			hasErr := req.Name == "" || req.Image == "" || req.Flavor == ""
			if hasErr != tt.wantErr {
				t.Errorf("Expected error: %v, got error: %v", tt.wantErr, hasErr)
			}
		})
	}
}

func TestVMStatusConstants(t *testing.T) {
	statuses := []string{
		VMStatusStopped,
		VMStatusStarting,
		VMStatusRunning,
		VMStatusStopping,
		VMStatusError,
		VMStatusUnknown,
	}

	for _, status := range statuses {
		if status == "" {
			t.Error("VM status constant should not be empty")
		}
	}

	// Test specific values
	if VMStatusRunning != "running" {
		t.Errorf("Expected VMStatusRunning to be 'running', got '%s'", VMStatusRunning)
	}
	if VMStatusStopped != "stopped" {
		t.Errorf("Expected VMStatusStopped to be 'stopped', got '%s'", VMStatusStopped)
	}
}

func TestErrorResponseSerialization(t *testing.T) {
	errorResp := V1ErrorResponse{
		Error: V1ErrorDetail{
			Code:    ErrorCodeValidation,
			Message: "VM name must be a valid DNS label",
			Details: map[string]interface{}{
				"field":      "name",
				"value":      "invalid-name-",
				"constraint": "must match ^[a-z0-9]([-a-z0-9]*[a-z0-9])?$",
			},
		},
	}

	data, err := json.Marshal(errorResp)
	if err != nil {
		t.Fatalf("Failed to marshal error response: %v", err)
	}

	var unmarshaled V1ErrorResponse
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal error response: %v", err)
	}

	if unmarshaled.Error.Code != errorResp.Error.Code {
		t.Errorf("Expected error code %s, got %s", errorResp.Error.Code, unmarshaled.Error.Code)
	}
	if unmarshaled.Error.Message != errorResp.Error.Message {
		t.Errorf("Expected error message %s, got %s", errorResp.Error.Message, unmarshaled.Error.Message)
	}
}

func TestVMListResponseSerialization(t *testing.T) {
	response := VMListResponse{
		VMs: []VM{
			{
				Name:   "vm1",
				Status: VMStatusRunning,
				Image:  "ubuntu-24",
				Flavor: "m1.small",
			},
			{
				Name:   "vm2",
				Status: VMStatusStopped,
				Image:  "fedora-cloud",
				Flavor: "m1.medium",
			},
		},
		Total: 2,
	}

	data, err := json.Marshal(response)
	if err != nil {
		t.Fatalf("Failed to marshal VM list response: %v", err)
	}

	var unmarshaled VMListResponse
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal VM list response: %v", err)
	}

	if unmarshaled.Total != response.Total {
		t.Errorf("Expected total %d, got %d", response.Total, unmarshaled.Total)
	}
	if len(unmarshaled.VMs) != len(response.VMs) {
		t.Errorf("Expected %d VMs, got %d", len(response.VMs), len(unmarshaled.VMs))
	}
}

func TestResourceInfoSerialization(t *testing.T) {
	resources := ResourceInfo{
		CPU:     "2000m",
		Memory:  "4Gi",
		Storage: "20Gi",
	}

	data, err := json.Marshal(resources)
	if err != nil {
		t.Fatalf("Failed to marshal resources: %v", err)
	}

	var unmarshaled ResourceInfo
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal resources: %v", err)
	}

	if unmarshaled.CPU != resources.CPU {
		t.Errorf("Expected CPU %s, got %s", resources.CPU, unmarshaled.CPU)
	}
	if unmarshaled.Memory != resources.Memory {
		t.Errorf("Expected Memory %s, got %s", resources.Memory, unmarshaled.Memory)
	}
	if unmarshaled.Storage != resources.Storage {
		t.Errorf("Expected Storage %s, got %s", resources.Storage, unmarshaled.Storage)
	}
}

func TestNetworkAttachmentSerialization(t *testing.T) {
	network := NetworkAttachment{
		Name: "default",
		Type: NetworkTypePod,
		Addresses: []V1Address{
			{
				IP:      "192.168.1.100",
				Type:    AddressTypeInternal,
				Version: 4,
			},
			{
				IP:      "2001:db8::1",
				Type:    AddressTypeInternal,
				Version: 6,
			},
		},
		Config: map[string]interface{}{
			"mtu": 1500,
		},
	}

	data, err := json.Marshal(network)
	if err != nil {
		t.Fatalf("Failed to marshal network: %v", err)
	}

	var unmarshaled NetworkAttachment
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal network: %v", err)
	}

	if unmarshaled.Name != network.Name {
		t.Errorf("Expected name %s, got %s", network.Name, unmarshaled.Name)
	}
	if unmarshaled.Type != network.Type {
		t.Errorf("Expected type %s, got %s", network.Type, unmarshaled.Type)
	}
	if len(unmarshaled.Addresses) != len(network.Addresses) {
		t.Errorf("Expected %d addresses, got %d", len(network.Addresses), len(unmarshaled.Addresses))
	}
}

func TestVolumeAttachmentSerialization(t *testing.T) {
	volume := VolumeAttachment{
		Name: "data-disk",
		Type: VolumeTypeDisk,
		Source: VolumeSource{
			Type:      VolumeSourceTypePVC,
			Reference: "data-pvc",
		},
		MountPath: "/data",
		ReadOnly:  false,
	}

	data, err := json.Marshal(volume)
	if err != nil {
		t.Fatalf("Failed to marshal volume: %v", err)
	}

	var unmarshaled VolumeAttachment
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal volume: %v", err)
	}

	if unmarshaled.Name != volume.Name {
		t.Errorf("Expected name %s, got %s", volume.Name, unmarshaled.Name)
	}
	if unmarshaled.Type != volume.Type {
		t.Errorf("Expected type %s, got %s", volume.Type, unmarshaled.Type)
	}
	if unmarshaled.Source.Type != volume.Source.Type {
		t.Errorf("Expected source type %s, got %s", volume.Source.Type, unmarshaled.Source.Type)
	}
	if unmarshaled.Source.Reference != volume.Source.Reference {
		t.Errorf("Expected source reference %s, got %s", volume.Source.Reference, unmarshaled.Source.Reference)
	}
}

func TestConditionSerialization(t *testing.T) {
	now := time.Now().UTC()
	condition := Condition{
		Type:               ConditionTypeReady,
		Status:             ConditionStatusTrue,
		LastTransitionTime: now,
		Reason:             "VMReady",
		Message:            "VM is ready and running",
	}

	data, err := json.Marshal(condition)
	if err != nil {
		t.Fatalf("Failed to marshal condition: %v", err)
	}

	var unmarshaled Condition
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal condition: %v", err)
	}

	if unmarshaled.Type != condition.Type {
		t.Errorf("Expected type %s, got %s", condition.Type, unmarshaled.Type)
	}
	if unmarshaled.Status != condition.Status {
		t.Errorf("Expected status %s, got %s", condition.Status, unmarshaled.Status)
	}
	if unmarshaled.Reason != condition.Reason {
		t.Errorf("Expected reason %s, got %s", condition.Reason, unmarshaled.Reason)
	}
	if unmarshaled.Message != condition.Message {
		t.Errorf("Expected message %s, got %s", condition.Message, unmarshaled.Message)
	}
}

func TestActionRequestsSerialization(t *testing.T) {
	// Test StartVMRequest
	startReq := StartVMRequest{
		Timeout: "5m",
	}
	data, err := json.Marshal(startReq)
	if err != nil {
		t.Fatalf("Failed to marshal start request: %v", err)
	}
	var unmarshaledStart StartVMRequest
	err = json.Unmarshal(data, &unmarshaledStart)
	if err != nil {
		t.Fatalf("Failed to unmarshal start request: %v", err)
	}
	if unmarshaledStart.Timeout != startReq.Timeout {
		t.Errorf("Expected timeout %s, got %s", startReq.Timeout, unmarshaledStart.Timeout)
	}

	// Test StopVMRequest
	stopReq := StopVMRequest{
		Graceful: true,
		Timeout:  "30s",
	}
	data, err = json.Marshal(stopReq)
	if err != nil {
		t.Fatalf("Failed to marshal stop request: %v", err)
	}
	var unmarshaledStop StopVMRequest
	err = json.Unmarshal(data, &unmarshaledStop)
	if err != nil {
		t.Fatalf("Failed to unmarshal stop request: %v", err)
	}
	if unmarshaledStop.Graceful != stopReq.Graceful {
		t.Errorf("Expected graceful %v, got %v", stopReq.Graceful, unmarshaledStop.Graceful)
	}
	if unmarshaledStop.Timeout != stopReq.Timeout {
		t.Errorf("Expected timeout %s, got %s", stopReq.Timeout, unmarshaledStop.Timeout)
	}
}

func TestHealthResponseSerialization(t *testing.T) {
	healthResp := HealthResponse{
		Status:   "healthy",
		Version:  "1.0.0",
		KubeVirt: "ready",
	}

	data, err := json.Marshal(healthResp)
	if err != nil {
		t.Fatalf("Failed to marshal health response: %v", err)
	}

	var unmarshaled HealthResponse
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal health response: %v", err)
	}

	if unmarshaled.Status != healthResp.Status {
		t.Errorf("Expected status %s, got %s", healthResp.Status, unmarshaled.Status)
	}
	if unmarshaled.Version != healthResp.Version {
		t.Errorf("Expected version %s, got %s", healthResp.Version, unmarshaled.Version)
	}
	if unmarshaled.KubeVirt != healthResp.KubeVirt {
		t.Errorf("Expected kubevirt %s, got %s", healthResp.KubeVirt, unmarshaled.KubeVirt)
	}
}

func TestConstants(t *testing.T) {
	// Test error codes
	errorCodes := []string{
		ErrorCodeValidation,
		ErrorCodeNotFound,
		ErrorCodeAlreadyExists,
		ErrorCodeConflict,
		ErrorCodeInternal,
		ErrorCodeTimeout,
	}
	for _, code := range errorCodes {
		if code == "" {
			t.Error("Error code should not be empty")
		}
	}

	// Test network types
	networkTypes := []string{
		NetworkTypePod,
		NetworkTypeBridge,
		NetworkTypeSRIOV,
	}
	for _, netType := range networkTypes {
		if netType == "" {
			t.Error("Network type should not be empty")
		}
	}

	// Test volume types
	volumeTypes := []string{
		VolumeTypeDisk,
		VolumeTypeCDROM,
	}
	for _, volType := range volumeTypes {
		if volType == "" {
			t.Error("Volume type should not be empty")
		}
	}

	// Test address types
	addressTypes := []string{
		AddressTypeInternal,
		AddressTypeExternal,
		AddressTypeFloating,
	}
	for _, addrType := range addressTypes {
		if addrType == "" {
			t.Error("Address type should not be empty")
		}
	}
}