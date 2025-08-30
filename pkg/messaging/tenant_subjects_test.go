package messaging

import (
	"context"
	"testing"
)

func TestTenantSubjectBuilder_BasicSubjects(t *testing.T) {
	builder := NewTenantSubjectBuilder()
	tenantID := "tenant-123"

	tests := []struct {
		name     string
		method   func() string
		expected string
	}{
		{
			name:     "TenantWorkflowIn",
			method:   func() string { return builder.TenantWorkflowIn(tenantID, "workflow-456") },
			expected: "tenant-123.workflows.workflow-456.in",
		},
		{
			name:     "TenantWorkflowOut",
			method:   func() string { return builder.TenantWorkflowOut(tenantID, "workflow-456") },
			expected: "tenant-123.workflows.workflow-456.out",
		},
		{
			name:     "TenantAgentIn",
			method:   func() string { return builder.TenantAgentIn(tenantID, "agent-789") },
			expected: "tenant-123.agents.agent-789.in",
		},
		{
			name:     "TenantAgentOut",
			method:   func() string { return builder.TenantAgentOut(tenantID, "agent-789") },
			expected: "tenant-123.agents.agent-789.out",
		},
		{
			name:     "TenantToolsCalls",
			method:   func() string { return builder.TenantToolsCalls(tenantID) },
			expected: "tenant-123.tools.calls",
		},
		{
			name:     "TenantToolsAudit",
			method:   func() string { return builder.TenantToolsAudit(tenantID) },
			expected: "tenant-123.tools.audit",
		},
		{
			name:     "TenantSystemControl",
			method:   func() string { return builder.TenantSystemControl(tenantID) },
			expected: "tenant-123.system.control",
		},
		{
			name:     "TenantSystemHealth",
			method:   func() string { return builder.TenantSystemHealth(tenantID) },
			expected: "tenant-123.system.health",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := tt.method()
			if actual != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, actual)
			}
		})
	}
}

func TestTenantSubjectBuilder_ContextAware(t *testing.T) {
	builder := NewTenantSubjectBuilder()

	// Create context with tenant information
	ctx := context.Background()
	tenantCtx := &TenantContext{
		TenantID:   "tenant-123",
		TenantName: "Test Tenant",
	}
	ctx = context.WithValue(ctx, "tenant_context", tenantCtx)

	tests := []struct {
		name     string
		method   func() (string, error)
		expected string
	}{
		{
			name:     "WorkflowInFromContext",
			method:   func() (string, error) { return builder.WorkflowInFromContext(ctx, "workflow-456") },
			expected: "tenant-123.workflows.workflow-456.in",
		},
		{
			name:     "WorkflowOutFromContext",
			method:   func() (string, error) { return builder.WorkflowOutFromContext(ctx, "workflow-456") },
			expected: "tenant-123.workflows.workflow-456.out",
		},
		{
			name:     "AgentInFromContext",
			method:   func() (string, error) { return builder.AgentInFromContext(ctx, "agent-789") },
			expected: "tenant-123.agents.agent-789.in",
		},
		{
			name:     "AgentOutFromContext",
			method:   func() (string, error) { return builder.AgentOutFromContext(ctx, "agent-789") },
			expected: "tenant-123.agents.agent-789.out",
		},
		{
			name:     "ToolsCallsFromContext",
			method:   func() (string, error) { return builder.ToolsCallsFromContext(ctx) },
			expected: "tenant-123.tools.calls",
		},
		{
			name:     "ToolsAuditFromContext",
			method:   func() (string, error) { return builder.ToolsAuditFromContext(ctx) },
			expected: "tenant-123.tools.audit",
		},
		{
			name:     "SystemControlFromContext",
			method:   func() (string, error) { return builder.SystemControlFromContext(ctx) },
			expected: "tenant-123.system.control",
		},
		{
			name:     "SystemHealthFromContext",
			method:   func() (string, error) { return builder.SystemHealthFromContext(ctx) },
			expected: "tenant-123.system.health",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual, err := tt.method()
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if actual != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, actual)
			}
		})
	}
}

func TestTenantSubjectBuilder_ValidateTenantSubject(t *testing.T) {
	builder := NewTenantSubjectBuilder()

	validSubjects := []string{
		"12345678-1234-1234-1234-123456789012.workflows.workflow-456.in",
		"87654321-4321-4321-4321-210987654321.agents.agent-789.out",
		"abcdefgh-abcd-abcd-abcd-abcdefghijkl.tools.calls",
		"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee.system.health",
	}

	invalidSubjects := []string{
		"workflows.workflow-456.in",                // Missing tenant ID
		"invalid-tenant.workflows.workflow-456.in", // Invalid tenant ID format
		"tenant-123",                 // Too few parts
		"",                           // Empty subject
		".workflows.workflow-456.in", // Empty tenant ID
	}

	for _, subject := range validSubjects {
		if err := builder.ValidateTenantSubject(subject); err != nil {
			t.Errorf("Expected valid subject '%s' to pass validation, got error: %v", subject, err)
		}
	}

	for _, subject := range invalidSubjects {
		if err := builder.ValidateTenantSubject(subject); err == nil {
			t.Errorf("Expected invalid subject '%s' to fail validation", subject)
		}
	}
}

func TestTenantSubjectBuilder_ExtractTenantFromSubject(t *testing.T) {
	builder := NewTenantSubjectBuilder()

	tests := []struct {
		name        string
		subject     string
		expectedID  string
		expectError bool
	}{
		{
			name:        "Valid workflow subject",
			subject:     "12345678-1234-1234-1234-123456789012.workflows.workflow-456.in",
			expectedID:  "12345678-1234-1234-1234-123456789012",
			expectError: false,
		},
		{
			name:        "Valid agent subject",
			subject:     "87654321-4321-4321-4321-210987654321.agents.agent-789.out",
			expectedID:  "87654321-4321-4321-4321-210987654321",
			expectError: false,
		},
		{
			name:        "Invalid subject format",
			subject:     "workflows.workflow-456.in",
			expectedID:  "",
			expectError: true,
		},
		{
			name:        "Empty subject",
			subject:     "",
			expectedID:  "",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualID, err := builder.ExtractTenantFromSubject(tt.subject)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
				return
			}

			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
				return
			}

			if actualID != tt.expectedID {
				t.Errorf("Expected tenant ID '%s', got '%s'", tt.expectedID, actualID)
			}
		})
	}
}

func TestTenantSubjectBuilder_IsTenantSubject(t *testing.T) {
	builder := NewTenantSubjectBuilder()

	tenantSubjects := []string{
		"12345678-1234-1234-1234-123456789012.workflows.workflow-456.in",
		"87654321-4321-4321-4321-210987654321.agents.agent-789.out",
	}

	nonTenantSubjects := []string{
		"workflows.workflow-456.in",
		"agents.agent-789.out",
		"invalid-format",
		"",
	}

	for _, subject := range tenantSubjects {
		if !builder.IsTenantSubject(subject) {
			t.Errorf("Expected '%s' to be identified as tenant subject", subject)
		}
	}

	for _, subject := range nonTenantSubjects {
		if builder.IsTenantSubject(subject) {
			t.Errorf("Expected '%s' to not be identified as tenant subject", subject)
		}
	}
}

func TestTenantSubjectBuilder_WildcardSubjects(t *testing.T) {
	builder := NewTenantSubjectBuilder()
	tenantID := "12345678-1234-1234-1234-123456789012"

	tests := []struct {
		name     string
		method   func() string
		expected string
	}{
		{
			name:     "BuildTenantWorkflowWildcard",
			method:   func() string { return builder.BuildTenantWorkflowWildcard(tenantID) },
			expected: "12345678-1234-1234-1234-123456789012.workflows.*",
		},
		{
			name:     "BuildTenantAgentWildcard",
			method:   func() string { return builder.BuildTenantAgentWildcard(tenantID) },
			expected: "12345678-1234-1234-1234-123456789012.agents.*",
		},
		{
			name:     "BuildTenantToolsWildcard",
			method:   func() string { return builder.BuildTenantToolsWildcard(tenantID) },
			expected: "12345678-1234-1234-1234-123456789012.tools.*",
		},
		{
			name:     "BuildTenantSystemWildcard",
			method:   func() string { return builder.BuildTenantSystemWildcard(tenantID) },
			expected: "12345678-1234-1234-1234-123456789012.system.*",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := tt.method()
			if actual != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, actual)
			}
		})
	}
}

func TestTenantSubjectBuilder_ValidateSubjectTenantAccess(t *testing.T) {
	builder := NewTenantSubjectBuilder()

	// Create context with tenant information
	ctx := context.Background()
	tenantCtx := &TenantContext{
		TenantID:   "12345678-1234-1234-1234-123456789012",
		TenantName: "Test Tenant",
	}
	ctx = context.WithValue(ctx, "tenant_context", tenantCtx)

	validSubjects := []string{
		"12345678-1234-1234-1234-123456789012.workflows.workflow-456.in",
		"12345678-1234-1234-1234-123456789012.agents.agent-789.out",
		"12345678-1234-1234-1234-123456789012.tools.calls",
	}

	invalidSubjects := []string{
		"87654321-4321-4321-4321-210987654321.workflows.workflow-456.in", // Different tenant
		"workflows.workflow-456.in",                                      // No tenant prefix
		"invalid-format",                                                 // Invalid format
	}

	for _, subject := range validSubjects {
		if err := builder.ValidateSubjectTenantAccess(ctx, subject); err != nil {
			t.Errorf("Expected valid access for subject '%s', got error: %v", subject, err)
		}
	}

	for _, subject := range invalidSubjects {
		if err := builder.ValidateSubjectTenantAccess(ctx, subject); err == nil {
			t.Errorf("Expected access denial for subject '%s'", subject)
		}
	}
}

func TestTenantSubjectBuilder_FilterSubjectsByTenant(t *testing.T) {
	builder := NewTenantSubjectBuilder()

	// Create context with tenant information
	ctx := context.Background()
	tenantCtx := &TenantContext{
		TenantID:   "12345678-1234-1234-1234-123456789012",
		TenantName: "Test Tenant",
	}
	ctx = context.WithValue(ctx, "tenant_context", tenantCtx)

	allSubjects := []string{
		"12345678-1234-1234-1234-123456789012.workflows.workflow-456.in",
		"87654321-4321-4321-4321-210987654321.workflows.workflow-789.in",
		"12345678-1234-1234-1234-123456789012.agents.agent-123.out",
		"workflows.legacy-subject", // Invalid format, should be filtered out
		"12345678-1234-1234-1234-123456789012.tools.calls",
		"87654321-4321-4321-4321-210987654321.tools.audit",
	}

	expectedFiltered := []string{
		"12345678-1234-1234-1234-123456789012.workflows.workflow-456.in",
		"12345678-1234-1234-1234-123456789012.agents.agent-123.out",
		"12345678-1234-1234-1234-123456789012.tools.calls",
	}

	actualFiltered := builder.FilterSubjectsByTenant(ctx, allSubjects)

	if len(actualFiltered) != len(expectedFiltered) {
		t.Errorf("Expected %d filtered subjects, got %d", len(expectedFiltered), len(actualFiltered))
		return
	}

	for i, expected := range expectedFiltered {
		if actualFiltered[i] != expected {
			t.Errorf("Expected filtered subject[%d] = '%s', got '%s'", i, expected, actualFiltered[i])
		}
	}
}

func TestTenantSubjectBuilder_MigrationUtilities(t *testing.T) {
	builder := NewTenantSubjectBuilder()
	tenantID := "12345678-1234-1234-1234-123456789012"

	tests := []struct {
		name             string
		legacySubject    string
		expectedMigrated string
	}{
		{
			name:             "Migrate workflow subject",
			legacySubject:    "workflows.workflow-456.in",
			expectedMigrated: "12345678-1234-1234-1234-123456789012.workflows.workflow-456.in",
		},
		{
			name:             "Migrate agent subject",
			legacySubject:    "agents.agent-789.out",
			expectedMigrated: "12345678-1234-1234-1234-123456789012.agents.agent-789.out",
		},
		{
			name:             "Already tenant-scoped subject",
			legacySubject:    "12345678-1234-1234-1234-123456789012.tools.calls",
			expectedMigrated: "12345678-1234-1234-1234-123456789012.tools.calls",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualMigrated := builder.MigrateToTenantSubject(tenantID, tt.legacySubject)
			if actualMigrated != tt.expectedMigrated {
				t.Errorf("Expected migrated subject '%s', got '%s'", tt.expectedMigrated, actualMigrated)
			}
		})
	}
}

func TestTenantSubjectBuilder_StripTenantFromSubject(t *testing.T) {
	builder := NewTenantSubjectBuilder()

	tests := []struct {
		name             string
		tenantSubject    string
		expectedStripped string
		expectError      bool
	}{
		{
			name:             "Strip tenant from workflow subject",
			tenantSubject:    "12345678-1234-1234-1234-123456789012.workflows.workflow-456.in",
			expectedStripped: "workflows.workflow-456.in",
			expectError:      false,
		},
		{
			name:             "Strip tenant from agent subject",
			tenantSubject:    "87654321-4321-4321-4321-210987654321.agents.agent-789.out",
			expectedStripped: "agents.agent-789.out",
			expectError:      false,
		},
		{
			name:             "Non-tenant subject",
			tenantSubject:    "workflows.workflow-456.in",
			expectedStripped: "workflows.workflow-456.in",
			expectError:      false,
		},
		{
			name:             "Invalid subject format",
			tenantSubject:    "invalid",
			expectedStripped: "",
			expectError:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualStripped, err := builder.StripTenantFromSubject(tt.tenantSubject)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
				return
			}

			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
				return
			}

			if actualStripped != tt.expectedStripped {
				t.Errorf("Expected stripped subject '%s', got '%s'", tt.expectedStripped, actualStripped)
			}
		})
	}
}

func TestTenantSubjectBuilder_ContextPanic(t *testing.T) {
	builder := NewTenantSubjectBuilder()

	// Test that context-aware methods panic when tenant context is missing
	ctx := context.Background()

	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when tenant context is missing")
		}
	}()

	// This should panic because there's no tenant context
	_, _ = builder.WorkflowInFromContext(ctx, "workflow-456")
}

func TestTenantSubjectBuilder_BuildTenantWildcardSubject(t *testing.T) {
	builder := NewTenantSubjectBuilder()
	tenantID := "12345678-1234-1234-1234-123456789012"

	tests := []struct {
		name     string
		category string
		expected string
	}{
		{
			name:     "Workflows wildcard",
			category: "workflows",
			expected: "12345678-1234-1234-1234-123456789012.workflows.*",
		},
		{
			name:     "Agents wildcard",
			category: "agents",
			expected: "12345678-1234-1234-1234-123456789012.agents.*",
		},
		{
			name:     "Tools wildcard",
			category: "tools",
			expected: "12345678-1234-1234-1234-123456789012.tools.*",
		},
		{
			name:     "System wildcard",
			category: "system",
			expected: "12345678-1234-1234-1234-123456789012.system.*",
		},
		{
			name:     "Custom category wildcard",
			category: "custom",
			expected: "12345678-1234-1234-1234-123456789012.custom.*",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := builder.BuildTenantWildcardSubject(tenantID, tt.category)
			if actual != tt.expected {
				t.Errorf("Expected '%s', got '%s'", tt.expected, actual)
			}
		})
	}
}
