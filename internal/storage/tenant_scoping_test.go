package storage

import (
	"context"
	"strings"
	"testing"

	"github.com/agentflow/agentflow/internal/logging"
)

func TestTenantScopedDB_InjectTenantScoping(t *testing.T) {
	logger := logging.NewLogger()
	tsdb := &TenantScopedDB{
		db:     nil, // Not needed for these tests
		logger: logger,
	}

	tests := []struct {
		name          string
		query         string
		tenantID      string
		args          []interface{}
		expectedQuery string
		expectedArgs  []interface{}
		expectError   bool
	}{
		{
			name:          "SELECT with WHERE clause",
			query:         "SELECT * FROM users WHERE email = $1",
			tenantID:      "tenant-123",
			args:          []interface{}{"test@example.com"},
			expectedQuery: "SELECT * FROM users WHERE users.tenant_id = $2 AND email = $1",
			expectedArgs:  []interface{}{"test@example.com", "tenant-123"},
		},
		{
			name:          "SELECT without WHERE clause",
			query:         "SELECT * FROM workflows ORDER BY created_at",
			tenantID:      "tenant-123",
			args:          []interface{}{},
			expectedQuery: "SELECT * FROM workflows WHERE workflows.tenant_id = $1 ORDER BY created_at",
			expectedArgs:  []interface{}{"tenant-123"},
		},
		{
			name:          "UPDATE with WHERE clause",
			query:         "UPDATE agents SET name = $1 WHERE id = $2",
			tenantID:      "tenant-123",
			args:          []interface{}{"New Name", "agent-456"},
			expectedQuery: "UPDATE agents SET name = $1 WHERE tenant_id = $3 AND id = $2",
			expectedArgs:  []interface{}{"New Name", "agent-456", "tenant-123"},
		},
		{
			name:          "DELETE with WHERE clause",
			query:         "DELETE FROM tools WHERE id = $1",
			tenantID:      "tenant-123",
			args:          []interface{}{"tool-789"},
			expectedQuery: "DELETE FROM tools WHERE tenant_id = $2 AND id = $1",
			expectedArgs:  []interface{}{"tool-789", "tenant-123"},
		},
		{
			name:          "Non-multi-tenant table (plans)",
			query:         "SELECT * FROM plans WHERE workflow_id = $1",
			tenantID:      "tenant-123",
			args:          []interface{}{"workflow-123"},
			expectedQuery: "SELECT * FROM plans WHERE workflow_id = $1",
			expectedArgs:  []interface{}{"workflow-123"},
		},
		{
			name:          "Skip DDL statements",
			query:         "CREATE TABLE test (id UUID PRIMARY KEY)",
			tenantID:      "tenant-123",
			args:          []interface{}{},
			expectedQuery: "CREATE TABLE test (id UUID PRIMARY KEY)",
			expectedArgs:  []interface{}{},
		},
		{
			name:          "Skip health check queries",
			query:         "SELECT 1",
			tenantID:      "tenant-123",
			args:          []interface{}{},
			expectedQuery: "SELECT 1",
			expectedArgs:  []interface{}{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualQuery, actualArgs, err := tsdb.injectTenantScoping(tt.query, tt.tenantID, tt.args...)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
				return
			}

			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
				return
			}

			if actualQuery != tt.expectedQuery {
				t.Errorf("Expected query:\n%s\nGot:\n%s", tt.expectedQuery, actualQuery)
			}

			if len(actualArgs) != len(tt.expectedArgs) {
				t.Errorf("Expected %d args, got %d", len(tt.expectedArgs), len(actualArgs))
				return
			}

			for i, expectedArg := range tt.expectedArgs {
				if actualArgs[i] != expectedArg {
					t.Errorf("Expected arg[%d] = %v, got %v", i, expectedArg, actualArgs[i])
				}
			}
		})
	}
}

func TestTenantScopedDB_ExtractTableName(t *testing.T) {
	tsdb := &TenantScopedDB{}

	tests := []struct {
		name          string
		query         string
		expectedTable string
	}{
		{
			name:          "Simple SELECT",
			query:         "select * from users",
			expectedTable: "users",
		},
		{
			name:          "SELECT with alias",
			query:         "select u.* from users u",
			expectedTable: "users",
		},
		{
			name:          "SELECT with JOIN",
			query:         "select * from users u join roles r on u.role_id = r.id",
			expectedTable: "users",
		},
		{
			name:          "INSERT statement",
			query:         "insert into workflows (name, config) values ($1, $2)",
			expectedTable: "workflows",
		},
		{
			name:          "UPDATE statement",
			query:         "update agents set name = $1 where id = $2",
			expectedTable: "agents",
		},
		{
			name:          "DELETE statement",
			query:         "delete from tools where id = $1",
			expectedTable: "tools",
		},
		{
			name:          "Quoted table name",
			query:         "select * from \"users\"",
			expectedTable: "users",
		},
		{
			name:          "Complex SELECT with subquery",
			query:         "select * from (select * from users) u",
			expectedTable: "(select",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualTable := tsdb.extractTableName(strings.ToLower(tt.query))
			if actualTable != tt.expectedTable {
				t.Errorf("Expected table '%s', got '%s'", tt.expectedTable, actualTable)
			}
		})
	}
}

func TestTenantScopedDB_IsMultiTenantTable(t *testing.T) {
	tsdb := &TenantScopedDB{}

	multiTenantTables := []string{
		"users", "agents", "workflows", "messages", "tools",
		"audits", "budgets", "rbac_roles", "rbac_bindings",
	}

	nonMultiTenantTables := []string{
		"plans", "tenants", "unknown_table", "",
	}

	for _, table := range multiTenantTables {
		if !tsdb.isMultiTenantTable(table) {
			t.Errorf("Expected table '%s' to be multi-tenant", table)
		}
	}

	for _, table := range nonMultiTenantTables {
		if tsdb.isMultiTenantTable(table) {
			t.Errorf("Expected table '%s' to not be multi-tenant", table)
		}
	}
}

func TestTenantScopedDB_HasExistingTenantCondition(t *testing.T) {
	tsdb := &TenantScopedDB{}

	tests := []struct {
		name      string
		query     string
		tableName string
		expected  bool
	}{
		{
			name:      "Has tenant_id condition",
			query:     "select * from users where tenant_id = $1",
			tableName: "users",
			expected:  true,
		},
		{
			name:      "Has qualified tenant_id condition",
			query:     "select * from users u where u.tenant_id = $1",
			tableName: "users",
			expected:  true,
		},
		{
			name:      "No tenant_id condition",
			query:     "select * from users where email = $1",
			tableName: "users",
			expected:  false,
		},
		{
			name:      "Has tenant_id in different context",
			query:     "select tenant_id from users where email = $1",
			tableName: "users",
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actual := tsdb.hasExistingTenantCondition(strings.ToLower(tt.query), tt.tableName)
			if actual != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, actual)
			}
		})
	}
}

func TestTenantScopedDB_ShouldSkipTenantScoping(t *testing.T) {
	tsdb := &TenantScopedDB{}

	skipQueries := []string{
		"CREATE TABLE test (id UUID)",
		"DROP TABLE test",
		"ALTER TABLE test ADD COLUMN name VARCHAR(255)",
		"CREATE INDEX idx_test ON test(name)",
		"SELECT 1",
		"SELECT version()",
		"EXPLAIN SELECT * FROM users",
		"DESCRIBE users",
	}

	processQueries := []string{
		"SELECT * FROM users",
		"INSERT INTO workflows (name) VALUES ($1)",
		"UPDATE agents SET name = $1",
		"DELETE FROM tools WHERE id = $1",
	}

	for _, query := range skipQueries {
		if !tsdb.shouldSkipTenantScoping(strings.ToLower(query)) {
			t.Errorf("Expected to skip tenant scoping for query: %s", query)
		}
	}

	for _, query := range processQueries {
		if tsdb.shouldSkipTenantScoping(strings.ToLower(query)) {
			t.Errorf("Expected to process tenant scoping for query: %s", query)
		}
	}
}

func TestTenantScopedDB_InjectTenantConditionSelect(t *testing.T) {
	tsdb := &TenantScopedDB{}

	tests := []struct {
		name          string
		query         string
		tableName     string
		tenantID      string
		args          []interface{}
		expectedQuery string
		expectedArgs  []interface{}
	}{
		{
			name:          "SELECT with existing WHERE",
			query:         "SELECT * FROM users WHERE email = $1",
			tableName:     "users",
			tenantID:      "tenant-123",
			args:          []interface{}{"test@example.com"},
			expectedQuery: "SELECT * FROM users WHERE users.tenant_id = $2 AND email = $1",
			expectedArgs:  []interface{}{"test@example.com", "tenant-123"},
		},
		{
			name:          "SELECT without WHERE, with ORDER BY",
			query:         "SELECT * FROM workflows ORDER BY created_at",
			tableName:     "workflows",
			tenantID:      "tenant-123",
			args:          []interface{}{},
			expectedQuery: "SELECT * FROM workflows WHERE workflows.tenant_id = $1 ORDER BY created_at",
			expectedArgs:  []interface{}{"tenant-123"},
		},
		{
			name:          "SELECT without WHERE, with LIMIT",
			query:         "SELECT * FROM agents LIMIT 10",
			tableName:     "agents",
			tenantID:      "tenant-123",
			args:          []interface{}{},
			expectedQuery: "SELECT * FROM agents WHERE agents.tenant_id = $1 LIMIT 10",
			expectedArgs:  []interface{}{"tenant-123"},
		},
		{
			name:          "SELECT with GROUP BY",
			query:         "SELECT type, COUNT(*) FROM tools GROUP BY type",
			tableName:     "tools",
			tenantID:      "tenant-123",
			args:          []interface{}{},
			expectedQuery: "SELECT type, COUNT(*) FROM tools WHERE tools.tenant_id = $1 GROUP BY type",
			expectedArgs:  []interface{}{"tenant-123"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			actualQuery, actualArgs, err := tsdb.injectTenantConditionSelect(
				tt.query, tt.tableName, tt.tenantID, tt.args...)

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if actualQuery != tt.expectedQuery {
				t.Errorf("Expected query:\n%s\nGot:\n%s", tt.expectedQuery, actualQuery)
			}

			if len(actualArgs) != len(tt.expectedArgs) {
				t.Errorf("Expected %d args, got %d", len(tt.expectedArgs), len(actualArgs))
				return
			}

			for i, expectedArg := range tt.expectedArgs {
				if actualArgs[i] != expectedArg {
					t.Errorf("Expected arg[%d] = %v, got %v", i, expectedArg, actualArgs[i])
				}
			}
		})
	}
}

func TestTenantScopedDB_WithContext(t *testing.T) {
	logger := logging.NewLogger()
	tsdb := NewTenantScopedDB(nil, logger)

	// Create context with tenant information
	ctx := context.Background()

	// Add tenant context directly to context
	tenantCtx := &TenantContext{
		TenantID:   "tenant-123",
		TenantName: "Test Tenant",
	}
	ctx = context.WithValue(ctx, "tenant_context", tenantCtx)

	// Test that MustGetTenantIDFromContext works with the context
	tenantID := MustGetTenantIDFromContext(ctx)
	if tenantID != "tenant-123" {
		t.Errorf("Expected tenant ID 'tenant-123', got '%s'", tenantID)
	}

	// Test query scoping with context
	query := "SELECT * FROM users WHERE email = $1"
	args := []interface{}{"test@example.com"}

	scopedQuery, scopedArgs, err := tsdb.injectTenantScoping(query, tenantID, args...)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	expectedQuery := "SELECT * FROM users WHERE users.tenant_id = $2 AND email = $1"
	expectedArgs := []interface{}{"test@example.com", "tenant-123"}

	if scopedQuery != expectedQuery {
		t.Errorf("Expected query:\n%s\nGot:\n%s", expectedQuery, scopedQuery)
	}

	if len(scopedArgs) != len(expectedArgs) {
		t.Errorf("Expected %d args, got %d", len(expectedArgs), len(scopedArgs))
		return
	}

	for i, expectedArg := range expectedArgs {
		if scopedArgs[i] != expectedArg {
			t.Errorf("Expected arg[%d] = %v, got %v", i, expectedArg, scopedArgs[i])
		}
	}
}

func TestTenantScopedDB_ContextPanic(t *testing.T) {
	logger := logging.NewLogger()
	tsdb := NewTenantScopedDB(nil, logger)

	// Test that operations panic when tenant context is missing
	ctx := context.Background()

	defer func() {
		if r := recover(); r == nil {
			t.Error("Expected panic when tenant context is missing")
		}
	}()

	// This should panic because there's no tenant context
	_, _ = tsdb.QueryContext(ctx, "SELECT * FROM users", nil)
}
