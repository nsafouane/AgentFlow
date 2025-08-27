package backup

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// BackupManifest represents the structure of a backup manifest file
type BackupManifest struct {
	BackupID         string                `json:"backup_id"`
	Timestamp        string                `json:"timestamp"`
	DatabaseURL      string                `json:"database_url"`
	CompressionLevel int                   `json:"compression_level"`
	ParallelJobs     int                   `json:"parallel_jobs"`
	Files            map[string]BackupFile `json:"files"`
	CriticalTables   []string              `json:"critical_tables"`
}

// BackupFile represents information about a backup file
type BackupFile struct {
	Filename string `json:"filename"`
	Hash     string `json:"hash"`
	Type     string `json:"type"`
}

// IntegrityValidator provides backup integrity validation functionality
type IntegrityValidator struct {
	backupDir string
}

// NewIntegrityValidator creates a new integrity validator
func NewIntegrityValidator(backupDir string) *IntegrityValidator {
	return &IntegrityValidator{
		backupDir: backupDir,
	}
}

// ValidateBackup validates the integrity of a backup by checking all file hashes
func (v *IntegrityValidator) ValidateBackup(backupID string) (*ValidationResult, error) {
	result := &ValidationResult{
		BackupID:  backupID,
		StartTime: time.Now(),
		Files:     make(map[string]FileValidation),
	}

	// Load and validate manifest
	manifestPath := filepath.Join(v.backupDir, fmt.Sprintf("agentflow_backup_%s_manifest.json", backupID))
	manifest, err := v.loadManifest(manifestPath)
	if err != nil {
		result.Success = false
		result.Error = fmt.Sprintf("Failed to load manifest: %v", err)
		result.EndTime = time.Now()
		return result, err
	}

	result.Manifest = manifest

	// Validate manifest file itself
	manifestHashPath := filepath.Join(v.backupDir, fmt.Sprintf("agentflow_backup_%s_manifest.sha256", backupID))
	if err := v.validateFileHash(manifestPath, manifestHashPath); err != nil {
		result.Files["manifest"] = FileValidation{
			Path:    manifestPath,
			Success: false,
			Error:   err.Error(),
		}
		result.Success = false
		result.Error = "Manifest integrity check failed"
		result.EndTime = time.Now()
		return result, err
	}

	result.Files["manifest"] = FileValidation{
		Path:    manifestPath,
		Success: true,
	}

	// Validate each backup file
	allValid := true
	for fileType, fileInfo := range manifest.Files {
		filePath := filepath.Join(v.backupDir, fileInfo.Filename)
		hashPath := filepath.Join(v.backupDir, fmt.Sprintf("agentflow_backup_%s_%s.sha256", backupID, fileType))

		validation := FileValidation{
			Path: filePath,
		}

		if err := v.validateFileHash(filePath, hashPath); err != nil {
			validation.Success = false
			validation.Error = err.Error()
			allValid = false
		} else {
			validation.Success = true
		}

		result.Files[fileType] = validation
	}

	result.Success = allValid
	result.EndTime = time.Now()

	if !allValid {
		result.Error = "One or more backup files failed integrity validation"
	}

	return result, nil
}

// GenerateFileHash generates a SHA256 hash for a file
func (v *IntegrityValidator) GenerateFileHash(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", fmt.Errorf("failed to open file %s: %w", filePath, err)
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return "", fmt.Errorf("failed to hash file %s: %w", filePath, err)
	}

	return hex.EncodeToString(hasher.Sum(nil)), nil
}

// WriteHashFile writes a hash to a .sha256 file in the standard format
func (v *IntegrityValidator) WriteHashFile(filePath, hashPath string) error {
	hash, err := v.GenerateFileHash(filePath)
	if err != nil {
		return err
	}

	filename := filepath.Base(filePath)
	content := fmt.Sprintf("%s  %s\n", hash, filename)

	return os.WriteFile(hashPath, []byte(content), 0644)
}

// ValidateHashFile validates a file against its .sha256 hash file
func (v *IntegrityValidator) ValidateHashFile(filePath, hashPath string) error {
	return v.validateFileHash(filePath, hashPath)
}

// ListBackups returns a list of available backup IDs
func (v *IntegrityValidator) ListBackups() ([]string, error) {
	pattern := filepath.Join(v.backupDir, "agentflow_backup_*_manifest.json")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to list backups: %w", err)
	}

	var backupIDs []string
	for _, match := range matches {
		filename := filepath.Base(match)
		// Extract backup ID from filename: agentflow_backup_{ID}_manifest.json
		if strings.HasPrefix(filename, "agentflow_backup_") && strings.HasSuffix(filename, "_manifest.json") {
			id := strings.TrimPrefix(filename, "agentflow_backup_")
			id = strings.TrimSuffix(id, "_manifest.json")
			backupIDs = append(backupIDs, id)
		}
	}

	return backupIDs, nil
}

// loadManifest loads and parses a backup manifest file
func (v *IntegrityValidator) loadManifest(manifestPath string) (*BackupManifest, error) {
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest file: %w", err)
	}

	var manifest BackupManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest JSON: %w", err)
	}

	return &manifest, nil
}

// validateFileHash validates a file against its expected hash from a .sha256 file
func (v *IntegrityValidator) validateFileHash(filePath, hashPath string) error {
	// Read expected hash from hash file
	hashData, err := os.ReadFile(hashPath)
	if err != nil {
		return fmt.Errorf("failed to read hash file %s: %w", hashPath, err)
	}

	// Parse hash file format: "hash  filename"
	hashLine := strings.TrimSpace(string(hashData))
	parts := strings.Fields(hashLine)
	if len(parts) < 1 {
		return fmt.Errorf("invalid hash file format in %s", hashPath)
	}
	expectedHash := parts[0]

	// Calculate actual hash
	actualHash, err := v.GenerateFileHash(filePath)
	if err != nil {
		return err
	}

	// Compare hashes
	if expectedHash != actualHash {
		return fmt.Errorf("hash mismatch for %s: expected %s, got %s", filePath, expectedHash, actualHash)
	}

	return nil
}

// ValidationResult represents the result of a backup validation
type ValidationResult struct {
	BackupID  string                    `json:"backup_id"`
	Success   bool                      `json:"success"`
	Error     string                    `json:"error,omitempty"`
	StartTime time.Time                 `json:"start_time"`
	EndTime   time.Time                 `json:"end_time"`
	Duration  time.Duration             `json:"duration"`
	Manifest  *BackupManifest           `json:"manifest,omitempty"`
	Files     map[string]FileValidation `json:"files"`
}

// FileValidation represents the validation result for a single file
type FileValidation struct {
	Path    string `json:"path"`
	Success bool   `json:"success"`
	Error   string `json:"error,omitempty"`
}

// GetDuration calculates and returns the validation duration
func (r *ValidationResult) GetDuration() time.Duration {
	if r.EndTime.IsZero() {
		return time.Since(r.StartTime)
	}
	return r.EndTime.Sub(r.StartTime)
}

// Summary returns a human-readable summary of the validation result
func (r *ValidationResult) Summary() string {
	duration := r.GetDuration()

	if r.Success {
		return fmt.Sprintf("Backup %s validation: SUCCESS (%.2fs, %d files verified)",
			r.BackupID, duration.Seconds(), len(r.Files))
	}

	failedCount := 0
	for _, file := range r.Files {
		if !file.Success {
			failedCount++
		}
	}

	return fmt.Sprintf("Backup %s validation: FAILED (%.2fs, %d/%d files failed) - %s",
		r.BackupID, duration.Seconds(), failedCount, len(r.Files), r.Error)
}
