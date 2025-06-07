package models

import (
	"time"
)

// VM represents a virtual machine in the v1 API
type VM struct {
	Name       string            `json:"name"`
	Status     string            `json:"status"`
	Image      string            `json:"image"`
	Flavor     string            `json:"flavor"`
	Created    time.Time         `json:"created"`
	Updated    time.Time         `json:"updated"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	Networks   []NetworkAttachment `json:"networks,omitempty"`
	Volumes    []VolumeAttachment  `json:"volumes,omitempty"`
	Node       string            `json:"node,omitempty"`
	Resources  *ResourceInfo     `json:"resources,omitempty"`
	Conditions []Condition       `json:"conditions,omitempty"`
}

// VM Status constants for v1 API
const (
	VMStatusStopped  = "stopped"
	VMStatusStarting = "starting"
	VMStatusRunning  = "running"
	VMStatusStopping = "stopping"
	VMStatusError    = "error"
	VMStatusUnknown  = "unknown"
)

// ResourceInfo represents actual resource allocation
type ResourceInfo struct {
	CPU     string `json:"cpu"`     // e.g., "1000m"
	Memory  string `json:"memory"`  // e.g., "2Gi"
	Storage string `json:"storage,omitempty"` // future extension
}

// NetworkAttachment represents a network attachment
type NetworkAttachment struct {
	Name      string      `json:"name"`
	Type      string      `json:"type"`
	Addresses []V1Address `json:"addresses,omitempty"`
	Config    map[string]interface{} `json:"config,omitempty"`
}

// V1Address represents an IP address assignment
type V1Address struct {
	IP      string `json:"ip"`
	Type    string `json:"type"`    // internal, external, floating
	Version int    `json:"version"` // IP version (4 or 6)
}

// VolumeAttachment represents a volume attachment (future extension)
type VolumeAttachment struct {
	Name      string                 `json:"name"`
	Type      string                 `json:"type"` // disk, cdrom
	Source    VolumeSource           `json:"source"`
	MountPath string                 `json:"mountPath,omitempty"`
	ReadOnly  bool                   `json:"readOnly,omitempty"`
	Config    map[string]interface{} `json:"config,omitempty"`
}

// VolumeSource represents volume source configuration
type VolumeSource struct {
	Type      string `json:"type"`      // pvc, configMap, secret, containerDisk
	Reference string `json:"reference"` // Source reference
}

// Condition represents a Kubernetes-style condition
type Condition struct {
	Type               string    `json:"type"`
	Status             string    `json:"status"`
	LastTransitionTime time.Time `json:"lastTransitionTime"`
	Reason             string    `json:"reason,omitempty"`
	Message            string    `json:"message,omitempty"`
}

// CreateVMRequest represents a request to create a new VM
type CreateVMRequest struct {
	Name      string            `json:"name" binding:"required"`
	Image     string            `json:"image" binding:"required"`
	Flavor    string            `json:"flavor" binding:"required"`
	Metadata  map[string]string `json:"metadata,omitempty"`
	AutoStart bool              `json:"autoStart,omitempty"`
	Networks  []NetworkConfig   `json:"networks,omitempty"`
	Volumes   []VolumeConfig    `json:"volumes,omitempty"`
}

// NetworkConfig represents network configuration for VM creation
type NetworkConfig struct {
	Name   string                 `json:"name,omitempty"`
	Type   string                 `json:"type,omitempty"` // pod, bridge, sriov
	Config map[string]interface{} `json:"config,omitempty"`
}

// VolumeConfig represents volume configuration for VM creation
type VolumeConfig struct {
	Name      string                 `json:"name,omitempty"`
	Type      string                 `json:"type,omitempty"` // disk, cdrom
	Source    VolumeSource           `json:"source"`
	MountPath string                 `json:"mountPath,omitempty"`
	ReadOnly  bool                   `json:"readOnly,omitempty"`
	Config    map[string]interface{} `json:"config,omitempty"`
}

// UpdateVMRequest represents a request to update VM metadata
type UpdateVMRequest struct {
	Metadata map[string]string `json:"metadata,omitempty"`
}

// VMActionRequest represents an action request with optional parameters
type VMActionRequest struct {
	Timeout string `json:"timeout,omitempty"`
}

// StartVMRequest represents a request to start a VM
type StartVMRequest struct {
	Timeout string `json:"timeout,omitempty"`
}

// StopVMRequest represents a request to stop a VM
type StopVMRequest struct {
	Graceful bool   `json:"graceful,omitempty"`
	Timeout  string `json:"timeout,omitempty"`
}

// RestartVMRequest represents a request to restart a VM
type RestartVMRequest struct {
	Graceful bool   `json:"graceful,omitempty"`
	Timeout  string `json:"timeout,omitempty"`
}

// VMResponse represents a response containing a single VM
type VMResponse struct {
	VM VM `json:"vm"`
}

// VMListResponse represents a response containing multiple VMs
type VMListResponse struct {
	VMs   []VM `json:"vms"`
	Total int  `json:"total"`
}

// VMActionResponse represents a response to a VM action
type VMActionResponse struct {
	Message string `json:"message"`
	VM      VM     `json:"vm"`
}

// ConsoleResponse represents console access information
type ConsoleResponse struct {
	Type  string `json:"type"`
	URL   string `json:"url"`
	Token string `json:"token"`
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status   string `json:"status"`
	Version  string `json:"version"`
	KubeVirt string `json:"kubevirt"`
}

// V1ErrorResponse represents an error response in v1 API format
type V1ErrorResponse struct {
	Error V1ErrorDetail `json:"error"`
}

// V1ErrorDetail represents error details in v1 API format
type V1ErrorDetail struct {
	Code    string                 `json:"code"`
	Message string                 `json:"message"`
	Details map[string]interface{} `json:"details,omitempty"`
}

// Error codes for v1 API
const (
	ErrorCodeValidation    = "VALIDATION_ERROR"
	ErrorCodeNotFound      = "NOT_FOUND"
	ErrorCodeAlreadyExists = "ALREADY_EXISTS"
	ErrorCodeConflict      = "CONFLICT"
	ErrorCodeInternal      = "INTERNAL_ERROR"
	ErrorCodeTimeout       = "TIMEOUT"
)

// Network types
const (
	NetworkTypePod    = "pod"
	NetworkTypeBridge = "bridge"
	NetworkTypeSRIOV  = "sriov"
)

// Volume types
const (
	VolumeTypeDisk  = "disk"
	VolumeTypeCDROM = "cdrom"
)

// Volume source types
const (
	VolumeSourceTypePVC           = "pvc"
	VolumeSourceTypeConfigMap     = "configMap"
	VolumeSourceTypeSecret        = "secret"
	VolumeSourceTypeContainerDisk = "containerDisk"
)

// Address types
const (
	AddressTypeInternal  = "internal"
	AddressTypeExternal  = "external"
	AddressTypeFloating  = "floating"
)

// Condition types
const (
	ConditionTypeReady = "Ready"
)

// Condition statuses
const (
	ConditionStatusTrue    = "True"
	ConditionStatusFalse   = "False"
	ConditionStatusUnknown = "Unknown"
)