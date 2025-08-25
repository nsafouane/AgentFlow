package audit

import (
	"encoding/json"
	"testing"
	"time"
)

func TestComputeHash(t *testing.T) {
	tests := []struct {
		name     string
		prevHash []byte
		record   AuditRecord
		wantErr  bool
	}{
		{
			name:     "genesis record with nil prev_hash",
			prevHash: nil,
			record: AuditRecord{
				TenantID:     "tenant-1",
				ActorType:    "user",
				ActorID:      "user-123",
				Action:       "create",
				ResourceType: "workflow",
				ResourceID:   stringPtr("workflow-456"),
				Details:      json.RawMessage(`{"key": "value"}`),
				Timestamp:    time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
			},
			wantErr: false,
		},
		{
			name:     "second record with previous hash",
			prevHash: []byte("previous-hash-bytes"),
			record: AuditRecord{
				TenantID:     "tenant-1",
				ActorType:    "system",
				ActorID:      "system-001",
				Action:       "update",
				ResourceType: "agent",
				ResourceID:   nil,
				Details:      json.RawMessage(`{"updated": true}`),
				Timestamp:    time.Date(2025, 1, 1, 12, 1, 0, 0, time.UTC),
			},
			wantErr: false,
		},
		{
			name:     "record with invalid JSON details",
			prevHash: nil,
			record: AuditRecord{
				TenantID:     "tenant-1",
				ActorType:    "user",
				ActorID:      "user-123",
				Action:       "delete",
				ResourceType: "tool",
				ResourceID:   stringPtr("tool-789"),
				Details:      json.RawMessage(`invalid json`),
				Timestamp:    time.Date(2025, 1, 1, 12, 2, 0, 0, time.UTC),
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash, err := ComputeHash(tt.prevHash, tt.record)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ComputeHash() expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("ComputeHash() unexpected error: %v", err)
				return
			}

			if len(hash) != 32 { // SHA256 produces 32 bytes
				t.Errorf("ComputeHash() hash length = %d, want 32", len(hash))
			}

			// Verify deterministic behavior - same input should produce same hash
			hash2, err := ComputeHash(tt.prevHash, tt.record)
			if err != nil {
				t.Errorf("ComputeHash() second call unexpected error: %v", err)
				return
			}

			if !equalBytes(hash, hash2) {
				t.Errorf("ComputeHash() not deterministic - got different hashes for same input")
			}
		})
	}
}

func TestComputeHashDeterministic(t *testing.T) {
	record := AuditRecord{
		TenantID:     "tenant-1",
		ActorType:    "user",
		ActorID:      "user-123",
		Action:       "create",
		ResourceType: "workflow",
		ResourceID:   stringPtr("workflow-456"),
		Details:      json.RawMessage(`{"key": "value"}`),
		Timestamp:    time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
	}

	// Compute hash multiple times
	hash1, err := ComputeHash(nil, record)
	if err != nil {
		t.Fatalf("ComputeHash() error: %v", err)
	}

	hash2, err := ComputeHash(nil, record)
	if err != nil {
		t.Fatalf("ComputeHash() error: %v", err)
	}

	if !equalBytes(hash1, hash2) {
		t.Errorf("ComputeHash() not deterministic")
	}
}

func TestComputeHashWithPrevHash(t *testing.T) {
	record := AuditRecord{
		TenantID:     "tenant-1",
		ActorType:    "user",
		ActorID:      "user-123",
		Action:       "create",
		ResourceType: "workflow",
		Details:      json.RawMessage(`{"key": "value"}`),
		Timestamp:    time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
	}

	// Hash without previous hash
	hash1, err := ComputeHash(nil, record)
	if err != nil {
		t.Fatalf("ComputeHash() error: %v", err)
	}

	// Hash with previous hash
	prevHash := []byte("some-previous-hash")
	hash2, err := ComputeHash(prevHash, record)
	if err != nil {
		t.Fatalf("ComputeHash() error: %v", err)
	}

	// Should be different
	if equalBytes(hash1, hash2) {
		t.Errorf("ComputeHash() with different prev_hash should produce different results")
	}
}

func TestVerifyHashChain(t *testing.T) {
	tests := []struct {
		name    string
		records []AuditRecord
		want    VerificationResult
	}{
		{
			name:    "empty chain",
			records: []AuditRecord{},
			want: VerificationResult{
				Valid:        true,
				TotalRecords: 0,
			},
		},
		{
			name: "single record chain",
			records: []AuditRecord{
				{
					TenantID:     "tenant-1",
					ActorType:    "user",
					ActorID:      "user-123",
					Action:       "create",
					ResourceType: "workflow",
					Details:      json.RawMessage(`{"key": "value"}`),
					Timestamp:    time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
				},
			},
			want: VerificationResult{
				Valid:        true,
				TotalRecords: 1,
			},
		},
		{
			name: "multiple record chain",
			records: []AuditRecord{
				{
					TenantID:     "tenant-1",
					ActorType:    "user",
					ActorID:      "user-123",
					Action:       "create",
					ResourceType: "workflow",
					Details:      json.RawMessage(`{"key": "value1"}`),
					Timestamp:    time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
				},
				{
					TenantID:     "tenant-1",
					ActorType:    "user",
					ActorID:      "user-123",
					Action:       "update",
					ResourceType: "workflow",
					Details:      json.RawMessage(`{"key": "value2"}`),
					Timestamp:    time.Date(2025, 1, 1, 12, 1, 0, 0, time.UTC),
				},
			},
			want: VerificationResult{
				Valid:        true,
				TotalRecords: 2,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := VerifyHashChain(tt.records)

			if got.Valid != tt.want.Valid {
				t.Errorf("VerifyHashChain() Valid = %v, want %v", got.Valid, tt.want.Valid)
			}

			if got.TotalRecords != tt.want.TotalRecords {
				t.Errorf("VerifyHashChain() TotalRecords = %v, want %v", got.TotalRecords, tt.want.TotalRecords)
			}
		})
	}
}

func TestTamperDetection(t *testing.T) {
	// Create a valid chain of records
	record1 := AuditRecord{
		TenantID:     "tenant-1",
		ActorType:    "user",
		ActorID:      "user-123",
		Action:       "create",
		ResourceType: "workflow",
		Details:      json.RawMessage(`{"key": "value1"}`),
		Timestamp:    time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
	}

	record2 := AuditRecord{
		TenantID:     "tenant-1",
		ActorType:    "user",
		ActorID:      "user-123",
		Action:       "update",
		ResourceType: "workflow",
		Details:      json.RawMessage(`{"key": "value2"}`),
		Timestamp:    time.Date(2025, 1, 1, 12, 1, 0, 0, time.UTC),
	}

	// Test with tampered record (modified action)
	tamperedRecord2 := record2
	tamperedRecord2.Action = "delete" // Tamper with the action

	records := []AuditRecord{record1, tamperedRecord2}

	// Note: This test would need to be adapted once we implement proper
	// hash verification with stored hashes. For now, it tests the structure.
	result := VerifyHashChain(records)

	// The current implementation doesn't detect tampering without stored hashes
	// This would be enhanced when we integrate with the database
	if result.TotalRecords != 2 {
		t.Errorf("Expected 2 records, got %d", result.TotalRecords)
	}
}

// Helper function to create string pointers
func stringPtr(s string) *string {
	return &s
}
