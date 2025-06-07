package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/kubevirt/kvirtos2/internal/models"
)

// MockKubeVirtClient is a mock implementation for testing
type MockKubeVirtClient struct {
	vms    []models.InternalVM
	vmID   int
	errors map[string]error
}

func NewMockKubeVirtClient() *MockKubeVirtClient {
	return &MockKubeVirtClient{
		vms:    make([]models.InternalVM, 0),
		vmID:   1,
		errors: make(map[string]error),
	}
}

func (m *MockKubeVirtClient) CreateVM(ctx context.Context, req models.CreateVMParams) (*models.InternalVM, error) {
	if err, exists := m.errors["CreateVM"]; exists {
		return nil, err
	}

	status := models.InternalStatusStopped
	if req.AutoStart {
		status = models.InternalStatusStarting
	}

	vm := models.InternalVM{
		ID:       "vm-" + string(rune(m.vmID)),
		Name:     req.Name,
		Status:   status,
		Created:  time.Now(),
		Updated:  time.Now(),
		Image:    req.Image,
		Flavor:   req.Flavor,
		Metadata: req.Metadata,
	}
	m.vms = append(m.vms, vm)
	m.vmID++
	return &vm, nil
}

func (m *MockKubeVirtClient) ListVMs(ctx context.Context) ([]models.InternalVM, error) {
	if err, exists := m.errors["ListVMs"]; exists {
		return nil, err
	}
	return m.vms, nil
}

func (m *MockKubeVirtClient) GetVM(ctx context.Context, serverID string) (*models.InternalVM, error) {
	if err, exists := m.errors["GetVM"]; exists {
		return nil, err
	}

	for _, vm := range m.vms {
		if vm.ID == serverID {
			return &vm, nil
		}
	}
	return nil, context.DeadlineExceeded
}

func (m *MockKubeVirtClient) DeleteVM(ctx context.Context, serverID string) error {
	if err, exists := m.errors["DeleteVM"]; exists {
		return err
	}

	for i, vm := range m.vms {
		if vm.ID == serverID {
			m.vms = append(m.vms[:i], m.vms[i+1:]...)
			return nil
		}
	}
	return context.DeadlineExceeded
}

func (m *MockKubeVirtClient) StartVM(ctx context.Context, serverID string) error {
	if err, exists := m.errors["StartVM"]; exists {
		return err
	}

	for i, vm := range m.vms {
		if vm.ID == serverID {
			m.vms[i].Status = models.InternalStatusRunning
			return nil
		}
	}
	return context.DeadlineExceeded
}

func (m *MockKubeVirtClient) StopVM(ctx context.Context, serverID string) error {
	if err, exists := m.errors["StopVM"]; exists {
		return err
	}

	for i, vm := range m.vms {
		if vm.ID == serverID {
			m.vms[i].Status = models.InternalStatusStopped
			return nil
		}
	}
	return context.DeadlineExceeded
}

func (m *MockKubeVirtClient) SetError(operation string, err error) {
	m.errors[operation] = err
}

func setupTestRouter() (*gin.Engine, *V1Handler, *MockKubeVirtClient) {
	gin.SetMode(gin.TestMode)
	
	mockClient := NewMockKubeVirtClient()
	handler := NewV1Handler(mockClient)
	
	router := gin.New()
	v1 := router.Group("/api/v1")
	{
		v1.GET("/vms", handler.ListVMs)
		v1.POST("/vms", handler.CreateVM)
		v1.GET("/vms/:name", handler.GetVM)
		v1.PUT("/vms/:name", handler.UpdateVM)
		v1.DELETE("/vms/:name", handler.DeleteVM)
		v1.POST("/vms/:name/start", handler.StartVM)
		v1.POST("/vms/:name/stop", handler.StopVM)
		v1.POST("/vms/:name/restart", handler.RestartVM)
		v1.GET("/vms/:name/console", handler.GetVMConsole)
	}
	
	return router, handler, mockClient
}

func TestHealth(t *testing.T) {
	router, handler, _ := setupTestRouter()
	router.GET("/health", handler.Health)

	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.HealthResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Status != "healthy" {
		t.Errorf("Expected status 'healthy', got '%s'", response.Status)
	}
}

func TestCreateVM(t *testing.T) {
	router, _, _ := setupTestRouter()

	reqBody := models.CreateVMRequest{
		Name:   "test-vm",
		Image:  "ubuntu-24",
		Flavor: "m1.small",
		Metadata: map[string]string{
			"env": "test",
		},
	}

	jsonBody, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/api/v1/vms", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d", http.StatusCreated, w.Code)
	}

	var response models.VMResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.VM.Name != reqBody.Name {
		t.Errorf("Expected VM name '%s', got '%s'", reqBody.Name, response.VM.Name)
	}
	if response.VM.Image != reqBody.Image {
		t.Errorf("Expected VM image '%s', got '%s'", reqBody.Image, response.VM.Image)
	}
	if response.VM.Status != models.VMStatusStopped {
		t.Errorf("Expected VM status '%s', got '%s'", models.VMStatusStopped, response.VM.Status)
	}
}

func TestCreateVMWithAutoStart(t *testing.T) {
	router, _, _ := setupTestRouter()

	reqBody := models.CreateVMRequest{
		Name:      "test-vm-autostart",
		Image:     "ubuntu-24",
		Flavor:    "m1.small",
		AutoStart: true,
	}

	jsonBody, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/api/v1/vms", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusCreated {
		t.Errorf("Expected status %d, got %d", http.StatusCreated, w.Code)
	}

	var response models.VMResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.VM.Status != models.VMStatusStarting {
		t.Errorf("Expected VM status '%s', got '%s'", models.VMStatusStarting, response.VM.Status)
	}
}

func TestCreateVMValidationError(t *testing.T) {
	router, _, _ := setupTestRouter()

	// Missing required fields
	reqBody := models.CreateVMRequest{
		Name: "test-vm",
		// Missing Image and Flavor
	}

	jsonBody, _ := json.Marshal(reqBody)
	req, _ := http.NewRequest("POST", "/api/v1/vms", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status %d, got %d", http.StatusBadRequest, w.Code)
	}

	var response models.V1ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal error response: %v", err)
	}

	if response.Error.Code != models.ErrorCodeValidation {
		t.Errorf("Expected error code '%s', got '%s'", models.ErrorCodeValidation, response.Error.Code)
	}
}

func TestListVMs(t *testing.T) {
	router, _, mockClient := setupTestRouter()

	// Add some test VMs
	mockClient.CreateVM(context.Background(), models.CreateVMParams{
		Name:   "vm1",
		Image:  "ubuntu-24",
		Flavor: "m1.small",
	})
	mockClient.CreateVM(context.Background(), models.CreateVMParams{
		Name:   "vm2",
		Image:  "fedora-cloud",
		Flavor: "m1.medium",
	})

	req, _ := http.NewRequest("GET", "/api/v1/vms", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.VMListResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Total != 2 {
		t.Errorf("Expected 2 VMs, got %d", response.Total)
	}
	if len(response.VMs) != 2 {
		t.Errorf("Expected 2 VMs in list, got %d", len(response.VMs))
	}
}

func TestGetVM(t *testing.T) {
	router, _, mockClient := setupTestRouter()

	// Add a test VM
	mockClient.CreateVM(context.Background(), models.CreateVMParams{
		Name:   "test-vm",
		Image:  "ubuntu-24",
		Flavor: "m1.small",
	})

	req, _ := http.NewRequest("GET", "/api/v1/vms/test-vm", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.VMResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.VM.Name != "test-vm" {
		t.Errorf("Expected VM name 'test-vm', got '%s'", response.VM.Name)
	}
}

func TestGetVMNotFound(t *testing.T) {
	router, _, _ := setupTestRouter()

	req, _ := http.NewRequest("GET", "/api/v1/vms/nonexistent-vm", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}

	var response models.V1ErrorResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal error response: %v", err)
	}

	if response.Error.Code != models.ErrorCodeNotFound {
		t.Errorf("Expected error code '%s', got '%s'", models.ErrorCodeNotFound, response.Error.Code)
	}
}

func TestDeleteVM(t *testing.T) {
	router, _, mockClient := setupTestRouter()

	// Add a test VM
	mockClient.CreateVM(context.Background(), models.CreateVMParams{
		Name:   "test-vm",
		Image:  "ubuntu-24",
		Flavor: "m1.small",
	})

	req, _ := http.NewRequest("DELETE", "/api/v1/vms/test-vm", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("Expected status %d, got %d", http.StatusNoContent, w.Code)
	}

	// Verify VM was deleted
	vms, _ := mockClient.ListVMs(context.Background())
	if len(vms) != 0 {
		t.Errorf("Expected 0 VMs after deletion, got %d", len(vms))
	}
}

func TestStartVM(t *testing.T) {
	router, _, mockClient := setupTestRouter()

	// Add a test VM
	mockClient.CreateVM(context.Background(), models.CreateVMParams{
		Name:   "test-vm",
		Image:  "ubuntu-24",
		Flavor: "m1.small",
	})

	req, _ := http.NewRequest("POST", "/api/v1/vms/test-vm/start", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("Expected status %d, got %d", http.StatusAccepted, w.Code)
	}

	var response models.VMActionResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Message == "" {
		t.Error("Expected non-empty message in action response")
	}
}

func TestStopVM(t *testing.T) {
	router, _, mockClient := setupTestRouter()

	// Add a test VM and start it
	mockClient.CreateVM(context.Background(), models.CreateVMParams{
		Name:   "test-vm",
		Image:  "ubuntu-24",
		Flavor: "m1.small",
	})
	mockClient.StartVM(context.Background(), "vm-1")

	req, _ := http.NewRequest("POST", "/api/v1/vms/test-vm/stop", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("Expected status %d, got %d", http.StatusAccepted, w.Code)
	}

	var response models.VMActionResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Message == "" {
		t.Error("Expected non-empty message in action response")
	}
}

func TestRestartVM(t *testing.T) {
	router, _, mockClient := setupTestRouter()

	// Add a test VM
	mockClient.CreateVM(context.Background(), models.CreateVMParams{
		Name:   "test-vm",
		Image:  "ubuntu-24",
		Flavor: "m1.small",
	})

	req, _ := http.NewRequest("POST", "/api/v1/vms/test-vm/restart", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusAccepted {
		t.Errorf("Expected status %d, got %d", http.StatusAccepted, w.Code)
	}

	var response models.VMActionResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Message == "" {
		t.Error("Expected non-empty message in action response")
	}
}

func TestGetVMConsole(t *testing.T) {
	router, _, mockClient := setupTestRouter()

	// Add a test VM
	mockClient.CreateVM(context.Background(), models.CreateVMParams{
		Name:   "test-vm",
		Image:  "ubuntu-24",
		Flavor: "m1.small",
	})

	req, _ := http.NewRequest("GET", "/api/v1/vms/test-vm/console", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response models.ConsoleResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	if response.Type == "" || response.URL == "" || response.Token == "" {
		t.Error("Console response missing required fields")
	}
}

func TestConvertStatus(t *testing.T) {
	handler := &V1Handler{}

	tests := []struct {
		internalStatus string
		v1Status       string
	}{
		{models.InternalStatusRunning, models.VMStatusRunning},
		{models.InternalStatusStarting, models.VMStatusStarting},
		{models.InternalStatusStopped, models.VMStatusStopped},
		{models.InternalStatusStopping, models.VMStatusStopping},
		{models.InternalStatusError, models.VMStatusError},
		{"unknown", models.VMStatusUnknown},
	}

	for _, test := range tests {
		result := handler.convertInternalStatus(test.internalStatus)
		if result != test.v1Status {
			t.Errorf("convertInternalStatus(%s) = %s, want %s", test.internalStatus, result, test.v1Status)
		}
	}
}

func TestMatchesLabel(t *testing.T) {
	handler := &V1Handler{}

	metadata := map[string]string{
		"env":     "prod",
		"project": "webapp",
	}

	tests := []struct {
		filter   string
		expected bool
	}{
		{"env=prod", true},
		{"env=dev", false},
		{"project=webapp", true},
		{"project=api", false},
		{"nonexistent=value", false},
		{"invalid-filter", false},
	}

	for _, test := range tests {
		result := handler.matchesLabel(metadata, test.filter)
		if result != test.expected {
			t.Errorf("matchesLabel(%v, %s) = %v, want %v", metadata, test.filter, result, test.expected)
		}
	}
}