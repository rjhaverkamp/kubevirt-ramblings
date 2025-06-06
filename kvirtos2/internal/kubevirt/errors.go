package kubevirt

import (
	"errors"
	"fmt"
)

// Predefined errors
var (
	ErrVMNotFound      = errors.New("virtual machine not found")
	ErrVMAlreadyExists = errors.New("virtual machine already exists")
	ErrInvalidRequest  = errors.New("invalid request")
	ErrVMAlreadyRunning = errors.New("virtual machine is already running")
	ErrVMAlreadyStopped = errors.New("virtual machine is already stopped")
	ErrUnauthorized    = errors.New("unauthorized access")
	ErrQuotaExceeded   = errors.New("resource quota exceeded")
)

// VMError represents an error that occurred during VM operations
type VMError struct {
	Operation string
	VMName    string
	VMID      string
	Err       error
}

func (e *VMError) Error() string {
	if e.VMName != "" {
		return fmt.Sprintf("failed to %s VM '%s': %v", e.Operation, e.VMName, e.Err)
	}
	if e.VMID != "" {
		return fmt.Sprintf("failed to %s VM with ID '%s': %v", e.Operation, e.VMID, e.Err)
	}
	return fmt.Sprintf("failed to %s VM: %v", e.Operation, e.Err)
}

func (e *VMError) Unwrap() error {
	return e.Err
}

// NewVMError creates a new VMError
func NewVMError(operation, vmName, vmID string, err error) *VMError {
	return &VMError{
		Operation: operation,
		VMName:    vmName,
		VMID:      vmID,
		Err:       err,
	}
}

// ValidationError represents a validation error
type ValidationError struct {
	Field   string
	Value   string
	Message string
}

func (e *ValidationError) Error() string {
	if e.Value != "" {
		return fmt.Sprintf("validation failed for field '%s' with value '%s': %s", e.Field, e.Value, e.Message)
	}
	return fmt.Sprintf("validation failed for field '%s': %s", e.Field, e.Message)
}

// NewValidationError creates a new ValidationError
func NewValidationError(field, value, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Value:   value,
		Message: message,
	}
}

// ConfigurationError represents a configuration error
type ConfigurationError struct {
	Component string
	Message   string
	Err       error
}

func (e *ConfigurationError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("configuration error in %s: %s - %v", e.Component, e.Message, e.Err)
	}
	return fmt.Sprintf("configuration error in %s: %s", e.Component, e.Message)
}

func (e *ConfigurationError) Unwrap() error {
	return e.Err
}

// NewConfigurationError creates a new ConfigurationError
func NewConfigurationError(component, message string, err error) *ConfigurationError {
	return &ConfigurationError{
		Component: component,
		Message:   message,
		Err:       err,
	}
}

// KubernetesError represents an error from Kubernetes API
type KubernetesError struct {
	Operation string
	Resource  string
	Namespace string
	Name      string
	Err       error
}

func (e *KubernetesError) Error() string {
	if e.Name != "" {
		return fmt.Sprintf("kubernetes error during %s of %s '%s/%s': %v", 
			e.Operation, e.Resource, e.Namespace, e.Name, e.Err)
	}
	return fmt.Sprintf("kubernetes error during %s of %s in namespace '%s': %v", 
		e.Operation, e.Resource, e.Namespace, e.Err)
}

func (e *KubernetesError) Unwrap() error {
	return e.Err
}

// NewKubernetesError creates a new KubernetesError
func NewKubernetesError(operation, resource, namespace, name string, err error) *KubernetesError {
	return &KubernetesError{
		Operation: operation,
		Resource:  resource,
		Namespace: namespace,
		Name:      name,
		Err:       err,
	}
}

// IsNotFoundError checks if the error is a "not found" error
func IsNotFoundError(err error) bool {
	var vmErr *VMError
	if errors.As(err, &vmErr) {
		return errors.Is(vmErr.Err, ErrVMNotFound)
	}
	return errors.Is(err, ErrVMNotFound)
}

// IsValidationError checks if the error is a validation error
func IsValidationError(err error) bool {
	var valErr *ValidationError
	return errors.As(err, &valErr)
}

// IsConfigurationError checks if the error is a configuration error
func IsConfigurationError(err error) bool {
	var confErr *ConfigurationError
	return errors.As(err, &confErr)
}

// IsKubernetesError checks if the error is a Kubernetes error
func IsKubernetesError(err error) bool {
	var k8sErr *KubernetesError
	return errors.As(err, &k8sErr)
}