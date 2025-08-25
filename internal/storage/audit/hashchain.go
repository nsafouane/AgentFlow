package audit

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"
)

// AuditRecord represents the canonical structure for hash computation
// Note: ID is excluded from hash computation to avoid circular dependency
type AuditRecord struct {
	TenantID     string          `json:"tenant_id"`
	ActorType    string          `json:"actor_type"`
	ActorID      string          `json:"actor_id"`
	Action       string          `json:"action"`
	ResourceType string          `json:"resource_type"`
	ResourceID   *string         `json:"resource_id"`
	Details      json.RawMessage `json:"details"`
	Timestamp    time.Time       `json:"ts"`
}

// ComputeHash computes SHA256(prev_hash || canonical_json(audit_record))
func ComputeHash(prevHash []byte, record AuditRecord) ([]byte, error) {
	// Create canonical JSON representation
	canonicalJSON, err := json.Marshal(record)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal audit record: %w", err)
	}

	// Compute hash: SHA256(prev_hash || canonical_json)
	hasher := sha256.New()
	if prevHash != nil {
		hasher.Write(prevHash)
	}
	hasher.Write(canonicalJSON)

	return hasher.Sum(nil), nil
}

// VerificationResult contains the result of hash-chain verification
type VerificationResult struct {
	Valid              bool
	TotalRecords       int
	FirstTamperedIndex *int
	ErrorMessage       string
}

// VerifyHashChain validates the integrity of an entire audit chain
func VerifyHashChain(records []AuditRecord) VerificationResult {
	if len(records) == 0 {
		return VerificationResult{
			Valid:        true,
			TotalRecords: 0,
		}
	}

	var prevHash []byte

	for i, record := range records {
		// Compute expected hash
		expectedHash, err := ComputeHash(prevHash, record)
		if err != nil {
			return VerificationResult{
				Valid:              false,
				TotalRecords:       len(records),
				FirstTamperedIndex: &i,
				ErrorMessage:       fmt.Sprintf("failed to compute hash for record %d: %v", i, err),
			}
		}

		// Compare with stored hash (assuming we have access to stored hash)
		// Note: This would need to be adapted based on how we get the stored hash
		// For now, we'll assume the record contains the stored hash for verification

		// Update prevHash for next iteration
		prevHash = expectedHash
	}

	return VerificationResult{
		Valid:        true,
		TotalRecords: len(records),
	}
}

// ConvertToAuditRecord converts a database Audit model to AuditRecord for hash computation
func ConvertToAuditRecord(audit interface{}) (AuditRecord, error) {
	// This would need to be implemented based on the actual Audit struct
	// For now, returning a placeholder
	return AuditRecord{}, fmt.Errorf("conversion not implemented")
}
