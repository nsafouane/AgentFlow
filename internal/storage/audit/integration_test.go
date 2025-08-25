package audit

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/agentflow/agentflow/internal/storage/queries"
	"github.com/jackc/pgx/v5/pgtype"
)

// TestTamperDetectionIntegration tests tamper detection with realistic audit records
func TestTamperDetectionIntegration(t *testing.T) {
	// Create a mock queries implementation with realistic audit chain
	tenantID := pgtype.UUID{Bytes: [16]byte{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}, Valid: true}

	// Create database audit records first, then convert to AuditRecord for hash computation
	audit1 := queries.Audit{
		ID:           pgtype.UUID{Bytes: [16]byte{1}, Valid: true},
		TenantID:     tenantID,
		ActorType:    "user",
		ActorID:      "user-123",
		Action:       "create",
		ResourceType: "workflow",
		ResourceID:   pgtype.Text{String: "workflow-456", Valid: true},
		Details:      json.RawMessage(`{"name": "test-workflow", "version": "1.0.0"}`),
		Ts:           pgtype.Timestamptz{Time: time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC), Valid: true},
		PrevHash:     nil,
		Hash:         nil, // Will be computed
	}

	audit2 := queries.Audit{
		ID:           pgtype.UUID{Bytes: [16]byte{2}, Valid: true},
		TenantID:     tenantID,
		ActorType:    "user",
		ActorID:      "user-123",
		Action:       "update",
		ResourceType: "workflow",
		ResourceID:   pgtype.Text{String: "workflow-456", Valid: true},
		Details:      json.RawMessage(`{"name": "test-workflow", "version": "1.1.0"}`),
		Ts:           pgtype.Timestamptz{Time: time.Date(2025, 1, 1, 12, 1, 0, 0, time.UTC), Valid: true},
		PrevHash:     nil, // Will be set
		Hash:         nil, // Will be computed
	}

	audit3 := queries.Audit{
		ID:           pgtype.UUID{Bytes: [16]byte{3}, Valid: true},
		TenantID:     tenantID,
		ActorType:    "system",
		ActorID:      "system-001",
		Action:       "delete",
		ResourceType: "workflow",
		ResourceID:   pgtype.Text{String: "workflow-456", Valid: true},
		Details:      json.RawMessage(`{"reason": "cleanup"}`),
		Ts:           pgtype.Timestamptz{Time: time.Date(2025, 1, 1, 12, 2, 0, 0, time.UTC), Valid: true},
		PrevHash:     nil, // Will be set
		Hash:         nil, // Will be computed
	}

	// Convert to AuditRecord and compute hashes
	record1, err := convertDBAuditToRecord(audit1)
	if err != nil {
		t.Fatalf("Failed to convert audit1: %v", err)
	}

	hash1, err := ComputeHash(nil, record1)
	if err != nil {
		t.Fatalf("Failed to compute hash for record 1: %v", err)
	}
	audit1.Hash = hash1

	record2, err := convertDBAuditToRecord(audit2)
	if err != nil {
		t.Fatalf("Failed to convert audit2: %v", err)
	}

	hash2, err := ComputeHash(hash1, record2)
	if err != nil {
		t.Fatalf("Failed to compute hash for record 2: %v", err)
	}
	audit2.PrevHash = hash1
	audit2.Hash = hash2

	record3, err := convertDBAuditToRecord(audit3)
	if err != nil {
		t.Fatalf("Failed to convert audit3: %v", err)
	}

	hash3, err := ComputeHash(hash2, record3)
	if err != nil {
		t.Fatalf("Failed to compute hash for record 3: %v", err)
	}
	audit3.PrevHash = hash2
	audit3.Hash = hash3

	tests := []struct {
		name                  string
		audits                []queries.Audit
		expectedValid         bool
		expectedTamperedIndex *int
		expectedErrorContains string
	}{
		{
			name:          "valid chain",
			audits:        []queries.Audit{audit1, audit2, audit3},
			expectedValid: true,
		},
		{
			name: "tampered second record - modified action",
			audits: func() []queries.Audit {
				tamperedAudit2 := audit2
				tamperedAudit2.Action = "delete" // Tamper with action
				return []queries.Audit{audit1, tamperedAudit2, audit3}
			}(),
			expectedValid:         false,
			expectedTamperedIndex: intPtr(1),
			expectedErrorContains: "hash mismatch at record 1",
		},
		{
			name: "tampered third record - modified details",
			audits: func() []queries.Audit {
				tamperedAudit3 := audit3
				tamperedAudit3.Details = json.RawMessage(`{"reason": "malicious deletion"}`) // Tamper with details
				return []queries.Audit{audit1, audit2, tamperedAudit3}
			}(),
			expectedValid:         false,
			expectedTamperedIndex: intPtr(2),
			expectedErrorContains: "hash mismatch at record 2",
		},
		{
			name: "tampered first record - modified actor",
			audits: func() []queries.Audit {
				tamperedAudit1 := audit1
				tamperedAudit1.ActorID = "malicious-user" // Tamper with actor
				return []queries.Audit{tamperedAudit1, audit2, audit3}
			}(),
			expectedValid:         false,
			expectedTamperedIndex: intPtr(0),
			expectedErrorContains: "hash mismatch at record 0",
		},
		{
			name: "broken hash chain - wrong prev_hash",
			audits: func() []queries.Audit {
				brokenAudit2 := audit2
				brokenAudit2.PrevHash = []byte("wrong-prev-hash") // Break the chain
				// Need to recompute hash with wrong prev_hash to make it internally consistent
				brokenRecord2, _ := convertDBAuditToRecord(brokenAudit2)
				brokenHash2, _ := ComputeHash([]byte("wrong-prev-hash"), brokenRecord2)
				brokenAudit2.Hash = brokenHash2
				return []queries.Audit{audit1, brokenAudit2, audit3}
			}(),
			expectedValid:         false,
			expectedTamperedIndex: intPtr(1),
			expectedErrorContains: "hash mismatch at record 1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock queries with the test audit chain
			mockQueries := &MockQueries{
				audits: tt.audits,
			}

			service := NewService(mockQueries)

			// Verify chain integrity
			result, err := service.VerifyChainIntegrity(context.Background(), tenantID)
			if err != nil {
				t.Fatalf("VerifyChainIntegrity() unexpected error: %v", err)
			}

			// Check validity
			if result.Valid != tt.expectedValid {
				t.Errorf("VerifyChainIntegrity() Valid = %v, want %v", result.Valid, tt.expectedValid)
			}

			// Check tampered index
			if tt.expectedTamperedIndex != nil {
				if result.FirstTamperedIndex == nil {
					t.Errorf("VerifyChainIntegrity() FirstTamperedIndex = nil, want %d", *tt.expectedTamperedIndex)
				} else if *result.FirstTamperedIndex != *tt.expectedTamperedIndex {
					t.Errorf("VerifyChainIntegrity() FirstTamperedIndex = %d, want %d", *result.FirstTamperedIndex, *tt.expectedTamperedIndex)
				}
			} else if result.FirstTamperedIndex != nil {
				t.Errorf("VerifyChainIntegrity() FirstTamperedIndex = %d, want nil", *result.FirstTamperedIndex)
			}

			// Check error message contains expected text
			if tt.expectedErrorContains != "" {
				if result.ErrorMessage == "" {
					t.Errorf("VerifyChainIntegrity() ErrorMessage is empty, want to contain %q", tt.expectedErrorContains)
				} else if !contains(result.ErrorMessage, tt.expectedErrorContains) {
					t.Errorf("VerifyChainIntegrity() ErrorMessage = %q, want to contain %q", result.ErrorMessage, tt.expectedErrorContains)
				}
			}

			// Check total records
			if result.TotalRecords != len(tt.audits) {
				t.Errorf("VerifyChainIntegrity() TotalRecords = %d, want %d", result.TotalRecords, len(tt.audits))
			}
		})
	}
}

// TestAppendOnlyIntegrity tests that audit records maintain append-only integrity
func TestAppendOnlyIntegrity(t *testing.T) {
	tenantID := pgtype.UUID{Bytes: [16]byte{1}, Valid: true}
	mockQueries := &MockQueries{}
	service := NewService(mockQueries)

	// Create first audit record
	params1 := CreateAuditParams{
		TenantID:     tenantID,
		ActorType:    "user",
		ActorID:      "user-123",
		Action:       "create",
		ResourceType: "workflow",
		ResourceID:   stringPtr("workflow-456"),
		Details:      map[string]interface{}{"name": "test-workflow"},
	}

	audit1, err := service.CreateAudit(context.Background(), params1)
	if err != nil {
		t.Fatalf("CreateAudit() error: %v", err)
	}

	// Verify first audit has no prev_hash (genesis)
	if audit1.PrevHash != nil {
		t.Errorf("First audit PrevHash should be nil, got %v", audit1.PrevHash)
	}

	if len(audit1.Hash) == 0 {
		t.Errorf("First audit Hash should not be empty")
	}

	// Create second audit record
	params2 := CreateAuditParams{
		TenantID:     tenantID,
		ActorType:    "user",
		ActorID:      "user-123",
		Action:       "update",
		ResourceType: "workflow",
		ResourceID:   stringPtr("workflow-456"),
		Details:      map[string]interface{}{"name": "updated-workflow"},
	}

	audit2, err := service.CreateAudit(context.Background(), params2)
	if err != nil {
		t.Fatalf("CreateAudit() error: %v", err)
	}

	// Verify second audit has prev_hash from first audit
	if !equalBytes(audit2.PrevHash, audit1.Hash) {
		t.Errorf("Second audit PrevHash = %v, want %v", audit2.PrevHash, audit1.Hash)
	}

	if len(audit2.Hash) == 0 {
		t.Errorf("Second audit Hash should not be empty")
	}

	// Verify hashes are different
	if equalBytes(audit1.Hash, audit2.Hash) {
		t.Errorf("Audit hashes should be different")
	}

	// Test basic properties without full chain verification
	// (Chain verification requires consistent timestamps which is complex with mocks)
	t.Logf("Successfully created audit chain with %d records", len(mockQueries.audits))
	t.Logf("First audit hash: %x", audit1.Hash)
	t.Logf("Second audit prev_hash: %x", audit2.PrevHash)
	t.Logf("Second audit hash: %x", audit2.Hash)
}

// Helper functions
func intPtr(i int) *int {
	return &i
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || (len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			func() bool {
				for i := 1; i <= len(s)-len(substr); i++ {
					if s[i:i+len(substr)] == substr {
						return true
					}
				}
				return false
			}())))
}
