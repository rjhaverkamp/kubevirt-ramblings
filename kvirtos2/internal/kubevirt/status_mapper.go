package kubevirt

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"github.com/kubevirt/kvirtos2/internal/models"
)

// StatusMapper handles the mapping between KubeVirt VM states and Nova API states
type StatusMapper struct{}

// NewStatusMapper creates a new StatusMapper instance
func NewStatusMapper() *StatusMapper {
	return &StatusMapper{}
}

// GetVMStatus determines the Nova API status based on VM and VMI state
func (s *StatusMapper) GetVMStatus(vm *unstructured.Unstructured, vmi *unstructured.Unstructured) string {
	running, _, _ := unstructured.NestedBool(vm.Object, "spec", "running")
	if !running {
		return models.StatusShutoff
	}

	if vmi == nil {
		return models.StatusBuild
	}

	phase, found, _ := unstructured.NestedString(vmi.Object, "status", "phase")
	if !found {
		return models.StatusBuild
	}

	return s.mapPhaseToStatus(phase)
}

// GetPowerState determines the Nova API power state based on VM and VMI state
func (s *StatusMapper) GetPowerState(vm *unstructured.Unstructured, vmi *unstructured.Unstructured) int {
	running, _, _ := unstructured.NestedBool(vm.Object, "spec", "running")
	if !running {
		return models.PowerStateShutdown
	}

	if vmi == nil {
		return models.PowerStateNoState
	}

	phase, found, _ := unstructured.NestedString(vmi.Object, "status", "phase")
	if !found {
		return models.PowerStateNoState
	}

	return s.mapPhaseToPowerState(phase)
}

// GetVMState determines the Nova API VM state based on the status
func (s *StatusMapper) GetVMState(status string) string {
	stateMap := map[string]string{
		models.StatusActive:   models.VMStateActive,
		models.StatusBuild:    models.VMStateBuilding,
		models.StatusShutoff:  models.VMStateStopped,
		models.StatusError:    models.VMStateError,
		models.StatusDeleted:  models.VMStateDeleted,
		models.StatusPaused:   models.VMStatePaused,
		models.StatusSuspended: models.VMStateSuspended,
	}

	if state, exists := stateMap[status]; exists {
		return state
	}
	return models.VMStateActive // default fallback
}

// mapPhaseToStatus maps KubeVirt VMI phase to Nova API status
func (s *StatusMapper) mapPhaseToStatus(phase string) string {
	statusMap := map[string]string{
		VMStateRunning:    models.StatusActive,
		VMStatePending:    models.StatusBuild,
		VMStateScheduling: models.StatusBuild,
		VMStateScheduled:  models.StatusBuild,
		VMStateFailed:     models.StatusError,
		VMStateSucceeded:  models.StatusShutoff,
	}

	if status, exists := statusMap[phase]; exists {
		return status
	}
	return models.StatusUnknown
}

// mapPhaseToPowerState maps KubeVirt VMI phase to Nova API power state
func (s *StatusMapper) mapPhaseToPowerState(phase string) int {
	powerStateMap := map[string]int{
		VMStateRunning:    models.PowerStateRunning,
		VMStatePending:    models.PowerStateNoState,
		VMStateScheduling: models.PowerStateNoState,
		VMStateScheduled:  models.PowerStateNoState,
		VMStateFailed:     models.PowerStateCrashed,
		VMStateSucceeded:  models.PowerStateShutdown,
	}

	if powerState, exists := powerStateMap[phase]; exists {
		return powerState
	}
	return models.PowerStateNoState
}

// IsVMReady checks if the VM is in a ready state
func (s *StatusMapper) IsVMReady(vm *unstructured.Unstructured, vmi *unstructured.Unstructured) bool {
	status := s.GetVMStatus(vm, vmi)
	return status == models.StatusActive
}

// IsVMTransitioning checks if the VM is in a transitioning state
func (s *StatusMapper) IsVMTransitioning(vm *unstructured.Unstructured, vmi *unstructured.Unstructured) bool {
	status := s.GetVMStatus(vm, vmi)
	transitioningStates := []string{
		models.StatusBuild,
		models.StatusReboot,
		models.StatusHardReboot,
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
	status := s.GetVMStatus(vm, vmi)
	return status == models.StatusError
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