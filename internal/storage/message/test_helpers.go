package message

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/agentflow/agentflow/internal/storage/queries"
)

// MockDB implements DBTX for testing
type MockDB struct {
	messages map[string]queries.Message
	tenants  map[string]queries.Tenant
}

func NewMockDB() *MockDB {
	return &MockDB{
		messages: make(map[string]queries.Message),
		tenants:  make(map[string]queries.Tenant),
	}
}

func (m *MockDB) Exec(ctx context.Context, query string, args ...interface{}) (pgconn.CommandTag, error) {
	return pgconn.CommandTag{}, nil
}

func (m *MockDB) Query(ctx context.Context, query string, args ...interface{}) (pgx.Rows, error) {
	return nil, nil
}

func (m *MockDB) QueryRow(ctx context.Context, query string, args ...interface{}) pgx.Row {
	return &MockRow{}
}

// MockRow implements pgx.Row for testing
type MockRow struct {
	data []interface{}
	err  error
}

func (r *MockRow) Scan(dest ...interface{}) error {
	if r.err != nil {
		return r.err
	}
	return nil
}

// MockQueries implements message-specific queries for testing
type MockQueries struct {
	messages map[string]queries.Message
	tenants  map[string]queries.Tenant
}

func NewMockQueries() *MockQueries {
	return &MockQueries{
		messages: make(map[string]queries.Message),
		tenants:  make(map[string]queries.Tenant),
	}
}

func (m *MockQueries) CreateMessage(ctx context.Context, arg queries.CreateMessageParams) (queries.Message, error) {
	msg := queries.Message{
		ID:           arg.ID,
		TenantID:     arg.TenantID,
		TraceID:      arg.TraceID,
		SpanID:       arg.SpanID,
		FromAgent:    arg.FromAgent,
		ToAgent:      arg.ToAgent,
		Type:         arg.Type,
		Payload:      arg.Payload,
		Metadata:     arg.Metadata,
		Cost:         arg.Cost,
		Ts:           arg.Ts,
		EnvelopeHash: arg.EnvelopeHash,
	}

	key := uuid.UUID(arg.ID.Bytes).String()
	m.messages[key] = msg
	return msg, nil
}

func (m *MockQueries) GetMessage(ctx context.Context, arg queries.GetMessageParams) (queries.Message, error) {
	key := uuid.UUID(arg.ID.Bytes).String()
	if msg, exists := m.messages[key]; exists {
		return msg, nil
	}
	return queries.Message{}, pgx.ErrNoRows
}

func (m *MockQueries) ListMessagesByTrace(ctx context.Context, arg queries.ListMessagesByTraceParams) ([]queries.Message, error) {
	var result []queries.Message
	for _, msg := range m.messages {
		if msg.TraceID.String == arg.TraceID.String {
			result = append(result, msg)
		}
	}
	return result, nil
}

// setupTestDB creates a mock database for testing
func setupTestDB(t *testing.T) *pgxpool.Pool {
	// For now, return nil since we'll use mock queries directly
	// In a real implementation, this would set up a test database
	return nil
}

// createTestTenant creates a test tenant using mock queries
func createTestTenant(t *testing.T, db *pgxpool.Pool) uuid.UUID {
	tenantID := uuid.New()
	// This is handled by the mock in the actual tests
	return tenantID
}
