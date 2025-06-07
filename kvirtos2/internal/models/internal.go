package models

import (
	"time"
)

// InternalVM represents a VM in the internal layer
type InternalVM struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Status      string            `json:"status"`
	Created     time.Time         `json:"created"`
	Updated     time.Time         `json:"updated"`
	Image       string            `json:"image"`
	Flavor      string            `json:"flavor"`
	Metadata    map[string]string `json:"metadata,omitempty"`
	Addresses   map[string][]InternalAddress `json:"addresses,omitempty"`
	Node        string            `json:"node,omitempty"`
	Resources   *InternalResources `json:"resources,omitempty"`
}

// CreateVMParams contains the parameters for creating a VM
type CreateVMParams struct {
	Name      string            `json:"name"`
	Image     string            `json:"image"`
	Flavor    string            `json:"flavor"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	Networks  []InternalNetwork `json:"networks,omitempty"`
	AutoStart bool              `json:"autoStart,omitempty"`
}

// InternalAddress represents an IP address
type InternalAddress struct {
	IP      string `json:"ip"`
	Type    string `json:"type"`    // internal, external, floating
	Version int    `json:"version"` // IP version (4 or 6)
}

// InternalNetwork represents a network configuration
type InternalNetwork struct {
	Name string `json:"name,omitempty"`
	Type string `json:"type,omitempty"`
}

// InternalResources represents resource allocation
type InternalResources struct {
	CPU    string `json:"cpu"`    // e.g., "1000m"
	Memory string `json:"memory"` // e.g., "2Gi"
}

// VM Status constants for internal use
const (
	InternalStatusStopped  = "stopped"
	InternalStatusStarting = "starting"
	InternalStatusRunning  = "running"
	InternalStatusStopping = "stopping"
	InternalStatusError    = "error"
	InternalStatusUnknown  = "unknown"
)

// Address types
const (
	InternalAddressTypeInternal = "internal"
	InternalAddressTypeExternal = "external"
	InternalAddressTypeFloating = "floating"
)

// Network types
const (
	InternalNetworkTypePod    = "pod"
	InternalNetworkTypeBridge = "bridge"
	InternalNetworkTypeSRIOV  = "sriov"
)