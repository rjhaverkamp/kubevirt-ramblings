package handlers

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/kubevirt/kvirtos2/internal/kubevirt"
	"github.com/kubevirt/kvirtos2/internal/models"
)

// ServerHandler handles Nova server API requests
type ServerHandler struct {
	kubevirtClient *kubevirt.Client
}

// NewServerHandler creates a new server handler
func NewServerHandler(kubevirtClient *kubevirt.Client) *ServerHandler {
	return &ServerHandler{
		kubevirtClient: kubevirtClient,
	}
}

// CreateServer handles POST /servers
func (h *ServerHandler) CreateServer(c *gin.Context) {
	var req models.CreateServerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: models.ErrorDetail{
				Code:    400,
				Message: "Invalid request body: " + err.Error(),
			},
		})
		return
	}

	server, err := h.kubevirtClient.CreateVM(c.Request.Context(), req.Server)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: models.ErrorDetail{
				Code:    500,
				Message: "Failed to create server: " + err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusAccepted, models.ServerResponse{Server: *server})
}

// ListServers handles GET /servers
func (h *ServerHandler) ListServers(c *gin.Context) {
	servers, err := h.kubevirtClient.ListVMs(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: models.ErrorDetail{
				Code:    500,
				Message: "Failed to list servers: " + err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, models.ServersResponse{Servers: servers})
}

// GetServer handles GET /servers/{id}
func (h *ServerHandler) GetServer(c *gin.Context) {
	serverID := c.Param("id")
	if serverID == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: models.ErrorDetail{
				Code:    400,
				Message: "Server ID is required",
			},
		})
		return
	}

	server, err := h.kubevirtClient.GetVM(c.Request.Context(), serverID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error: models.ErrorDetail{
					Code:    404,
					Message: "Server not found",
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: models.ErrorDetail{
				Code:    500,
				Message: "Failed to get server: " + err.Error(),
			},
		})
		return
	}

	c.JSON(http.StatusOK, models.ServerResponse{Server: *server})
}

// DeleteServer handles DELETE /servers/{id}
func (h *ServerHandler) DeleteServer(c *gin.Context) {
	serverID := c.Param("id")
	if serverID == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: models.ErrorDetail{
				Code:    400,
				Message: "Server ID is required",
			},
		})
		return
	}

	err := h.kubevirtClient.DeleteVM(c.Request.Context(), serverID)
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			c.JSON(http.StatusNotFound, models.ErrorResponse{
				Error: models.ErrorDetail{
					Code:    404,
					Message: "Server not found",
				},
			})
			return
		}
		c.JSON(http.StatusInternalServerError, models.ErrorResponse{
			Error: models.ErrorDetail{
				Code:    500,
				Message: "Failed to delete server: " + err.Error(),
			},
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// ServerAction handles POST /servers/{id}/action
func (h *ServerHandler) ServerAction(c *gin.Context) {
	serverID := c.Param("id")
	if serverID == "" {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: models.ErrorDetail{
				Code:    400,
				Message: "Server ID is required",
			},
		})
		return
	}

	var actionReq map[string]interface{}
	if err := c.ShouldBindJSON(&actionReq); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse{
			Error: models.ErrorDetail{
				Code:    400,
				Message: "Invalid request body: " + err.Error(),
			},
		})
		return
	}

	// Handle different actions
	if _, exists := actionReq["os-start"]; exists {
		err := h.kubevirtClient.StartVM(c.Request.Context(), serverID)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				c.JSON(http.StatusNotFound, models.ErrorResponse{
					Error: models.ErrorDetail{
						Code:    404,
						Message: "Server not found",
					},
				})
				return
			}
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{
				Error: models.ErrorDetail{
					Code:    500,
					Message: "Failed to start server: " + err.Error(),
				},
			})
			return
		}
		c.Status(http.StatusAccepted)
		return
	}

	if _, exists := actionReq["os-stop"]; exists {
		err := h.kubevirtClient.StopVM(c.Request.Context(), serverID)
		if err != nil {
			if strings.Contains(err.Error(), "not found") {
				c.JSON(http.StatusNotFound, models.ErrorResponse{
					Error: models.ErrorDetail{
						Code:    404,
						Message: "Server not found",
					},
				})
				return
			}
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{
				Error: models.ErrorDetail{
					Code:    500,
					Message: "Failed to stop server: " + err.Error(),
				},
			})
			return
		}
		c.Status(http.StatusAccepted)
		return
	}

	if rebootAction, exists := actionReq["reboot"]; exists {
		// Handle reboot action
		rebootData, ok := rebootAction.(map[string]interface{})
		if !ok {
			c.JSON(http.StatusBadRequest, models.ErrorResponse{
				Error: models.ErrorDetail{
					Code:    400,
					Message: "Invalid reboot action format",
				},
			})
			return
		}

		rebootType, _ := rebootData["type"].(string)
		if rebootType == "" {
			rebootType = "SOFT"
		}

		// For simplicity, we'll implement reboot as stop+start
		err := h.kubevirtClient.StopVM(c.Request.Context(), serverID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{
				Error: models.ErrorDetail{
					Code:    500,
					Message: "Failed to reboot server: " + err.Error(),
				},
			})
			return
		}

		err = h.kubevirtClient.StartVM(c.Request.Context(), serverID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.ErrorResponse{
				Error: models.ErrorDetail{
					Code:    500,
					Message: "Failed to reboot server: " + err.Error(),
				},
			})
			return
		}

		c.Status(http.StatusAccepted)
		return
	}

	c.JSON(http.StatusBadRequest, models.ErrorResponse{
		Error: models.ErrorDetail{
			Code:    400,
			Message: "Unknown action",
		},
	})
}