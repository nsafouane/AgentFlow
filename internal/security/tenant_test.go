package security

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/agentflow/agentflow/internal/logging"
)

func TestTenantContext_ContextHelpers(t *testing.T) {
	tenantCtx := &TenantContext{
		TenantID:   "tenant-123",
		TenantName: "Test Tenant",
		ResourceLimits: map[string]interface{}{
			"max_workflows": 100,
		},
	}

	// Test WithTenantContext and GetTenantContext
	ctx := context.Background()
	ctx = WithTenantContext(ctx, tenantCtx)

	retrievedCtx, err := GetTenantContext(ctx)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if retrievedCtx.TenantID != tenantCtx.TenantID {
		t.Errorf("Expected tenant ID '%s', got '%s'", tenantCtx.TenantID, retrievedCtx.TenantID)
	}

	if retrievedCtx.TenantName != tenantCtx.TenantName {
		t.Errorf("Expected tenant name '%s', got '%s'", tenantCtx.TenantName, retrievedCtx.TenantName)
	}

	// Test GetTenantIDFromTenantContext
	tenantID, err := GetTenantIDFromTenantContext(ctx)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}

	if tenantID != tenantCtx.TenantID {
		t.Errorf("Expected tenant ID '%s', got '%s'", tenantCtx.TenantID, tenantID)
	}

	// Test MustGetTenantID
	retrievedTenantID := MustGetTenantID(ctx)
	if retrievedTenantID != tenantCtx.TenantID {
		t.Errorf("Expected tenant ID '%s', got '%s'", tenantCtx.TenantID, retrievedTenantID)
	}
}

func TestTenantContext_MissingContext(t *testing.T) {
	ctx := context.Background()

	// Test GetTenantContext with missing context
	_, err := GetTenantContext(ctx)
	if err == nil {
		t.Error("Expected error for missing tenant context")
	}

	if !strings.Contains(err.Error(), "tenant context not found") {
		t.Errorf("Expected 'tenant context not found' error, got: %v", err)
	}

	// Test GetTenantIDFromTenantContext with missing context
	_, err = GetTenantIDFromTenantContext(ctx)
	if err == nil {
		t.Error("Expected error for missing tenant context")
	}

	// Test MustGetTenantID with missing context (should panic)
	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic for missing tenant context")
		}
	}()

	MustGetTenantID(ctx)
}

func TestTenantIsolationMiddleware_PublicEndpoints(t *testing.T) {
	logger := logging.NewLogger()
	middleware := NewTenantIsolationMiddleware(logger, nil)

	publicEndpoints := []string{
		"/health",
		"/api/v1/health",
		"/",
		"/api",
		"/api/v1/auth/token",
	}

	for _, endpoint := range publicEndpoints {
		if !middleware.isPublicEndpoint(endpoint) {
			t.Errorf("Expected endpoint '%s' to be public", endpoint)
		}
	}

	privateEndpoints := []string{
		"/api/v1/workflows",
		"/api/v1/agents",
		"/api/v1/tools",
		"/api/v1/users",
	}

	for _, endpoint := range privateEndpoints {
		if middleware.isPublicEndpoint(endpoint) {
			t.Errorf("Expected endpoint '%s' to be private", endpoint)
		}
	}
}

func TestValidateRequestTenantScope(t *testing.T) {
	logger := logging.NewLogger()
	middleware := NewTenantIsolationMiddleware(logger, nil)

	tests := []struct {
		name        string
		method      string
		url         string
		headers     map[string]string
		userTenant  string
		expectError bool
	}{
		{
			name:        "Valid request no tenant params",
			method:      "GET",
			url:         "/api/v1/workflows",
			userTenant:  "tenant-123",
			expectError: false,
		},
		{
			name:        "Valid request matching query tenant",
			method:      "GET",
			url:         "/api/v1/workflows?tenant_id=tenant-123",
			userTenant:  "tenant-123",
			expectError: false,
		},
		{
			name:        "Invalid request mismatched query tenant",
			method:      "GET",
			url:         "/api/v1/workflows?tenant_id=tenant-456",
			userTenant:  "tenant-123",
			expectError: true,
		},
		{
			name:        "Valid request matching header tenant",
			method:      "GET",
			url:         "/api/v1/workflows",
			headers:     map[string]string{"X-Tenant-ID": "tenant-123"},
			userTenant:  "tenant-123",
			expectError: false,
		},
		{
			name:        "Invalid request mismatched header tenant",
			method:      "GET",
			url:         "/api/v1/workflows",
			headers:     map[string]string{"X-Tenant-ID": "tenant-456"},
			userTenant:  "tenant-123",
			expectError: true,
		},
		{
			name:        "Valid request matching path tenant",
			method:      "GET",
			url:         "/api/v1/tenants/12345678-1234-1234-1234-123456789123/workflows",
			userTenant:  "12345678-1234-1234-1234-123456789123",
			expectError: false,
		},
		{
			name:        "Invalid request mismatched path tenant",
			method:      "GET",
			url:         "/api/v1/tenants/12345678-1234-1234-1234-123456789456/workflows",
			userTenant:  "12345678-1234-1234-1234-123456789123",
			expectError: true,
		},
		{
			name:        "Valid request path with non-UUID",
			method:      "GET",
			url:         "/api/v1/tenants/current/workflows",
			userTenant:  "tenant-123",
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.url, nil)

			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			err := middleware.validateRequestTenantScope(req, tt.userTenant)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}

			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestTenantIsolationMiddleware_WriteError(t *testing.T) {
	logger := logging.NewLogger()
	middleware := NewTenantIsolationMiddleware(logger, nil)

	rr := httptest.NewRecorder()
	middleware.writeError(rr, "test_error", "Test error message", 400)

	if rr.Code != 400 {
		t.Errorf("Expected status 400, got %d", rr.Code)
	}

	var response map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
		t.Fatalf("Failed to decode response: %v", err)
	}

	if response["success"] != false {
		t.Error("Expected success=false")
	}

	errorObj := response["error"].(map[string]interface{})
	if errorObj["code"] != "test_error" {
		t.Errorf("Expected error code 'test_error', got '%s'", errorObj["code"])
	}

	if errorObj["message"] != "Test error message" {
		t.Errorf("Expected error message 'Test error message', got '%s'", errorObj["message"])
	}
}
