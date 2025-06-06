package models

import (
	"time"
)

// Server represents a Nova server (VM) in the API
type Server struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Status      string            `json:"status"`
	TaskState   *string           `json:"task_state"`
	PowerState  int               `json:"power_state"`
	VMState     string            `json:"vm_state"`
	Created     time.Time         `json:"created"`
	Updated     time.Time         `json:"updated"`
	Flavor      Flavor            `json:"flavor"`
	Image       Image             `json:"image"`
	Metadata    map[string]string `json:"metadata"`
	Addresses   map[string][]Address `json:"addresses"`
	Links       []Link            `json:"links"`
	TenantID    string            `json:"tenant_id"`
	UserID      string            `json:"user_id"`
	HostID      string            `json:"hostId"`
	Progress    int               `json:"progress"`
}

// CreateServerRequest represents the request to create a new server
type CreateServerRequest struct {
	Server CreateServerParams `json:"server"`
}

// CreateServerParams contains the parameters for creating a server
type CreateServerParams struct {
	Name            string            `json:"name" binding:"required"`
	ImageRef        string            `json:"imageRef" binding:"required"`
	FlavorRef       string            `json:"flavorRef" binding:"required"`
	Metadata        map[string]string `json:"metadata,omitempty"`
	PersonalityFile []PersonalityFile `json:"personality,omitempty"`
	SecurityGroups  []SecurityGroup   `json:"security_groups,omitempty"`
	Networks        []Network         `json:"networks,omitempty"`
	KeyName         string            `json:"key_name,omitempty"`
	UserData        string            `json:"user_data,omitempty"`
	AvailabilityZone string           `json:"availability_zone,omitempty"`
}

// PersonalityFile represents a file to be injected into the server
type PersonalityFile struct {
	Path     string `json:"path"`
	Contents string `json:"contents"`
}

// SecurityGroup represents a security group
type SecurityGroup struct {
	Name string `json:"name"`
}

// Network represents a network attachment
type Network struct {
	UUID    string `json:"uuid,omitempty"`
	Port    string `json:"port,omitempty"`
	FixedIP string `json:"fixed_ip,omitempty"`
}

// Flavor represents a server flavor
type Flavor struct {
	ID    string `json:"id"`
	Name  string `json:"name,omitempty"`
	Links []Link `json:"links,omitempty"`
}

// Image represents a server image
type Image struct {
	ID    string `json:"id"`
	Name  string `json:"name,omitempty"`
	Links []Link `json:"links,omitempty"`
}

// Address represents an IP address
type Address struct {
	Version int    `json:"version"`
	Addr    string `json:"addr"`
	Type    string `json:"OS-EXT-IPS:type"`
}

// Link represents a link to a resource
type Link struct {
	Href string `json:"href"`
	Rel  string `json:"rel"`
	Type string `json:"type,omitempty"`
}

// ServersResponse represents the response for listing servers
type ServersResponse struct {
	Servers []Server `json:"servers"`
}

// ServerResponse represents the response for a single server
type ServerResponse struct {
	Server Server `json:"server"`
}

// ServerActionRequest represents an action request on a server
type ServerActionRequest struct {
	Action map[string]interface{} `json:"-"`
}

// Common server actions
type StartAction struct {
	Start interface{} `json:"os-start"`
}

type StopAction struct {
	Stop interface{} `json:"os-stop"`
}

type RebootAction struct {
	Reboot RebootParams `json:"reboot"`
}

type RebootParams struct {
	Type string `json:"type"` // "SOFT" or "HARD"
}

// Error response
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

type ErrorDetail struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// VM Status constants matching Nova API
const (
	StatusActive     = "ACTIVE"
	StatusBuild      = "BUILD"
	StatusDeleted    = "DELETED"
	StatusError      = "ERROR"
	StatusHardReboot = "HARD_REBOOT"
	StatusPassword   = "PASSWORD"
	StatusPaused     = "PAUSED"
	StatusReboot     = "REBOOT"
	StatusRebuild    = "REBUILD"
	StatusRescue     = "RESCUE"
	StatusResize     = "RESIZE"
	StatusRevertResize = "REVERT_RESIZE"
	StatusShutoff    = "SHUTOFF"
	StatusSuspended  = "SUSPENDED"
	StatusUnknown    = "UNKNOWN"
	StatusVerifyResize = "VERIFY_RESIZE"
)

// VM State constants
const (
	VMStateActive     = "active"
	VMStateBuilding   = "building"
	VMStateDeleted    = "deleted"
	VMStateError      = "error"
	VMStatePaused     = "paused"
	VMStateRescued    = "rescued"
	VMStateResized    = "resized"
	VMStateStopped    = "stopped"
	VMStateSuspended  = "suspended"
)

// Power State constants
const (
	PowerStateNoState = 0
	PowerStateRunning = 1
	PowerStatePaused  = 3
	PowerStateShutdown = 4
	PowerStateCrashed = 6
	PowerStateSuspended = 7
)