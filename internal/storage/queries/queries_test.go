package queries

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

// TestSQLCGeneration verifies that sqlc-generated code compiles and has expected interfaces
func TestSQLCGeneration(t *testing.T) {
	// This test ensures that sqlc-generated code compiles correctly
	// We don't need a real database connection for compilation testing

	// Test that we can create a Queries instance with mock
	mock := &MockDB{}
	queries := New(mock)

	if queries == nil {
		t.Error("Failed to create Queries instance")
	}

	// Test core table method signatures exist - these should compile
	var _ func(context.Context, CreateTenantParams) (Tenant, error) = queries.CreateTenant
	var _ func(context.Context, pgtype.UUID) (Tenant, error) = queries.GetTenant
	var _ func(context.Context) ([]Tenant, error) = queries.ListTenants

	var _ func(context.Context, CreateUserParams) (User, error) = queries.CreateUser
	var _ func(context.Context, GetUserParams) (User, error) = queries.GetUser

	var _ func(context.Context, CreateAgentParams) (Agent, error) = queries.CreateAgent
	var _ func(context.Context, GetAgentParams) (Agent, error) = queries.GetAgent

	var _ func(context.Context, CreateWorkflowParams) (Workflow, error) = queries.CreateWorkflow
	var _ func(context.Context, GetWorkflowParams) (Workflow, error) = queries.GetWorkflow

	var _ func(context.Context, CreateMessageParams) (Message, error) = queries.CreateMessage
	var _ func(context.Context, GetMessageParams) (Message, error) = queries.GetMessage

	var _ func(context.Context, CreateAuditParams) (Audit, error) = queries.CreateAudit
	var _ func(context.Context, GetAuditParams) (Audit, error) = queries.GetAudit

	t.Log("All expected sqlc-generated methods exist and compile correctly")
}

// TestCoreStructs verifies the generated structs have expected fields
func TestCoreStructs(t *testing.T) {
	// Test that core structs have expected fields
	var uuid pgtype.UUID
	uuid.Scan("550e8400-e29b-41d4-a716-446655440000")

	// Test Tenant struct
	tenant := Tenant{
		ID:        uuid,
		Name:      "test-tenant",
		Tier:      "free",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if !tenant.ID.Valid {
		t.Error("Tenant.ID field not accessible")
	}
	if tenant.Name == "" {
		t.Error("Tenant.Name field not accessible")
	}

	// Test User struct
	user := User{
		ID:       uuid,
		TenantID: uuid,
		Email:    "test@example.com",
		Role:     "viewer",
	}

	if !user.ID.Valid {
		t.Error("User.ID field not accessible")
	}
	if user.Email == "" {
		t.Error("User.Email field not accessible")
	}

	// Test Agent struct
	agent := Agent{
		ID:       uuid,
		TenantID: uuid,
		Name:     "test-agent",
		Type:     "worker",
	}

	if !agent.ID.Valid {
		t.Error("Agent.ID field not accessible")
	}
	if agent.Name == "" {
		t.Error("Agent.Name field not accessible")
	}
}

// TestQuerierInterface verifies that the Querier interface is properly generated
func TestQuerierInterface(t *testing.T) {
	// Test that Queries implements Querier interface
	mock := &MockDB{}
	queries := New(mock)

	var _ Querier = queries

	// Test that DBTX interface is properly defined
	var _ DBTX = mock
}

// MockDB implements DBTX for testing
type MockDB struct{}

func (m *MockDB) Exec(ctx context.Context, query string, args ...interface{}) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func (m *MockDB) Query(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error) {
	return nil, nil
}

func (m *MockDB) QueryRow(ctx context.Context, query string, args ...interface{}) pgx.Row {
	return nil
}

// TestMockDBImplementsDBTX verifies our mock implements the DBTX interface
func TestMockDBImplementsDBTX(t *testing.T) {
	mock := &MockDB{}
	var _ DBTX = mock

	// Test that we can create Queries with mock
	queries := New(mock)
	if queries == nil {
		t.Error("Failed to create Queries with mock DB")
	}
}
