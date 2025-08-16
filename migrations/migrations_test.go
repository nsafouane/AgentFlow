package migrations

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// TestMigrationNaming validates that all migration files follow the correct naming convention
func TestMigrationNaming(t *testing.T) {
	// Migration files should follow the pattern: YYYYMMDDHHMMSS_description.sql
	migrationPattern := regexp.MustCompile(`^\d{14}_[a-z0-9_]+\.sql$`)

	files, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("Failed to read migrations directory: %v", err)
	}

	migrationCount := 0
	for _, file := range files {
		if file.IsDir() || strings.HasPrefix(file.Name(), ".") {
			continue
		}

		if !strings.HasSuffix(file.Name(), ".sql") {
			continue
		}

		migrationCount++
		if !migrationPattern.MatchString(file.Name()) {
			t.Errorf("Migration file %s does not follow naming convention YYYYMMDDHHMMSS_description.sql", file.Name())
		}
	}

	if migrationCount == 0 {
		t.Error("No migration files found")
	}
}

// TestMigrationStructure validates that migration files have proper up/down structure
func TestMigrationStructure(t *testing.T) {
	files, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("Failed to read migrations directory: %v", err)
	}

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".sql") || strings.HasPrefix(file.Name(), ".") {
			continue
		}

		content, err := os.ReadFile(file.Name())
		if err != nil {
			t.Errorf("Failed to read migration file %s: %v", file.Name(), err)
			continue
		}

		contentStr := string(content)

		// Check for required goose directives
		if !strings.Contains(contentStr, "-- +goose Up") {
			t.Errorf("Migration file %s missing '-- +goose Up' directive", file.Name())
		}

		if !strings.Contains(contentStr, "-- +goose Down") {
			t.Errorf("Migration file %s missing '-- +goose Down' directive", file.Name())
		}

		// Validate that Up comes before Down
		upIndex := strings.Index(contentStr, "-- +goose Up")
		downIndex := strings.Index(contentStr, "-- +goose Down")

		if upIndex >= 0 && downIndex >= 0 && upIndex >= downIndex {
			t.Errorf("Migration file %s has '-- +goose Down' before '-- +goose Up'", file.Name())
		}
	}
}

// TestMigrationSQLSyntax performs basic SQL syntax validation
func TestMigrationSQLSyntax(t *testing.T) {
	files, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("Failed to read migrations directory: %v", err)
	}

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".sql") || strings.HasPrefix(file.Name(), ".") {
			continue
		}

		content, err := os.ReadFile(file.Name())
		if err != nil {
			t.Errorf("Failed to read migration file %s: %v", file.Name(), err)
			continue
		}

		contentStr := string(content)

		// Basic SQL syntax checks
		if strings.Contains(contentStr, "CREATE TABLE") && !strings.Contains(contentStr, "DROP TABLE") {
			t.Logf("Warning: Migration file %s creates tables but may not have corresponding DROP in down migration", file.Name())
		}

		// Check for potentially dangerous operations without proper safeguards
		if strings.Contains(strings.ToUpper(contentStr), "DROP DATABASE") {
			t.Errorf("Migration file %s contains dangerous DROP DATABASE operation", file.Name())
		}

		if strings.Contains(strings.ToUpper(contentStr), "TRUNCATE") && !strings.Contains(contentStr, "-- SAFE:") {
			t.Errorf("Migration file %s contains TRUNCATE without safety comment", file.Name())
		}
	}
}

// TestMigrationFilePermissions ensures migration files have correct permissions
func TestMigrationFilePermissions(t *testing.T) {
	files, err := os.ReadDir(".")
	if err != nil {
		t.Fatalf("Failed to read migrations directory: %v", err)
	}

	for _, file := range files {
		if file.IsDir() || !strings.HasSuffix(file.Name(), ".sql") || strings.HasPrefix(file.Name(), ".") {
			continue
		}

		info, err := file.Info()
		if err != nil {
			t.Errorf("Failed to get file info for %s: %v", file.Name(), err)
			continue
		}

		// Check that migration files are readable
		mode := info.Mode()
		if mode&0400 == 0 {
			t.Errorf("Migration file %s is not readable", file.Name())
		}
	}
}

// TestMigrationDirectoryStructure validates the overall migration directory structure
func TestMigrationDirectoryStructure(t *testing.T) {
	// Check that migrations directory exists and is accessible
	if _, err := os.Stat("."); os.IsNotExist(err) {
		t.Fatal("Migrations directory does not exist")
	}

	// Check for required files
	expectedFiles := []string{
		"20240101000000_initial_schema.sql",
	}

	for _, expectedFile := range expectedFiles {
		if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
			t.Errorf("Required migration file %s does not exist", expectedFile)
		}
	}

	// Ensure we have at least one migration
	files, err := filepath.Glob("*.sql")
	if err != nil {
		t.Fatalf("Failed to glob migration files: %v", err)
	}

	if len(files) == 0 {
		t.Error("No migration files found in migrations directory")
	}
}
