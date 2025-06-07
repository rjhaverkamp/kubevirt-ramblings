package kubevirt

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"github.com/kubevirt/kvirtos2/internal/models"
)

// StatusMapper handles the mapping between KubeVirt VM states and internal VM states
type StatusMapper struct{}

// NewStatusMapper creates a new StatusMapper instance
func NewStatusMapper() *StatusMapper {
	return &StatusMapper{}
}

// GetInternalVMStatus determines the internal VM status based on VM and VMI state
func (s *StatusMapper) GetInternalVMStatus(vm *unstructured.Unstructured, vmi *unstructured.Unstructured) string {
	running, _, _ := unstructured.NestedBool(vm.Object, "spec", "running")
	if !running {
		return models.InternalStatusStopped
	}

	if vmi == nil {
		return models.InternalStatusStarting
	}

	phase, found, _ := unstructured.NestedString(vmi.Object, "status", "phase")
	if !found {
		return models.InternalStatusStarting
	}

	return s.mapPhaseToInternalStatus(phase)
}



// mapPhaseToInternalStatus maps KubeVirt VMI phase to internal status
func (s *StatusMapper) mapPhaseToInternalStatus(phase string) string {
	statusMap := map[string]string{
		VMStateRunning:    models.InternalStatusRunning,
		VMStatePending:    models.InternalStatusStarting,
		VMStateScheduling: models.InternalStatusStarting,
		VMStateScheduled:  models.InternalStatusStarting,
		VMStateFailed:     models.InternalStatusError,
		VMStateSucceeded:  models.InternalStatusStopped,
	}

	if status, exists := statusMap[phase]; exists {
		return status
	}
	return models.InternalStatusUnknown
}

// IsVMReady checks if the VM is in a ready state using internal status
func (s *StatusMapper) IsVMReady(vm *unstructured.Unstructured, vmi *unstructured.Unstructured) bool {
	status := s.GetInternalVMStatus(vm, vmi)
	return status == models.InternalStatusRunning
}

// IsVMTransitioning checks if the VM is in a transitioning state
func (s *StatusMapper) IsVMTransitioning(vm *unstructured.Unstructured, vmi *unstructured.Unstructured) bool {
	status := s.GetInternalVMStatus(vm, vmi)
	transitioningStates := []string{
		models.InternalStatusStarting,
		models.InternalStatusStopping,
	}

	for _, state := range transitioningStates {
		if status == state {
			return true
		}
	}
	return false
}

// IsVMError checks if the VM is in an error state
func (s *StatusMapper) IsVMError(vm *unstructured.Unstructured, vmi *unstructured.Unstructured) bool {
	status := s.GetInternalVMStatus(vm, vmi)
	return status == models.InternalStatusError
}

// GetStatusReason provides additional context for the VM status
func (s *StatusMapper) GetStatusReason(vm *unstructured.Unstructured, vmi *unstructured.Unstructured) string {
	if vmi == nil {
		running, _, _ := unstructured.NestedBool(vm.Object, "spec", "running")
		if running {
			return "Starting"
		}
		return "Stopped"
	}

	phase, found, _ := unstructured.NestedString(vmi.Object, "status", "phase")
	if !found {
		return "Unknown"
	}

	// Check for more detailed reasons in conditions
	conditions, found, _ := unstructured.NestedSlice(vmi.Object, "status", "conditions")
	if found {
		for _, condition := range conditions {
			condMap, ok := condition.(map[string]interface{})
			if !ok {
				continue
			}

			condType, _ := condMap["type"].(string)
			status, _ := condMap["status"].(string)
			reason, _ := condMap["reason"].(string)

			if condType == "Ready" && status == "False" && reason != "" {
				return reason
			}
		}
	}

	return phase
}