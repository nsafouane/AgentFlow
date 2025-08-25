package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/agentflow/agentflow/internal/storage/queries"
	"github.com/jackc/pgx/v5/pgtype"
)

// AuditQuerier defines the interface for audit database operations
type AuditQuerier interface {
	CreateAudit(ctx context.Context, arg queries.CreateAuditParams) (queries.Audit, error)
	GetLatestAudit(ctx context.Context, tenantID pgtype.UUID) (queries.Audit, error)
	GetAuditChain(ctx context.Context, tenantID pgtype.UUID) ([]queries.Audit, error)
}

// Service provides audit operations with hash-chain integrity
type Service struct {
	queries AuditQuerier
}

// NewService creates a new audit service
func NewService(queries AuditQuerier) *Service {
	return &Service{
		queries: queries,
	}
}

// CreateAuditParams represents parameters for creating an audit record
type CreateAuditParams struct {
	TenantID     pgtype.UUID
	ActorType    string
	ActorID      string
	Action       string
	ResourceType string
	ResourceID   *string
	Details      map[string]interface{}
}

// CreateAudit creates a new audit record with proper hash-chain maintenance
func (s *Service) CreateAudit(ctx context.Context, params CreateAuditParams) (*queries.Audit, error) {
	// Get the latest audit record for this tenant to get prev_hash
	var prevHash []byte
	latestAudit, err := s.queries.GetLatestAudit(ctx, params.TenantID)
	if err != nil {
		// If no previous audit exists, prevHash remains nil (genesis record)
		if err.Error() != "no rows in result set" {
			return nil, fmt.Errorf("failed to get latest audit: %w", err)
		}
	} else {
		prevHash = latestAudit.Hash
	}

	// Marshal details to JSON
	detailsJSON, err := json.Marshal(params.Details)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal audit details: %w", err)
	}

	// Use consistent timestamp for hash computation and database insertion
	timestamp := time.Now()

	// Create audit record for hash computation
	auditRecord := AuditRecord{
		TenantID:     uuidToString(params.TenantID),
		ActorType:    params.ActorType,
		ActorID:      params.ActorID,
		Action:       params.Action,
		ResourceType: params.ResourceType,
		ResourceID:   params.ResourceID,
		Details:      detailsJSON,
		Timestamp:    timestamp,
	}

	// Compute hash
	hash, err := ComputeHash(prevHash, auditRecord)
	if err != nil {
		return nil, fmt.Errorf("failed to compute audit hash: %w", err)
	}

	// Convert ResourceID to pgtype.Text
	var resourceIDPG pgtype.Text
	if params.ResourceID != nil {
		resourceIDPG = pgtype.Text{String: *params.ResourceID, Valid: true}
	}

	// Insert audit record with computed hash
	createParams := queries.CreateAuditParams{
		TenantID:     params.TenantID,
		ActorType:    params.ActorType,
		ActorID:      params.ActorID,
		Action:       params.Action,
		ResourceType: params.ResourceType,
		ResourceID:   resourceIDPG,
		Details:      detailsJSON,
		PrevHash:     prevHash,
		Hash:         hash,
	}

	audit, err := s.queries.CreateAudit(ctx, createParams)
	if err != nil {
		return nil, fmt.Errorf("failed to create audit record: %w", err)
	}

	return &audit, nil
}

// VerifyChainIntegrity verifies the hash-chain integrity for a tenant
func (s *Service) VerifyChainIntegrity(ctx context.Context, tenantID pgtype.UUID) (VerificationResult, error) {
	// Get all audit records for the tenant in chronological order
	audits, err := s.queries.GetAuditChain(ctx, tenantID)
	if err != nil {
		return VerificationResult{
			Valid:        false,
			ErrorMessage: fmt.Sprintf("failed to retrieve audit chain: %v", err),
		}, err
	}

	// Convert to AuditRecord format for verification
	records := make([]AuditRecord, len(audits))
	for i, audit := range audits {
		record, err := convertDBAuditToRecord(audit)
		if err != nil {
			return VerificationResult{
				Valid:              false,
				TotalRecords:       len(audits),
				FirstTamperedIndex: &i,
				ErrorMessage:       fmt.Sprintf("failed to convert audit record %d: %v", i, err),
			}, err
		}
		records[i] = record
	}

	// Verify hash chain with stored hashes
	return s.verifyHashChainWithStoredHashes(records, audits), nil
}

// verifyHashChainWithStoredHashes verifies the chain by comparing computed vs stored hashes
func (s *Service) verifyHashChainWithStoredHashes(records []AuditRecord, audits []queries.Audit) VerificationResult {
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

		// Compare with stored hash
		storedHash := audits[i].Hash
		if !equalBytes(expectedHash, storedHash) {
			return VerificationResult{
				Valid:              false,
				TotalRecords:       len(records),
				FirstTamperedIndex: &i,
				ErrorMessage:       fmt.Sprintf("hash mismatch at record %d", i),
			}
		}

		// Update prevHash for next iteration
		prevHash = storedHash
	}

	return VerificationResult{
		Valid:        true,
		TotalRecords: len(records),
	}
}

// convertDBAuditToRecord converts a database Audit to AuditRecord for hash computation
func convertDBAuditToRecord(audit queries.Audit) (AuditRecord, error) {
	var resourceID *string
	if audit.ResourceID.Valid {
		resourceID = &audit.ResourceID.String
	}

	return AuditRecord{
		TenantID:     uuidToString(audit.TenantID),
		ActorType:    audit.ActorType,
		ActorID:      audit.ActorID,
		Action:       audit.Action,
		ResourceType: audit.ResourceType,
		ResourceID:   resourceID,
		Details:      audit.Details,
		Timestamp:    audit.Ts.Time,
	}, nil
}

// equalBytes compares two byte slices for equality
func equalBytes(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

// uuidToString converts a pgtype.UUID to string representation
func uuidToString(uuid pgtype.UUID) string {
	if !uuid.Valid {
		return ""
	}
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		uuid.Bytes[0:4],
		uuid.Bytes[4:6],
		uuid.Bytes[6:8],
		uuid.Bytes[8:10],
		uuid.Bytes[10:16])
}
