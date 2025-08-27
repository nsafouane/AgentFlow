package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/agentflow/agentflow/internal/storage/audit"
	"github.com/agentflow/agentflow/internal/storage/queries"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// AuditVerifyResult represents the JSON output for audit verify command
type AuditVerifyResult struct {
	Status             string `json:"status"` // "success", "tampered", "error"
	Timestamp          string `json:"timestamp"`
	TotalRecords       int    `json:"total_records"`
	VerifiedRecords    int    `json:"verified_records"`
	ThroughputPerSec   int    `json:"throughput_per_sec"`
	Duration           string `json:"duration"`
	FirstTamperedIndex *int   `json:"first_tampered_index,omitempty"`
	ErrorMessage       string `json:"error_message,omitempty"`
}

// auditCmd handles audit-related operations
func auditCmd(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("audit command requires a subcommand: verify")
	}

	subcommand := args[0]
	subArgs := args[1:]

	switch subcommand {
	case "verify":
		return verifyAuditChain(subArgs)
	default:
		return fmt.Errorf("unknown audit subcommand: %s", subcommand)
	}
}

func verifyAuditChain(args []string) error {
	// Parse command line arguments
	var tenantID *pgtype.UUID
	var jsonOutput bool

	for _, arg := range args {
		if arg == "--json" {
			jsonOutput = true
		} else if len(arg) > 12 && arg[:12] == "--tenant-id=" {
			// Parse tenant ID
			tenantIDStr := arg[12:]
			var uuid pgtype.UUID
			err := uuid.Scan(tenantIDStr)
			if err != nil {
				return fmt.Errorf("invalid tenant ID format: %s", tenantIDStr)
			}
			tenantID = &uuid
		}
	}

	// Get database connection string from environment
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://agentflow:dev_password@localhost:5432/agentflow_dev?sslmode=disable"
	}

	// Connect to database
	conn, err := pgx.Connect(context.Background(), dbURL)
	if err != nil {
		result := AuditVerifyResult{
			Status:       "error",
			Timestamp:    time.Now().UTC().Format(time.RFC3339),
			ErrorMessage: fmt.Sprintf("Failed to connect to database: %v", err),
		}
		outputResult(result, jsonOutput)
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer conn.Close(context.Background())

	// Create queries and audit service
	q := queries.New(conn)
	auditService := audit.NewService(q)

	startTime := time.Now()

	if tenantID != nil {
		// Verify specific tenant
		return verifyTenant(auditService, *tenantID, jsonOutput, startTime)
	} else {
		// Verify all tenants
		return verifyAllTenants(auditService, conn, jsonOutput, startTime)
	}
}

func verifyTenant(auditService *audit.Service, tenantID pgtype.UUID, jsonOutput bool, startTime time.Time) error {
	result, err := auditService.VerifyChainIntegrity(context.Background(), tenantID)
	if err != nil {
		auditResult := AuditVerifyResult{
			Status:       "error",
			Timestamp:    time.Now().UTC().Format(time.RFC3339),
			ErrorMessage: fmt.Sprintf("Verification failed: %v", err),
		}
		outputResult(auditResult, jsonOutput)
		return fmt.Errorf("verification failed: %w", err)
	}

	duration := time.Since(startTime)
	throughput := 0
	if duration.Seconds() > 0 {
		throughput = int(float64(result.TotalRecords) / duration.Seconds())
	}

	auditResult := AuditVerifyResult{
		Status:             getStatus(result),
		Timestamp:          time.Now().UTC().Format(time.RFC3339),
		TotalRecords:       result.TotalRecords,
		VerifiedRecords:    result.TotalRecords,
		ThroughputPerSec:   throughput,
		Duration:           duration.String(),
		FirstTamperedIndex: result.FirstTamperedIndex,
		ErrorMessage:       result.ErrorMessage,
	}

	if !result.Valid {
		auditResult.VerifiedRecords = 0
		if result.FirstTamperedIndex != nil {
			auditResult.VerifiedRecords = *result.FirstTamperedIndex
		}
	}

	outputResult(auditResult, jsonOutput)

	if !result.Valid {
		return fmt.Errorf("audit hash-chain integrity compromised")
	}

	return nil
}

func verifyAllTenants(auditService *audit.Service, conn *pgx.Conn, jsonOutput bool, startTime time.Time) error {
	// Get all tenant IDs
	rows, err := conn.Query(context.Background(), "SELECT id FROM tenants")
	if err != nil {
		result := AuditVerifyResult{
			Status:       "error",
			Timestamp:    time.Now().UTC().Format(time.RFC3339),
			ErrorMessage: fmt.Sprintf("Failed to get tenants: %v", err),
		}
		outputResult(result, jsonOutput)
		return fmt.Errorf("failed to get tenants: %w", err)
	}
	defer rows.Close()

	var tenantIDs []pgtype.UUID
	for rows.Next() {
		var tenantID pgtype.UUID
		err := rows.Scan(&tenantID)
		if err != nil {
			result := AuditVerifyResult{
				Status:       "error",
				Timestamp:    time.Now().UTC().Format(time.RFC3339),
				ErrorMessage: fmt.Sprintf("Failed to scan tenant ID: %v", err),
			}
			outputResult(result, jsonOutput)
			return fmt.Errorf("failed to scan tenant ID: %w", err)
		}
		tenantIDs = append(tenantIDs, tenantID)
	}

	// Verify each tenant
	totalRecords := 0
	totalVerified := 0
	var firstError *audit.VerificationResult

	for _, tenantID := range tenantIDs {
		result, err := auditService.VerifyChainIntegrity(context.Background(), tenantID)
		if err != nil {
			auditResult := AuditVerifyResult{
				Status:       "error",
				Timestamp:    time.Now().UTC().Format(time.RFC3339),
				ErrorMessage: fmt.Sprintf("Verification failed for tenant %s: %v", tenantID.Bytes, err),
			}
			outputResult(auditResult, jsonOutput)
			return fmt.Errorf("verification failed for tenant: %w", err)
		}

		totalRecords += result.TotalRecords
		if result.Valid {
			totalVerified += result.TotalRecords
		} else {
			if firstError == nil {
				firstError = &result
			}
			if result.FirstTamperedIndex != nil {
				totalVerified += *result.FirstTamperedIndex
			}
		}
	}

	duration := time.Since(startTime)
	throughput := 0
	if duration.Seconds() > 0 {
		throughput = int(float64(totalRecords) / duration.Seconds())
	}

	status := "success"
	var firstTamperedIndex *int
	var errorMessage string

	if firstError != nil {
		status = "tampered"
		firstTamperedIndex = firstError.FirstTamperedIndex
		errorMessage = firstError.ErrorMessage
	}

	auditResult := AuditVerifyResult{
		Status:             status,
		Timestamp:          time.Now().UTC().Format(time.RFC3339),
		TotalRecords:       totalRecords,
		VerifiedRecords:    totalVerified,
		ThroughputPerSec:   throughput,
		Duration:           duration.String(),
		FirstTamperedIndex: firstTamperedIndex,
		ErrorMessage:       errorMessage,
	}

	outputResult(auditResult, jsonOutput)

	if firstError != nil {
		return fmt.Errorf("audit hash-chain integrity compromised")
	}

	return nil
}

func getStatus(result audit.VerificationResult) string {
	if result.Valid {
		return "success"
	}
	return "tampered"
}

func outputResult(result AuditVerifyResult, jsonOutput bool) {
	if jsonOutput {
		// Output JSON
		fmt.Printf(`{
  "status": "%s",
  "timestamp": "%s",
  "total_records": %d,
  "verified_records": %d,
  "throughput_per_sec": %d,
  "duration": "%s"`,
			result.Status,
			result.Timestamp,
			result.TotalRecords,
			result.VerifiedRecords,
			result.ThroughputPerSec,
			result.Duration)

		if result.FirstTamperedIndex != nil {
			fmt.Printf(`,
  "first_tampered_index": %d`, *result.FirstTamperedIndex)
		}

		if result.ErrorMessage != "" {
			fmt.Printf(`,
  "error_message": "%s"`, result.ErrorMessage)
		}

		fmt.Println("\n}")
	} else {
		// Human-readable output
		fmt.Printf("Audit Hash-Chain Verification\n")
		fmt.Printf("Status: %s\n", result.Status)
		fmt.Printf("Total Records: %d\n", result.TotalRecords)
		fmt.Printf("Verified Records: %d\n", result.VerifiedRecords)
		fmt.Printf("Throughput: %d entries/sec\n", result.ThroughputPerSec)
		fmt.Printf("Duration: %s\n", result.Duration)

		if result.FirstTamperedIndex != nil {
			fmt.Printf("First Tampered Index: %d\n", *result.FirstTamperedIndex)
		}

		if result.ErrorMessage != "" {
			fmt.Printf("Error: %s\n", result.ErrorMessage)
		}

		switch result.Status {
		case "success":
			fmt.Println("✓ Hash-chain integrity verified successfully")
		case "tampered":
			fmt.Println("✗ Hash-chain integrity compromised - tampering detected")
		default:
			fmt.Println("✗ Verification failed due to error")
		}
	}
}
