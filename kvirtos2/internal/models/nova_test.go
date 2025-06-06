package models

import (
	"testing"
	"time"
)

func TestServerModel(t *testing.T) {
	server := Server{
		ID:       "test-id",
		Name:     "test-vm",
		Status:   StatusActive,
		VMState:  VMStateActive,
		Created:  time.Now(),
		Updated:  time.Now(),
		Metadata: map[string]string{"test": "value"},
	}

	if server.ID != "test-id" {
		t.Errorf("Expected ID 'test-id', got %s", server.ID)
	}

	if server.Status != StatusActive {
		t.Errorf("Expected status %s, got %s", StatusActive, server.Status)
	}
}

func TestCreateServerRequest(t *testing.T) {
	req := CreateServerRequest{
		Server: CreateServerParams{
			Name:      "test-vm",
			ImageRef:  "ubuntu-20.04",
			FlavorRef: "m1.small",
			Metadata:  map[string]string{"env": "test"},
		},
	}

	if req.Server.Name != "test-vm" {
		t.Errorf("Expected name 'test-vm', got %s", req.Server.Name)
	}

	if req.Server.ImageRef != "ubuntu-20.04" {
		t.Errorf("Expected image 'ubuntu-20.04', got %s", req.Server.ImageRef)
	}
}

func TestStatusConstants(t *testing.T) {
	statuses := []string{
		StatusActive,
		StatusBuild,
		StatusDeleted,
		StatusError,
		StatusShutoff,
	}

	for _, status := range statuses {
		if status == "" {
			t.Error("Status constant should not be empty")
		}
	}
}