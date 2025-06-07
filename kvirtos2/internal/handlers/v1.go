package handlers

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kubevirt/kvirtos2/internal/kubevirt"
	"github.com/kubevirt/kvirtos2/internal/models"
)

// V1Handler handles v1 API requests
type V1Handler struct {
	kubevirtClient kubevirt.VMClientInterface
}

// NewV1Handler creates a new v1 handler
func NewV1Handler(kubevirtClient kubevirt.VMClientInterface) *V1Handler {
	return &V1Handler{
		kubevirtClient: kubevirtClient,
	}
}

// Health handles GET /health
func (h *V1Handler) Health(c *gin.Context) {
	c.JSON(http.StatusOK, models.HealthResponse{
		Status:   "healthy",
		Version:  "1.0.0",
		KubeVirt: "ready",
	})
}

// CreateVM handles POST /api/v1/vms
func (h *V1Handler) CreateVM(c *gin.Context) {
	var req models.CreateVMRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.V1ErrorResponse{
			Error: models.V1ErrorDetail{
				Code:    models.ErrorCodeValidation,
				Message: "Invalid request body: " + err.Error(),
			},
		})
		return
	}

	// Convert v1 request to internal format
	internalReq := models.CreateVMParams{
		Name:      req.Name,
		Image:     req.Image,
		Flavor:    req.Flavor,
		Metadata:  req.Metadata,
		AutoStart: req.AutoStart,
	}

	vm, err := h.kubevirtClient.CreateVM(c.Request.Context(), internalReq)
	if err != nil {
		if strings.Contains(err.Error(), "validation") {
			c.JSON(http.StatusBadRequest, models.V1ErrorResponse{
				Error: models.V1ErrorDetail{
					Code:    models.ErrorCodeValidation,
					Message: err.Error(),
				},
			})
			return
		}
		if strings.Contains(err.Error(), "already exists") {
			c.JSON(http.StatusConflict, models.V1ErrorResponse{
				Error: models.V1ErrorDetail{
					Code:    models.ErrorCodeAlreadyExists,
					Message: "VM with this name already exists",
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, models.V1ErrorResponse{
			Error: models.V1ErrorDetail{
				Code:    models.ErrorCodeInternal,
				Message: "Failed to create VM: " + err.Error(),
			},
		})
		return
	}

	// Convert to v1 VM format
	v1VM := h.convertInternalToV1VM(vm)

	c.JSON(http.StatusCreated, models.VMResponse{VM: v1VM})
}

// ListVMs handles GET /api/v1/vms
func (h *V1Handler) ListVMs(c *gin.Context) {
	// Parse query parameters
	status := c.Query("status")
	label := c.Query("label")

	internalVMs, err := h.kubevirtClient.ListVMs(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.V1ErrorResponse{
			Error: models.V1ErrorDetail{
				Code:    models.ErrorCodeInternal,
				Message: "Failed to list VMs: " + err.Error(),
			},
		})
		return
	}

	vms := make([]models.VM, 0, len(internalVMs))
	for _, internalVM := range internalVMs {
		vm := h.convertInternalToV1VM(&internalVM)
		
		// Apply filters
		if status != "" && vm.Status != status {
			continue
		}
		if label != "" && !h.matchesLabel(vm.Metadata, label) {
			continue
		}
		
		vms = append(vms, vm)
	}

	c.JSON(http.StatusOK, models.VMListResponse{
		VMs:   vms,
		Total: len(vms),
	})
}

// GetVM handles GET /api/v1/vms/{name}
func (h *V1Handler) GetVM(c *gin.Context) {
	vmName := c.Param("name")
	if vmName == "" {
		c.JSON(http.StatusBadRequest, models.V1ErrorResponse{
			Error: models.V1ErrorDetail{
				Code:    models.ErrorCodeValidation,
				Message: "VM name is required",
			},
		})
		return
	}

	// Find VM by name
	internalVM, err := h.getVMByName(c, vmName)
	if err != nil {
		return // Error already handled
	}

	vm := h.convertInternalToV1VM(internalVM)
	c.JSON(http.StatusOK, models.VMResponse{VM: vm})
}

// UpdateVM handles PUT /api/v1/vms/{name}
func (h *V1Handler) UpdateVM(c *gin.Context) {
	vmName := c.Param("name")
	if vmName == "" {
		c.JSON(http.StatusBadRequest, models.V1ErrorResponse{
			Error: models.V1ErrorDetail{
				Code:    models.ErrorCodeValidation,
				Message: "VM name is required",
			},
		})
		return
	}

	var req models.UpdateVMRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.V1ErrorResponse{
			Error: models.V1ErrorDetail{
				Code:    models.ErrorCodeValidation,
				Message: "Invalid request body: " + err.Error(),
			},
		})
		return
	}

	// Find VM by name
	internalVM, err := h.getVMByName(c, vmName)
	if err != nil {
		return // Error already handled
	}

	// For now, we only support metadata updates
	// In a full implementation, this would update the VM's annotations
	c.JSON(http.StatusOK, models.VMResponse{VM: h.convertInternalToV1VM(internalVM)})
}

// DeleteVM handles DELETE /api/v1/vms/{name}
func (h *V1Handler) DeleteVM(c *gin.Context) {
	vmName := c.Param("name")
	if vmName == "" {
		c.JSON(http.StatusBadRequest, models.V1ErrorResponse{
			Error: models.V1ErrorDetail{
				Code:    models.ErrorCodeValidation,
				Message: "VM name is required",
			},
		})
		return
	}

	force := c.Query("force") == "true"

	// Find VM by name first
	internalVM, err := h.getVMByName(c, vmName)
	if err != nil {
		return // Error already handled
	}

	// Check if VM is running and force is not specified
	if internalVM.Status == models.InternalStatusRunning && !force {
		c.JSON(http.StatusConflict, models.V1ErrorResponse{
			Error: models.V1ErrorDetail{
				Code:    models.ErrorCodeConflict,
				Message: "VM is running and force=false",
			},
		})
		return
	}

	err = h.kubevirtClient.DeleteVM(c.Request.Context(), internalVM.ID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, models.V1ErrorResponse{
				Error: models.V1ErrorDetail{
					Code:    models.ErrorCodeNotFound,
					Message: "VM not found",
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, models.V1ErrorResponse{
			Error: models.V1ErrorDetail{
				Code:    models.ErrorCodeInternal,
				Message: "Failed to delete VM: " + err.Error(),
			},
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// StartVM handles POST /api/v1/vms/{name}/start
func (h *V1Handler) StartVM(c *gin.Context) {
	vmName := c.Param("name")
	if vmName == "" {
		c.JSON(http.StatusBadRequest, models.V1ErrorResponse{
			Error: models.V1ErrorDetail{
				Code:    models.ErrorCodeValidation,
				Message: "VM name is required",
			},
		})
		return
	}

	var req models.StartVMRequest
	c.ShouldBindJSON(&req) // Optional parameters

	// Find VM by name first
	internalVM, err := h.getVMByName(c, vmName)
	if err != nil {
		return // Error already handled
	}

	// Check if already running
	if internalVM.Status == models.InternalStatusRunning {
		vm := h.convertInternalToV1VM(internalVM)
		c.JSON(http.StatusAccepted, models.VMActionResponse{
			Message: "VM is already running",
			VM:      vm,
		})
		return
	}

	err = h.kubevirtClient.StartVM(c.Request.Context(), internalVM.ID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, models.V1ErrorResponse{
				Error: models.V1ErrorDetail{
					Code:    models.ErrorCodeNotFound,
					Message: "VM not found",
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, models.V1ErrorResponse{
			Error: models.V1ErrorDetail{
				Code:    models.ErrorCodeInternal,
				Message: "Failed to start VM: " + err.Error(),
			},
		})
		return
	}

	vm := h.convertInternalToV1VM(internalVM)
	vm.Status = models.VMStatusStarting

	c.JSON(http.StatusAccepted, models.VMActionResponse{
		Message: "VM start initiated",
		VM:      vm,
	})
}

// StopVM handles POST /api/v1/vms/{name}/stop
func (h *V1Handler) StopVM(c *gin.Context) {
	vmName := c.Param("name")
	if vmName == "" {
		c.JSON(http.StatusBadRequest, models.V1ErrorResponse{
			Error: models.V1ErrorDetail{
				Code:    models.ErrorCodeValidation,
				Message: "VM name is required",
			},
		})
		return
	}

	var req models.StopVMRequest
	c.ShouldBindJSON(&req) // Optional parameters

	// Find VM by name first
	internalVM, err := h.getVMByName(c, vmName)
	if err != nil {
		return // Error already handled
	}

	// Check if already stopped
	if internalVM.Status == models.InternalStatusStopped {
		vm := h.convertInternalToV1VM(internalVM)
		c.JSON(http.StatusAccepted, models.VMActionResponse{
			Message: "VM is already stopped",
			VM:      vm,
		})
		return
	}

	err = h.kubevirtClient.StopVM(c.Request.Context(), internalVM.ID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, models.V1ErrorResponse{
				Error: models.V1ErrorDetail{
					Code:    models.ErrorCodeNotFound,
					Message: "VM not found",
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, models.V1ErrorResponse{
			Error: models.V1ErrorDetail{
				Code:    models.ErrorCodeInternal,
				Message: "Failed to stop VM: " + err.Error(),
			},
		})
		return
	}

	vm := h.convertInternalToV1VM(internalVM)
	vm.Status = models.VMStatusStopping

	c.JSON(http.StatusAccepted, models.VMActionResponse{
		Message: "VM stop initiated",
		VM:      vm,
	})
}

// RestartVM handles POST /api/v1/vms/{name}/restart
func (h *V1Handler) RestartVM(c *gin.Context) {
	vmName := c.Param("name")
	if vmName == "" {
		c.JSON(http.StatusBadRequest, models.V1ErrorResponse{
			Error: models.V1ErrorDetail{
				Code:    models.ErrorCodeValidation,
				Message: "VM name is required",
			},
		})
		return
	}

	var req models.RestartVMRequest
	c.ShouldBindJSON(&req) // Optional parameters

	// Find VM by name first
	internalVM, err := h.getVMByName(c, vmName)
	if err != nil {
		return // Error already handled
	}

	// Implement restart as stop + start
	err = h.kubevirtClient.StopVM(c.Request.Context(), internalVM.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.V1ErrorResponse{
			Error: models.V1ErrorDetail{
				Code:    models.ErrorCodeInternal,
				Message: "Failed to restart VM: " + err.Error(),
			},
		})
		return
	}

	err = h.kubevirtClient.StartVM(c.Request.Context(), internalVM.ID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.V1ErrorResponse{
			Error: models.V1ErrorDetail{
				Code:    models.ErrorCodeInternal,
				Message: "Failed to restart VM: " + err.Error(),
			},
		})
		return
	}

	vm := h.convertInternalToV1VM(internalVM)
	vm.Status = models.VMStatusStarting

	c.JSON(http.StatusAccepted, models.VMActionResponse{
		Message: "VM restart initiated",
		VM:      vm,
	})
}

// GetVMConsole handles GET /api/v1/vms/{name}/console
func (h *V1Handler) GetVMConsole(c *gin.Context) {
	vmName := c.Param("name")
	if vmName == "" {
		c.JSON(http.StatusBadRequest, models.V1ErrorResponse{
			Error: models.V1ErrorDetail{
				Code:    models.ErrorCodeValidation,
				Message: "VM name is required",
			},
		})
		return
	}

	// Find VM by name first
	_, err := h.getVMByName(c, vmName)
	if err != nil {
		return // Error already handled
	}

	// For now, return a placeholder response
	c.JSON(http.StatusOK, models.ConsoleResponse{
		Type:  "vnc",
		URL:   "ws://localhost:8080/api/v1/vms/" + vmName + "/console/vnc",
		Token: "console-token-123",
	})
}

// Helper methods

// getVMByName finds a VM by name and handles errors
func (h *V1Handler) getVMByName(c *gin.Context, vmName string) (*models.InternalVM, error) {
	internalVMs, err := h.kubevirtClient.ListVMs(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.V1ErrorResponse{
			Error: models.V1ErrorDetail{
				Code:    models.ErrorCodeInternal,
				Message: "Failed to list VMs: " + err.Error(),
			},
		})
		return nil, err
	}

	for _, internalVM := range internalVMs {
		if internalVM.Name == vmName {
			return &internalVM, nil
		}
	}

	c.JSON(http.StatusNotFound, models.V1ErrorResponse{
		Error: models.V1ErrorDetail{
			Code:    models.ErrorCodeNotFound,
			Message: "VM not found",
		},
	})
	return nil, fmt.Errorf("VM not found")
}

// convertInternalToV1VM converts from internal VM model to v1 VM model
func (h *V1Handler) convertInternalToV1VM(internalVM *models.InternalVM) models.VM {
	vm := models.VM{
		Name:     internalVM.Name,
		Status:   h.convertInternalStatus(internalVM.Status),
		Image:    internalVM.Image,
		Flavor:   internalVM.Flavor,
		Created:  internalVM.Created,
		Updated:  internalVM.Updated,
		Metadata: internalVM.Metadata,
		Node:     internalVM.Node,
	}

	// Convert networks
	if len(internalVM.Addresses) > 0 {
		vm.Networks = make([]models.NetworkAttachment, 0, len(internalVM.Addresses))
		for netName, addresses := range internalVM.Addresses {
			network := models.NetworkAttachment{
				Name:      netName,
				Type:      models.NetworkTypePod,
				Addresses: make([]models.V1Address, 0, len(addresses)),
			}
			for _, addr := range addresses {
				network.Addresses = append(network.Addresses, models.V1Address{
					IP:      addr.IP,
					Type:    h.convertAddressType(addr.Type),
					Version: addr.Version,
				})
			}
			vm.Networks = append(vm.Networks, network)
		}
	}

	// Set resources if available
	if internalVM.Resources != nil {
		vm.Resources = &models.ResourceInfo{
			CPU:    internalVM.Resources.CPU,
			Memory: internalVM.Resources.Memory,
		}
	}

	return vm
}

// convertInternalStatus converts from internal status to v1 status
func (h *V1Handler) convertInternalStatus(internalStatus string) string {
	switch internalStatus {
	case models.InternalStatusRunning:
		return models.VMStatusRunning
	case models.InternalStatusStarting:
		return models.VMStatusStarting
	case models.InternalStatusStopped:
		return models.VMStatusStopped
	case models.InternalStatusStopping:
		return models.VMStatusStopping
	case models.InternalStatusError:
		return models.VMStatusError
	default:
		return models.VMStatusUnknown
	}
}

// convertAddressType converts from internal address type to v1 address type
func (h *V1Handler) convertAddressType(internalType string) string {
	switch internalType {
	case models.InternalAddressTypeInternal:
		return models.AddressTypeInternal
	case models.InternalAddressTypeExternal:
		return models.AddressTypeExternal
	case models.InternalAddressTypeFloating:
		return models.AddressTypeFloating
	default:
		return models.AddressTypeInternal
	}
}

// matchesLabel checks if VM metadata matches the label filter
func (h *V1Handler) matchesLabel(metadata map[string]string, labelFilter string) bool {
	if metadata == nil {
		return false
	}
	
	// Parse label filter (key=value format)
	parts := strings.SplitN(labelFilter, "=", 2)
	if len(parts) != 2 {
		return false
	}
	
	key, value := parts[0], parts[1]
	return metadata[key] == value
}