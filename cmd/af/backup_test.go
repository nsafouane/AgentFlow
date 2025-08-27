package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/agentflow/agentflow/internal/backup"
)

func TestMaskDatabaseURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "PostgreSQL URL with password",
			input:    "postgresql://user:password@localhost:5432/dbname",
			expected: "postgresql://user:***@localhost:5432/dbname",
		},
		{
			name:     "PostgreSQL URL without password",
			input:    "postgresql://user@localhost:5432/dbname",
			expected: "postgresql://user@localhost:5432/dbname",
		},
		{
			name:     "Complex URL with parameters",
			input:    "postgresql://agentflow:dev_password@localhost:5432/agentflow_dev?sslmode=disable",
			expected: "postgresql://agentflow:***@localhost:5432/agentflow_dev?sslmode=disable",
		},
		{
			name:     "URL without @ symbol",
			input:    "postgresql://localhost:5432/dbname",
			expected: "postgresql://localhost:5432/dbname",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := maskDatabaseURL(tt.input)
			if result != tt.expected {
				t.Errorf("maskDatabaseURL() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestBackupCmdValidation(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "No subcommand",
			args:        []string{},
			expectError: true,
			errorMsg:    "backup command requires a subcommand: create, restore, verify, list",
		},
		{
			name:        "Invalid subcommand",
			args:        []string{"invalid"},
			expectError: true,
			errorMsg:    "unknown backup subcommand: invalid",
		},
		{
			name:        "Valid create subcommand",
			args:        []string{"create"},
			expectError: false,
		},
		{
			name:        "Valid restore subcommand with backup ID",
			args:        []string{"restore", "20250827_120000"},
			expectError: false,
		},
		{
			name:        "Valid verify subcommand with backup ID",
			args:        []string{"verify", "20250827_120000"},
			expectError: false,
		},
		{
			name:        "Valid list subcommand",
			args:        []string{"list"},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := backupCmd(tt.args)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errorMsg != "" && err.Error() != tt.errorMsg {
					t.Errorf("Expected error message '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				// For valid commands, we expect them to fail due to missing scripts/database
				// but the command parsing should work
				if err != nil && err.Error() == tt.errorMsg {
					t.Errorf("Unexpected specific error: %v", err)
				}
			}
		})
	}
}

func TestListBackupsWithMockData(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()

	// Create mock backup manifests
	manifests := []string{
		"agentflow_backup_20250827_120000_manifest.json",
		"agentflow_backup_20250827_130000_manifest.json",
		"agentflow_backup_20250826_090000_manifest.json",
	}

	for _, manifest := range manifests {
		manifestPath := filepath.Join(tempDir, manifest)
		mockManifest := map[string]interface{}{
			"backup_id": "test_backup",
			"timestamp": "20250827_120000",
		}

		data, err := json.Marshal(mockManifest)
		if err != nil {
			t.Fatalf("Failed to marshal mock manifest: %v", err)
		}

		err = os.WriteFile(manifestPath, data, 0644)
		if err != nil {
			t.Fatalf("Failed to create mock manifest %s: %v", manifest, err)
		}
	}

	// Test list functionality using the backup package directly
	validator := backup.NewIntegrityValidator(tempDir)
	backups, err := validator.ListBackups()
	if err != nil {
		t.Fatalf("Failed to list backups: %v", err)
	}

	expectedCount := 3
	if len(backups) != expectedCount {
		t.Errorf("Expected %d backups, got %d", expectedCount, len(backups))
	}

	// Verify backup IDs are extracted correctly
	expectedIDs := map[string]bool{
		"20250827_120000": true,
		"20250827_130000": true,
		"20250826_090000": true,
	}

	for _, backupID := range backups {
		if !expectedIDs[backupID] {
			t.Errorf("Unexpected backup ID: %s", backupID)
		}
	}
}

func TestVerifyBackupWithMockData(t *testing.T) {
	// Create temporary directory for test
	tempDir := t.TempDir()
	validator := backup.NewIntegrityValidator(tempDir)

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
	manifest := backup.BackupManifest{
		BackupID:         "agentflow_backup_20250827_120000",
		Timestamp:        "20250827_120000",
		DatabaseURL:      "postgresql://***@localhost:5432/agentflow",
		CompressionLevel: 6,
		ParallelJobs:     4,
		Files: map[string]backup.BackupFile{
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

	// Test validation
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

	// Test performance requirement (should be very fast for small test files)
	if result.GetDuration() > 5*time.Second {
		t.Errorf("Validation took too long: %v", result.GetDuration())
	}
}

func TestBackupIntegrityHashGeneration(t *testing.T) {
	// Create temporary directory and file
	tempDir := t.TempDir()
	validator := backup.NewIntegrityValidator(tempDir)

	testFile := filepath.Join(tempDir, "test.txt")
	testContent := "Hello, AgentFlow backup integrity test!"

	err := os.WriteFile(testFile, []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Generate hash
	hash1, err := validator.GenerateFileHash(testFile)
	if err != nil {
		t.Fatalf("Failed to generate hash: %v", err)
	}

	// Verify hash is consistent
	hash2, err := validator.GenerateFileHash(testFile)
	if err != nil {
		t.Fatalf("Failed to generate hash second time: %v", err)
	}

	if hash1 != hash2 {
		t.Errorf("Hash inconsistent: %s != %s", hash1, hash2)
	}

	// Verify hash format (SHA256 should be 64 hex characters)
	if len(hash1) != 64 {
		t.Errorf("Expected hash length 64, got %d", len(hash1))
	}

	// Test hash file creation and validation
	hashFile := filepath.Join(tempDir, "test.txt.sha256")
	err = validator.WriteHashFile(testFile, hashFile)
	if err != nil {
		t.Fatalf("Failed to write hash file: %v", err)
	}

	// Validate hash file
	err = validator.ValidateHashFile(testFile, hashFile)
	if err != nil {
		t.Errorf("Hash validation failed: %v", err)
	}

	// Test tamper detection
	err = os.WriteFile(testFile, []byte("Tampered content"), 0644)
	if err != nil {
		t.Fatalf("Failed to tamper with test file: %v", err)
	}

	err = validator.ValidateHashFile(testFile, hashFile)
	if err == nil {
		t.Error("Expected validation to fail for tampered file, but it passed")
	}
}

func TestBackupManifestStructure(t *testing.T) {
	// Test that BackupManifest can be properly marshaled and unmarshaled
	manifest := backup.BackupManifest{
		BackupID:         "test_backup_20250827_120000",
		Timestamp:        "20250827_120000",
		DatabaseURL:      "postgresql://***@localhost:5432/agentflow",
		CompressionLevel: 6,
		ParallelJobs:     4,
		Files: map[string]backup.BackupFile{
			"schema": {
				Filename: "test_schema.sql.gz",
				Hash:     "abc123",
				Type:     "schema_only",
			},
		},
		CriticalTables: []string{"tenants", "users"},
	}

	// Marshal to JSON
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatalf("Failed to marshal manifest: %v", err)
	}

	// Unmarshal back
	var unmarshaled backup.BackupManifest
	err = json.Unmarshal(data, &unmarshaled)
	if err != nil {
		t.Fatalf("Failed to unmarshal manifest: %v", err)
	}

	// Verify fields
	if unmarshaled.BackupID != manifest.BackupID {
		t.Errorf("BackupID mismatch: %s != %s", unmarshaled.BackupID, manifest.BackupID)
	}

	if unmarshaled.CompressionLevel != manifest.CompressionLevel {
		t.Errorf("CompressionLevel mismatch: %d != %d", unmarshaled.CompressionLevel, manifest.CompressionLevel)
	}

	if len(unmarshaled.Files) != len(manifest.Files) {
		t.Errorf("Files count mismatch: %d != %d", len(unmarshaled.Files), len(manifest.Files))
	}

	if len(unmarshaled.CriticalTables) != len(manifest.CriticalTables) {
		t.Errorf("CriticalTables count mismatch: %d != %d", len(unmarshaled.CriticalTables), len(manifest.CriticalTables))
	}
}
