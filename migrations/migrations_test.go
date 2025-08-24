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
		"20250824000000_core_schema.sql",
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

// TestCoreSchemaStructure validates the core schema migration has all required tables
func TestCoreSchemaStructure(t *testing.T) {
	content, err := os.ReadFile("20250824000000_core_schema.sql")
	if err != nil {
		t.Fatalf("Failed to read core schema migration: %v", err)
	}

	contentStr := string(content)

	// Required core tables
	requiredTables := []string{
		"CREATE TABLE tenants",
		"CREATE TABLE users",
		"CREATE TABLE agents",
		"CREATE TABLE workflows",
		"CREATE TABLE plans",
		"CREATE TABLE messages",
		"CREATE TABLE tools",
		"CREATE TABLE audits",
		"CREATE TABLE budgets",
		"CREATE TABLE rbac_roles",
		"CREATE TABLE rbac_bindings",
	}

	for _, table := range requiredTables {
		if !strings.Contains(contentStr, table) {
			t.Errorf("Core schema migration missing required table: %s", table)
		}
	}

	// Required indexes for multi-tenant isolation
	requiredIndexes := []string{
		"idx_users_tenant_id",
		"idx_agents_tenant_id",
		"idx_workflows_tenant_id",
		"idx_messages_tenant_id",
		"idx_tools_tenant_id",
		"idx_audits_tenant_id",
		"idx_budgets_tenant_id",
		"idx_rbac_roles_tenant_id",
		"idx_rbac_bindings_tenant_id",
	}

	for _, index := range requiredIndexes {
		if !strings.Contains(contentStr, index) {
			t.Errorf("Core schema migration missing required index: %s", index)
		}
	}

	// Required JSONB GIN indexes
	requiredGinIndexes := []string{
		"idx_plans_state_gin",
		"idx_agents_config_gin",
		"idx_messages_payload_gin",
		"idx_tools_schema_gin",
		"idx_audits_details_gin",
		"idx_budgets_limits_gin",
		"idx_rbac_roles_permissions_gin",
	}

	for _, index := range requiredGinIndexes {
		if !strings.Contains(contentStr, index) {
			t.Errorf("Core schema migration missing required GIN index: %s", index)
		}
	}

	// Verify multi-tenant isolation (tenant_id columns)
	tenantScopedTables := []string{
		"users", "agents", "workflows", "messages",
		"tools", "audits", "budgets", "rbac_roles", "rbac_bindings",
	}

	for _, table := range tenantScopedTables {
		if !strings.Contains(contentStr, "tenant_id UUID NOT NULL REFERENCES tenants(id)") {
			// Check if table has tenant_id reference
			tablePattern := "CREATE TABLE " + table
			tableIndex := strings.Index(contentStr, tablePattern)
			if tableIndex == -1 {
				continue // Table not found, will be caught by required tables test
			}

			// Find the end of this table definition
			nextTableIndex := strings.Index(contentStr[tableIndex+len(tablePattern):], "CREATE TABLE")
			var tableDefinition string
			if nextTableIndex == -1 {
				tableDefinition = contentStr[tableIndex:]
			} else {
				tableDefinition = contentStr[tableIndex : tableIndex+len(tablePattern)+nextTableIndex]
			}

			if !strings.Contains(tableDefinition, "tenant_id") {
				t.Errorf("Table %s missing tenant_id column for multi-tenant isolation", table)
			}
		}
	}
}

// TestCoreSchemaDownMigration validates the down migration properly cleans up
func TestCoreSchemaDownMigration(t *testing.T) {
	content, err := os.ReadFile("20250824000000_core_schema.sql")
	if err != nil {
		t.Fatalf("Failed to read core schema migration: %v", err)
	}

	contentStr := string(content)

	// Find the down migration section
	downIndex := strings.Index(contentStr, "-- +goose Down")
	if downIndex == -1 {
		t.Fatal("Core schema migration missing down migration section")
	}

	downSection := contentStr[downIndex:]

	// Required DROP TABLE statements in down migration
	requiredDrops := []string{
		"DROP TABLE IF EXISTS rbac_bindings",
		"DROP TABLE IF EXISTS rbac_roles",
		"DROP TABLE IF EXISTS budgets",
		"DROP TABLE IF EXISTS audits",
		"DROP TABLE IF EXISTS tools",
		"DROP TABLE IF EXISTS messages",
		"DROP TABLE IF EXISTS plans",
		"DROP TABLE IF EXISTS workflows",
		"DROP TABLE IF EXISTS agents",
		"DROP TABLE IF EXISTS users",
		"DROP TABLE IF EXISTS tenants",
	}

	for _, drop := range requiredDrops {
		if !strings.Contains(downSection, drop) {
			t.Errorf("Core schema down migration missing: %s", drop)
		}
	}
}

// TestMultiTenantConstraints validates foreign key constraints for multi-tenant isolation
func TestMultiTenantConstraints(t *testing.T) {
	content, err := os.ReadFile("20250824000000_core_schema.sql")
	if err != nil {
		t.Fatalf("Failed to read core schema migration: %v", err)
	}

	contentStr := string(content)

	// Verify CASCADE delete constraints for tenant isolation
	cascadeConstraints := []string{
		"REFERENCES tenants(id) ON DELETE CASCADE",
		"REFERENCES users(id) ON DELETE CASCADE",
		"REFERENCES workflows(id) ON DELETE CASCADE",
		"REFERENCES rbac_roles(id) ON DELETE CASCADE",
	}

	for _, constraint := range cascadeConstraints {
		if !strings.Contains(contentStr, constraint) {
			t.Errorf("Core schema missing CASCADE constraint: %s", constraint)
		}
	}

	// Verify unique constraints for tenant isolation
	uniqueConstraints := []string{
		"UNIQUE(tenant_id, email)",
		"UNIQUE(tenant_id, name)",
		"UNIQUE(tenant_id, name, version)",
		"UNIQUE(tenant_id, user_id, role_id)",
	}

	for _, constraint := range uniqueConstraints {
		if !strings.Contains(contentStr, constraint) {
			t.Errorf("Core schema missing unique constraint: %s", constraint)
		}
	}
}
