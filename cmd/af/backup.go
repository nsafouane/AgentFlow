package main

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/agentflow/agentflow/internal/backup"
)

// backupCmd handles backup-related operations
func backupCmd(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("backup command requires a subcommand: create, restore, verify, list")
	}

	subcommand := args[0]
	subArgs := args[1:]

	switch subcommand {
	case "create":
		return createBackup(subArgs)
	case "restore":
		return restoreBackup(subArgs)
	case "verify":
		return verifyBackup(subArgs)
	case "list":
		return listBackups(subArgs)
	default:
		return fmt.Errorf("unknown backup subcommand: %s", subcommand)
	}
}

// createBackup creates a new database backup
func createBackup(args []string) error {
	// Parse arguments
	dbURL := os.Getenv("AF_DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgresql://agentflow:agentflow@localhost:5432/agentflow"
	}

	backupDir := "./backups"
	if len(args) > 0 {
		backupDir = args[0]
	}

	fmt.Printf("Creating backup...\n")
	fmt.Printf("Database: %s\n", maskDatabaseURL(dbURL))
	fmt.Printf("Backup directory: %s\n", backupDir)

	// Ensure backup directory exists
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return fmt.Errorf("failed to create backup directory: %w", err)
	}

	// Choose script based on platform
	var scriptPath string
	if runtime.GOOS == "windows" {
		scriptPath = "scripts/backup-database.ps1"
		if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
			return fmt.Errorf("backup script not found: %s", scriptPath)
		}
	} else {
		scriptPath = "scripts/backup-database.sh"
		if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
			return fmt.Errorf("backup script not found: %s", scriptPath)
		}
	}

	// Execute backup script
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-File", scriptPath,
			"-DatabaseUrl", dbURL, "-BackupDir", backupDir)
	} else {
		cmd = exec.Command("bash", scriptPath, dbURL, backupDir)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	startTime := time.Now()
	err := cmd.Run()
	duration := time.Since(startTime)

	if err != nil {
		return fmt.Errorf("backup script failed: %w", err)
	}

	fmt.Printf("\nBackup completed successfully in %.2f seconds\n", duration.Seconds())

	// Verify the backup was created
	validator := backup.NewIntegrityValidator(backupDir)
	backups, err := validator.ListBackups()
	if err != nil {
		fmt.Printf("Warning: Could not verify backup creation: %v\n", err)
		return nil
	}

	if len(backups) > 0 {
		// Get the most recent backup (assuming it's the one we just created)
		latestBackup := backups[len(backups)-1]
		fmt.Printf("Backup ID: %s\n", latestBackup)

		// Quick integrity check
		result, err := validator.ValidateBackup(latestBackup)
		if err != nil {
			fmt.Printf("Warning: Could not verify backup integrity: %v\n", err)
		} else if result.Success {
			fmt.Printf("Backup integrity: VERIFIED\n")
		} else {
			fmt.Printf("Warning: Backup integrity check failed: %s\n", result.Error)
		}
	}

	return nil
}

// restoreBackup restores from a backup
func restoreBackup(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("restore requires a backup ID")
	}

	backupID := args[0]

	dbURL := os.Getenv("AF_DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgresql://agentflow:agentflow@localhost:5432/agentflow"
	}

	backupDir := "./backups"
	if len(args) > 1 {
		backupDir = args[1]
	}

	restoreType := "full"
	if len(args) > 2 {
		restoreType = args[2]
	}

	fmt.Printf("Restoring backup...\n")
	fmt.Printf("Backup ID: %s\n", backupID)
	fmt.Printf("Database: %s\n", maskDatabaseURL(dbURL))
	fmt.Printf("Backup directory: %s\n", backupDir)
	fmt.Printf("Restore type: %s\n", restoreType)

	// Verify backup exists and is valid before attempting restore
	validator := backup.NewIntegrityValidator(backupDir)
	result, err := validator.ValidateBackup(backupID)
	if err != nil {
		return fmt.Errorf("backup validation failed: %w", err)
	}

	if !result.Success {
		return fmt.Errorf("backup integrity check failed: %s", result.Error)
	}

	fmt.Printf("Backup integrity: VERIFIED\n")

	// Choose script based on platform
	var scriptPath string
	if runtime.GOOS == "windows" {
		scriptPath = "scripts/restore-database.ps1"
		if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
			return fmt.Errorf("restore script not found: %s", scriptPath)
		}
	} else {
		scriptPath = "scripts/restore-database.sh"
		if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
			return fmt.Errorf("restore script not found: %s", scriptPath)
		}
	}

	// Execute restore script
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("powershell", "-ExecutionPolicy", "Bypass", "-File", scriptPath,
			"-BackupId", backupID, "-DatabaseUrl", dbURL, "-BackupDir", backupDir, "-RestoreType", restoreType)
	} else {
		cmd = exec.Command("bash", scriptPath, backupID, dbURL, backupDir, restoreType)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin // Allow interactive confirmation

	startTime := time.Now()
	err = cmd.Run()
	duration := time.Since(startTime)

	if err != nil {
		return fmt.Errorf("restore script failed: %w", err)
	}

	fmt.Printf("\nRestore completed successfully in %.2f seconds\n", duration.Seconds())

	if duration.Seconds() > 300 {
		fmt.Printf("Warning: Restore took longer than 5 minutes (%.2fs)\n", duration.Seconds())
		fmt.Printf("Consider optimizing backup size or database performance\n")
	}

	return nil
}

// verifyBackup verifies the integrity of a backup
func verifyBackup(args []string) error {
	if len(args) == 0 {
		return fmt.Errorf("verify requires a backup ID")
	}

	backupID := args[0]

	backupDir := "./backups"
	if len(args) > 1 {
		backupDir = args[1]
	}

	outputJSON := false
	if len(args) > 2 && args[2] == "--json" {
		outputJSON = true
	}

	validator := backup.NewIntegrityValidator(backupDir)

	fmt.Printf("Verifying backup integrity...\n")
	fmt.Printf("Backup ID: %s\n", backupID)
	fmt.Printf("Backup directory: %s\n", backupDir)

	result, err := validator.ValidateBackup(backupID)
	if err != nil {
		if outputJSON {
			jsonResult := map[string]interface{}{
				"success":   false,
				"error":     err.Error(),
				"backup_id": backupID,
			}
			jsonData, _ := json.MarshalIndent(jsonResult, "", "  ")
			fmt.Println(string(jsonData))
		} else {
			fmt.Printf("Verification failed: %v\n", err)
		}
		return err
	}

	if outputJSON {
		jsonData, err := json.MarshalIndent(result, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal result to JSON: %w", err)
		}
		fmt.Println(string(jsonData))
	} else {
		fmt.Printf("\n%s\n", result.Summary())

		if result.Success {
			fmt.Printf("\nFile verification details:\n")
			for fileType, validation := range result.Files {
				if validation.Success {
					fmt.Printf("  ✓ %s: %s\n", fileType, filepath.Base(validation.Path))
				} else {
					fmt.Printf("  ✗ %s: %s - %s\n", fileType, filepath.Base(validation.Path), validation.Error)
				}
			}
		} else {
			fmt.Printf("\nFailed files:\n")
			for fileType, validation := range result.Files {
				if !validation.Success {
					fmt.Printf("  ✗ %s: %s\n", fileType, validation.Error)
				}
			}
		}
	}

	if !result.Success {
		return fmt.Errorf("backup verification failed")
	}

	return nil
}

// listBackups lists available backups
func listBackups(args []string) error {
	backupDir := "./backups"
	if len(args) > 0 {
		backupDir = args[0]
	}

	outputJSON := false
	if len(args) > 1 && args[1] == "--json" {
		outputJSON = true
	}

	validator := backup.NewIntegrityValidator(backupDir)
	backups, err := validator.ListBackups()
	if err != nil {
		return fmt.Errorf("failed to list backups: %w", err)
	}

	if len(backups) == 0 {
		if outputJSON {
			fmt.Println("[]")
		} else {
			fmt.Printf("No backups found in %s\n", backupDir)
		}
		return nil
	}

	if outputJSON {
		jsonData, err := json.MarshalIndent(backups, "", "  ")
		if err != nil {
			return fmt.Errorf("failed to marshal backups to JSON: %w", err)
		}
		fmt.Println(string(jsonData))
	} else {
		fmt.Printf("Available backups in %s:\n", backupDir)
		for _, backupID := range backups {
			fmt.Printf("  %s\n", backupID)
		}
		fmt.Printf("\nTotal: %d backups\n", len(backups))
	}

	return nil
}

// maskDatabaseURL masks sensitive information in database URL
func maskDatabaseURL(url string) string {
	// Replace password with ***
	if strings.Contains(url, "@") {
		parts := strings.Split(url, "@")
		if len(parts) >= 2 {
			userPart := parts[0]
			// Look for the last colon that's not part of ://
			lastColonIndex := strings.LastIndex(userPart, ":")
			if lastColonIndex > 0 && lastColonIndex < len(userPart)-2 {
				// Make sure it's not part of ://
				if userPart[lastColonIndex:lastColonIndex+3] != "://" {
					// Replace password part with ***
					maskedUserPart := userPart[:lastColonIndex+1] + "***"
					parts[0] = maskedUserPart
					return strings.Join(parts, "@")
				}
			}
		}
	}
	return url
}
