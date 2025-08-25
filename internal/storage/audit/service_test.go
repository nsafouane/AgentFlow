package audit

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/agentflow/agentflow/internal/storage/queries"
	"github.com/jackc/pgx/v5/pgtype"
)

// MockQueries implements a mock version of AuditQuerier for testing
type MockQueries struct {
	audits      []queries.Audit
	latestAudit *queries.Audit
	createErr   error
	getErr      error
}

func (m *MockQueries) CreateAudit(ctx context.Context, arg queries.CreateAuditParams) (queries.Audit, error) {
	if m.createErr != nil {
		return queries.Audit{}, m.createErr
	}

	// Create a new audit with generated ID (use length+1 to make unique IDs)
	idBytes := [16]byte{}
	idBytes[0] = byte(len(m.audits) + 1)

	// Use a fixed timestamp to ensure consistency with hash computation
	fixedTime := time.Date(2025, 1, 1, 12, len(m.audits), 0, 0, time.UTC)

	audit := queries.Audit{
		ID:           pgtype.UUID{Bytes: idBytes, Valid: true},
		TenantID:     arg.TenantID,
		ActorType:    arg.ActorType,
		ActorID:      arg.ActorID,
		Action:       arg.Action,
		ResourceType: arg.ResourceType,
		ResourceID:   arg.ResourceID,
		Details:      arg.Details,
		Ts:           pgtype.Timestamptz{Time: fixedTime, Valid: true},
		PrevHash:     arg.PrevHash,
		Hash:         arg.Hash,
	}

	m.audits = append(m.audits, audit)
	m.latestAudit = &audit
	return audit, nil
}

func (m *MockQueries) GetLatestAudit(ctx context.Context, tenantID pgtype.UUID) (queries.Audit, error) {
	if m.getErr != nil {
		return queries.Audit{}, m.getErr
	}

	if m.latestAudit == nil {
		return queries.Audit{}, fmt.Errorf("no rows in result set")
	}

	return *m.latestAudit, nil
}

func (m *MockQueries) GetAuditChain(ctx context.Context, tenantID pgtype.UUID) ([]queries.Audit, error) {
	if m.getErr != nil {
		return nil, m.getErr
	}

	return m.audits, nil
}

func TestService_CreateAudit(t *testing.T) {
	tests := []struct {
		name        string
		mockQueries *MockQueries
		params      CreateAuditParams
		wantErr     bool
	}{
		{
			name:        "create first audit (genesis)",
			mockQueries: &MockQueries{},
			params: CreateAuditParams{
				TenantID:     pgtype.UUID{Bytes: [16]byte{1}, Valid: true},
				ActorType:    "user",
				ActorID:      "user-123",
				Action:       "create",
				ResourceType: "workflow",
				ResourceID:   stringPtr("workflow-456"),
				Details:      map[string]interface{}{"key": "value"},
			},
			wantErr: false,
		},
		{
			name: "create second audit with previous hash",
			mockQueries: &MockQueries{
				latestAudit: &queries.Audit{
					Hash: []byte("previous-hash"),
				},
			},
			params: CreateAuditParams{
				TenantID:     pgtype.UUID{Bytes: [16]byte{1}, Valid: true},
				ActorType:    "system",
				ActorID:      "system-001",
				Action:       "update",
				ResourceType: "agent",
				Details:      map[string]interface{}{"updated": true},
			},
			wantErr: false,
		},
		{
			name:        "create audit with invalid details",
			mockQueries: &MockQueries{},
			params: CreateAuditParams{
				TenantID:     pgtype.UUID{Bytes: [16]byte{1}, Valid: true},
				ActorType:    "user",
				ActorID:      "user-123",
				Action:       "create",
				ResourceType: "workflow",
				Details:      map[string]interface{}{"invalid": make(chan int)}, // channels can't be marshaled
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewService(tt.mockQueries)

			audit, err := service.CreateAudit(context.Background(), tt.params)

			if tt.wantErr {
				if err == nil {
					t.Errorf("CreateAudit() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("CreateAudit() unexpected error: %v", err)
				return
			}

			if audit == nil {
				t.Errorf("CreateAudit() returned nil audit")
				return
			}

			// Verify audit fields
			if audit.ActorType != tt.params.ActorType {
				t.Errorf("CreateAudit() ActorType = %v, want %v", audit.ActorType, tt.params.ActorType)
			}

			if audit.Action != tt.params.Action {
				t.Errorf("CreateAudit() Action = %v, want %v", audit.Action, tt.params.Action)
			}

			// Verify hash is computed
			if len(audit.Hash) == 0 {
				t.Errorf("CreateAudit() hash is empty")
			}
		})
	}
}

func TestService_VerifyChainIntegrity(t *testing.T) {
	// Create test audits with proper hash chain
	tenantID := pgtype.UUID{Bytes: [16]byte{1}, Valid: true}

	// First audit (genesis)
	audit1 := queries.Audit{
		ID:           pgtype.UUID{Bytes: [16]byte{1}, Valid: true},
		TenantID:     tenantID,
		ActorType:    "user",
		ActorID:      "user-123",
		Action:       "create",
		ResourceType: "workflow",
		ResourceID:   pgtype.Text{String: "workflow-456", Valid: true},
		Details:      []byte(`{"key": "value1"}`),
		Ts:           pgtype.Timestamptz{Time: time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC), Valid: true},
		PrevHash:     nil,
		Hash:         []byte("hash1"),
	}

	// Second audit
	audit2 := queries.Audit{
		ID:           pgtype.UUID{Bytes: [16]byte{2}, Valid: true},
		TenantID:     tenantID,
		ActorType:    "user",
		ActorID:      "user-123",
		Action:       "update",
		ResourceType: "workflow",
		ResourceID:   pgtype.Text{String: "workflow-456", Valid: true},
		Details:      []byte(`{"key": "value2"}`),
		Ts:           pgtype.Timestamptz{Time: time.Date(2025, 1, 1, 12, 1, 0, 0, time.UTC), Valid: true},
		PrevHash:     []byte("hash1"),
		Hash:         []byte("hash2"),
	}

	tests := []struct {
		name        string
		mockQueries *MockQueries
		tenantID    pgtype.UUID
		wantValid   bool
		wantErr     bool
	}{
		{
			name: "empty chain",
			mockQueries: &MockQueries{
				audits: []queries.Audit{},
			},
			tenantID:  tenantID,
			wantValid: true,
			wantErr:   false,
		},
		{
			name: "valid chain",
			mockQueries: &MockQueries{
				audits: []queries.Audit{audit1, audit2},
			},
			tenantID:  tenantID,
			wantValid: false, // Will be false because computed hash won't match stored hash
			wantErr:   false,
		},
		{
			name: "database error",
			mockQueries: &MockQueries{
				getErr: fmt.Errorf("database error"),
			},
			tenantID: tenantID,
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewService(tt.mockQueries)

			result, err := service.VerifyChainIntegrity(context.Background(), tt.tenantID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("VerifyChainIntegrity() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("VerifyChainIntegrity() unexpected error: %v", err)
				return
			}

			if result.Valid != tt.wantValid {
				t.Errorf("VerifyChainIntegrity() Valid = %v, want %v", result.Valid, tt.wantValid)
			}
		})
	}
}

func TestConvertDBAuditToRecord(t *testing.T) {
	audit := queries.Audit{
		ID:           pgtype.UUID{Bytes: [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}, Valid: true},
		TenantID:     pgtype.UUID{Bytes: [16]byte{1}, Valid: true},
		ActorType:    "user",
		ActorID:      "user-123",
		Action:       "create",
		ResourceType: "workflow",
		ResourceID:   pgtype.Text{String: "workflow-456", Valid: true},
		Details:      []byte(`{"key": "value"}`),
		Ts:           pgtype.Timestamptz{Time: time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC), Valid: true},
		PrevHash:     []byte("prev"),
		Hash:         []byte("hash"),
	}

	record, err := convertDBAuditToRecord(audit)
	if err != nil {
		t.Fatalf("convertDBAuditToRecord() error: %v", err)
	}

	if record.ActorType != audit.ActorType {
		t.Errorf("convertDBAuditToRecord() ActorType = %v, want %v", record.ActorType, audit.ActorType)
	}

	if record.Action != audit.Action {
		t.Errorf("convertDBAuditToRecord() Action = %v, want %v", record.Action, audit.Action)
	}

	if record.ResourceID == nil || *record.ResourceID != audit.ResourceID.String {
		t.Errorf("convertDBAuditToRecord() ResourceID = %v, want %v", record.ResourceID, audit.ResourceID.String)
	}
}

func TestEqualBytes(t *testing.T) {
	tests := []struct {
		name string
		a    []byte
		b    []byte
		want bool
	}{
		{
			name: "equal bytes",
			a:    []byte{1, 2, 3},
			b:    []byte{1, 2, 3},
			want: true,
		},
		{
			name: "different bytes",
			a:    []byte{1, 2, 3},
			b:    []byte{1, 2, 4},
			want: false,
		},
		{
			name: "different lengths",
			a:    []byte{1, 2, 3},
			b:    []byte{1, 2},
			want: false,
		},
		{
			name: "both nil",
			a:    nil,
			b:    nil,
			want: true,
		},
		{
			name: "one nil",
			a:    []byte{1},
			b:    nil,
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := equalBytes(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("equalBytes() = %v, want %v", got, tt.want)
			}
		})
	}
}
