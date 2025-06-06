package kubevirt

import "k8s.io/apimachinery/pkg/runtime/schema"

const (
	// Labels
	AppLabel        = "app"
	AppLabelValue   = "kvirtos2"
	VMIDLabel       = "vm-id"
	ImageRefLabel   = "image-ref"
	FlavorRefLabel  = "flavor-ref"

	// Annotation prefixes
	MetadataAnnotationPrefix = "kvirtos2.io/metadata."

	// Default values
	DefaultNamespace = "default"
	DefaultUserID    = "kvirtos2"

	// KubeVirt API
	KubeVirtAPIVersion = "kubevirt.io/v1"
	VMKind             = "VirtualMachine"
	VMIKind            = "VirtualMachineInstance"

	// Network defaults
	DefaultNetworkName = "default"
	DefaultDiskName    = "disk0"
	DefaultDiskBus     = "virtio"

	// VM states
	VMStateRunning    = "Running"
	VMStatePending    = "Pending"
	VMStateScheduling = "Scheduling"
	VMStateScheduled  = "Scheduled"
	VMStateFailed     = "Failed"
	VMStateSucceeded  = "Succeeded"
)

var (
	// KubeVirt GVRs
	VirtualMachineGVR = schema.GroupVersionResource{
		Group:    "kubevirt.io",
		Version:  "v1",
		Resource: "virtualmachines",
	}
	VirtualMachineInstanceGVR = schema.GroupVersionResource{
		Group:    "kubevirt.io",
		Version:  "v1",
		Resource: "virtualmachineinstances",
	}
)