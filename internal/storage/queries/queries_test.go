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

	// Test method signatures exist - these should compile
	var _ func(context.Context, pgtype.UUID) (MigrationBaseline, error) = queries.GetMigrationBaseline
	var _ func(context.Context) ([]MigrationBaseline, error) = queries.ListMigrationBaselines
	var _ func(context.Context, string) (MigrationBaseline, error) = queries.CreateMigrationBaseline

	t.Log("All expected sqlc-generated methods exist and compile correctly")
}

// TestMigrationBaselineStruct verifies the generated struct has expected fields
func TestMigrationBaselineStruct(t *testing.T) {
	// Test that MigrationBaseline struct has expected fields
	var uuid pgtype.UUID
	uuid.Scan("550e8400-e29b-41d4-a716-446655440000")

	baseline := MigrationBaseline{
		ID:          uuid,
		CreatedAt:   time.Now(),
		Description: "test description",
	}

	if !baseline.ID.Valid {
		t.Error("MigrationBaseline.ID field not accessible")
	}

	if baseline.CreatedAt.IsZero() {
		t.Error("MigrationBaseline.CreatedAt field not accessible")
	}

	if baseline.Description == "" {
		t.Error("MigrationBaseline.Description field not accessible")
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
