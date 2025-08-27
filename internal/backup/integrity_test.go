package backup

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestIntegrityValidator_GenerateFileHash(t *testing.T) {
	// Create temporary directory
	tempDir := t.TempDir()
	validator := NewIntegrityValidator(tempDir)

	// Create test file with known content
	testFile := filepath.Join(tempDir, "test.txt")
	testContent := "Hello, AgentFlow backup test!"
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Generate hash
	hash, err := validator.GenerateFileHash(testFile)
	if err != nil {
		t.Fatalf("Failed to generate hash: %v", err)
	}

	// Verify hash is 64 characters (SHA256 hex)
	if len(hash) != 64 {
		t.Errorf("Expected hash length 64, got %d", len(hash))
	}

	// Verify hash is consistent
	hash2, err := validator.GenerateFileHash(testFile)
	if err != nil {
		t.Fatalf("Failed to generate hash second time: %v", err)
	}

	if hash != hash2 {
		t.Errorf("Hash inconsistent: %s != %s", hash, hash2)
	}
}

func TestIntegrityValidator_WriteAndValidateHashFile(t *testing.T) {
	tempDir := t.TempDir()
	validator := NewIntegrityValidator(tempDir)

	// Create test file
	testFile := filepath.Join(tempDir, "test.txt")
	testContent := "Test content for hash validation"
	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Write hash file
	hashFile := filepath.Join(tempDir, "test.txt.sha256")
	err = validator.WriteHashFile(testFile, hashFile)
	if err != nil {
		t.Fatalf("Failed to write hash file: %v", err)
	}

	// Verify hash file exists and has correct format
	hashContent, err := os.ReadFile(hashFile)
	if err != nil {
		t.Fatalf("Failed to read hash file: %v", err)
	}

	hashLine := strings.TrimSpace(string(hashContent))
	parts := strings.Fields(hashLine)
	if len(parts) != 2 {
		t.Errorf("Expected hash file format 'hash filename', got: %s", hashLine)
	}

	if parts[1] != "test.txt" {
		t.Errorf("Expected filename 'test.txt', got: %s", parts[1])
	}

	// Validate hash file
	err = validator.ValidateHashFile(testFile, hashFile)
	if err != nil {
		t.Errorf("Hash validation failed: %v", err)
	}
}

func TestIntegrityValidator_ValidateHashFile_Tampered(t *testing.T) {
	tempDir := t.TempDir()
	validator := NewIntegrityValidator(tempDir)

	// Create test file and hash
	testFile := filepath.Join(tempDir, "test.txt")
	err := os.WriteFile(testFile, []byte("Original content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	hashFile := filepath.Join(tempDir, "test.txt.sha256")
	err = validator.WriteHashFile(testFile, hashFile)
	if err != nil {
		t.Fatalf("Failed to write hash file: %v", err)
	}

	// Tamper with the file
	err = os.WriteFile(testFile, []byte("Tampered content"), 0644)
	if err != nil {
		t.Fatalf("Failed to tamper with test file: %v", err)
	}

	// Validation should fail
	err = validator.ValidateHashFile(testFile, hashFile)
	if err == nil {
		t.Error("Expected validation to fail for tampered file, but it passed")
	}

	if !strings.Contains(err.Error(), "hash mismatch") {
		t.Errorf("Expected 'hash mismatch' error, got: %v", err)
	}
}

func TestIntegrityValidator_ValidateBackup_Success(t *testing.T) {
	tempDir := t.TempDir()
	validator := NewIntegrityValidator(tempDir)

	backupID := "20250827_120000"

	// Create mock backup files
	schemaFile := filepath.Join(tempDir, "agentflow_backup_20250827_120000_schema.sql.gz")
	dataFile := filepath.Join(tempDir, "agentflow_backup_20250827_120000_data.tar.gz")
	criticalFile := filepath.Join(tempDir, "agentflow_backup_20250827_120000_critical.sql.gz")

	err := os.WriteFile(schemaFile, []byte("-- Schema content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create schema file: %v", err)
	}

	err = os.WriteFile(dataFile, []byte("Data content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create data file: %v", err)
	}

	err = os.WriteFile(criticalFile, []byte("-- Critical tables content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create critical file: %v", err)
	}

	// Create hash files
	err = validator.WriteHashFile(schemaFile, filepath.Join(tempDir, "agentflow_backup_20250827_120000_schema.sha256"))
	if err != nil {
		t.Fatalf("Failed to create schema hash: %v", err)
	}

	err = validator.WriteHashFile(dataFile, filepath.Join(tempDir, "agentflow_backup_20250827_120000_data.sha256"))
	if err != nil {
		t.Fatalf("Failed to create data hash: %v", err)
	}

	err = validator.WriteHashFile(criticalFile, filepath.Join(tempDir, "agentflow_backup_20250827_120000_critical.sha256"))
	if err != nil {
		t.Fatalf("Failed to create critical hash: %v", err)
	}

	// Create manifest
	manifest := BackupManifest{
		BackupID:         "agentflow_backup_20250827_120000",
		Timestamp:        "20250827_120000",
		DatabaseURL:      "postgresql://***@localhost:5432/agentflow",
		CompressionLevel: 6,
		ParallelJobs:     4,
		Files: map[string]BackupFile{
			"schema": {
				Filename: "agentflow_backup_20250827_120000_schema.sql.gz",
				Hash:     "placeholder",
				Type:     "schema_only",
			},
			"data": {
				Filename: "agentflow_backup_20250827_120000_data.tar.gz",
				Hash:     "placeholder",
				Type:     "full_data",
			},
			"critical": {
				Filename: "agentflow_backup_20250827_120000_critical.sql.gz",
				Hash:     "placeholder",
				Type:     "critical_tables",
			},
		},
		CriticalTables: []string{"tenants", "users", "rbac_roles", "rbac_bindings", "audits"},
	}

	manifestFile := filepath.Join(tempDir, "agentflow_backup_20250827_120000_manifest.json")
	manifestData, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("Failed to marshal manifest: %v", err)
	}

	err = os.WriteFile(manifestFile, manifestData, 0644)
	if err != nil {
		t.Fatalf("Failed to create manifest file: %v", err)
	}

	err = validator.WriteHashFile(manifestFile, filepath.Join(tempDir, "agentflow_backup_20250827_120000_manifest.sha256"))
	if err != nil {
		t.Fatalf("Failed to create manifest hash: %v", err)
	}

	// Validate backup
	result, err := validator.ValidateBackup(backupID)
	if err != nil {
		t.Fatalf("Validation failed: %v", err)
	}

	if !result.Success {
		t.Errorf("Expected validation success, got failure: %s", result.Error)
	}

	if len(result.Files) != 4 { // manifest + 3 backup files
		t.Errorf("Expected 4 files validated, got %d", len(result.Files))
	}

	// Check that all files passed validation
	for fileType, validation := range result.Files {
		if !validation.Success {
			t.Errorf("File %s validation failed: %s", fileType, validation.Error)
		}
	}
}

func TestIntegrityValidator_ValidateBackup_MissingManifest(t *testing.T) {
	tempDir := t.TempDir()
	validator := NewIntegrityValidator(tempDir)

	backupID := "nonexistent"

	result, err := validator.ValidateBackup(backupID)
	if err == nil {
		t.Error("Expected error for missing manifest, got nil")
	}

	if result.Success {
		t.Error("Expected validation failure for missing manifest")
	}

	if !strings.Contains(result.Error, "Failed to load manifest") {
		t.Errorf("Expected manifest error, got: %s", result.Error)
	}
}

func TestIntegrityValidator_ValidateBackup_TamperedFile(t *testing.T) {
	tempDir := t.TempDir()
	validator := NewIntegrityValidator(tempDir)

	backupID := "20250827_120000"

	// Create mock backup files
	schemaFile := filepath.Join(tempDir, "agentflow_backup_20250827_120000_schema.sql.gz")
	err := os.WriteFile(schemaFile, []byte("-- Original schema content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create schema file: %v", err)
	}

	// Create hash file for original content
	err = validator.WriteHashFile(schemaFile, filepath.Join(tempDir, "agentflow_backup_20250827_120000_schema.sha256"))
	if err != nil {
		t.Fatalf("Failed to create schema hash: %v", err)
	}

	// Create manifest
	manifest := BackupManifest{
		BackupID:         "agentflow_backup_20250827_120000",
		Timestamp:        "20250827_120000",
		DatabaseURL:      "postgresql://***@localhost:5432/agentflow",
		CompressionLevel: 6,
		ParallelJobs:     4,
		Files: map[string]BackupFile{
			"schema": {
				Filename: "agentflow_backup_20250827_120000_schema.sql.gz",
				Hash:     "placeholder",
				Type:     "schema_only",
			},
		},
		CriticalTables: []string{"tenants", "users"},
	}

	manifestFile := filepath.Join(tempDir, "agentflow_backup_20250827_120000_manifest.json")
	manifestData, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("Failed to marshal manifest: %v", err)
	}

	err = os.WriteFile(manifestFile, manifestData, 0644)
	if err != nil {
		t.Fatalf("Failed to create manifest file: %v", err)
	}

	err = validator.WriteHashFile(manifestFile, filepath.Join(tempDir, "agentflow_backup_20250827_120000_manifest.sha256"))
	if err != nil {
		t.Fatalf("Failed to create manifest hash: %v", err)
	}

	// Tamper with schema file after hash was created
	err = os.WriteFile(schemaFile, []byte("-- Tampered schema content"), 0644)
	if err != nil {
		t.Fatalf("Failed to tamper with schema file: %v", err)
	}

	// Validate backup - should fail
	result, err := validator.ValidateBackup(backupID)
	if err != nil {
		t.Fatalf("Unexpected error during validation: %v", err)
	}

	if result.Success {
		t.Error("Expected validation failure for tampered file")
	}

	// Check that schema file validation failed
	schemaValidation, exists := result.Files["schema"]
	if !exists {
		t.Error("Expected schema file validation result")
	} else if schemaValidation.Success {
		t.Error("Expected schema file validation to fail")
	}
}

func TestIntegrityValidator_ListBackups(t *testing.T) {
	tempDir := t.TempDir()
	validator := NewIntegrityValidator(tempDir)

	// Create some mock manifest files
	manifests := []string{
		"agentflow_backup_20250827_120000_manifest.json",
		"agentflow_backup_20250827_130000_manifest.json",
		"agentflow_backup_20250826_090000_manifest.json",
	}

	for _, manifest := range manifests {
		manifestPath := filepath.Join(tempDir, manifest)
		err := os.WriteFile(manifestPath, []byte("{}"), 0644)
		if err != nil {
			t.Fatalf("Failed to create manifest %s: %v", manifest, err)
		}
	}

	// Create some non-manifest files (should be ignored)
	err := os.WriteFile(filepath.Join(tempDir, "other_file.txt"), []byte("content"), 0644)
	if err != nil {
		t.Fatalf("Failed to create other file: %v", err)
	}

	// List backups
	backupIDs, err := validator.ListBackups()
	if err != nil {
		t.Fatalf("Failed to list backups: %v", err)
	}

	expectedIDs := []string{"20250827_120000", "20250827_130000", "20250826_090000"}
	if len(backupIDs) != len(expectedIDs) {
		t.Errorf("Expected %d backup IDs, got %d", len(expectedIDs), len(backupIDs))
	}

	// Check that all expected IDs are present (order may vary)
	idMap := make(map[string]bool)
	for _, id := range backupIDs {
		idMap[id] = true
	}

	for _, expectedID := range expectedIDs {
		if !idMap[expectedID] {
			t.Errorf("Expected backup ID %s not found in results", expectedID)
		}
	}
}

func TestValidationResult_Summary(t *testing.T) {
	// Test successful validation
	result := &ValidationResult{
		BackupID:  "test_backup",
		Success:   true,
		StartTime: time.Now().Add(-2 * time.Second),
		EndTime:   time.Now(),
		Files: map[string]FileValidation{
			"schema": {Success: true},
			"data":   {Success: true},
		},
	}

	summary := result.Summary()
	if !strings.Contains(summary, "SUCCESS") {
		t.Errorf("Expected SUCCESS in summary, got: %s", summary)
	}
	if !strings.Contains(summary, "2 files verified") {
		t.Errorf("Expected file count in summary, got: %s", summary)
	}

	// Test failed validation
	result.Success = false
	result.Error = "Hash mismatch"
	result.Files["data"] = FileValidation{Success: false, Error: "Hash mismatch"}

	summary = result.Summary()
	if !strings.Contains(summary, "FAILED") {
		t.Errorf("Expected FAILED in summary, got: %s", summary)
	}
	if !strings.Contains(summary, "1/2 files failed") {
		t.Errorf("Expected failure count in summary, got: %s", summary)
	}
}
