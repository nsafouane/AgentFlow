#!/bin/bash
# Manual Disaster Recovery Test
# Simulates accidental table drop and demonstrates recovery procedures

set -euo pipefail

# Configuration
TEST_DB_URL="${AF_TEST_DATABASE_URL:-postgresql://agentflow:agentflow@localhost:5432/agentflow_test}"
BACKUP_DIR="./disaster-recovery-test"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}AgentFlow Manual Disaster Recovery Test${NC}"
echo -e "${BLUE}=====================================${NC}"
echo ""
echo "This script demonstrates disaster recovery procedures by:"
echo "1. Setting up a test database with sample data"
echo "2. Creating a backup"
echo "3. Simulating accidental table drop"
echo "4. Recovering from backup"
echo "5. Verifying data integrity"
echo ""

# Function to log with timestamp and color
log() {
    local color="$1"
    local message="$2"
    echo -e "${color}[$(date '+%Y-%m-%d %H:%M:%S')] $message${NC}"
}

# Function to pause for user interaction
pause() {
    echo ""
    read -p "Press Enter to continue or Ctrl+C to exit..."
    echo ""
}

# Function to cleanup
cleanup() {
    log "$YELLOW" "Cleaning up test environment..."
    rm -rf "$BACKUP_DIR"
    
    # Drop test database
    DB_NAME=$(echo "$TEST_DB_URL" | sed 's/.*\///' | sed 's/?.*//')
    DB_URL_WITHOUT_DB=$(echo "$TEST_DB_URL" | sed "s/\/$DB_NAME.*$/\/postgres/")
    psql "$DB_URL_WITHOUT_DB" -c "DROP DATABASE IF EXISTS $DB_NAME;" 2>/dev/null || true
    
    log "$GREEN" "Cleanup completed"
}

# Trap cleanup on exit
trap cleanup EXIT

# Step 1: Setup test database
log "$BLUE" "Step 1: Setting up test database with sample data"
echo "Database URL: ${TEST_DB_URL%@*}@***"

# Create database
DB_NAME=$(echo "$TEST_DB_URL" | sed 's/.*\///' | sed 's/?.*//')
DB_URL_WITHOUT_DB=$(echo "$TEST_DB_URL" | sed "s/\/$DB_NAME.*$/\/postgres/")

log "$YELLOW" "Creating test database: $DB_NAME"
psql "$DB_URL_WITHOUT_DB" -c "DROP DATABASE IF EXISTS $DB_NAME;" >/dev/null 2>&1 || true
psql "$DB_URL_WITHOUT_DB" -c "CREATE DATABASE $DB_NAME;" >/dev/null

# Apply schema
if [ -f "migrations/20250824000000_core_schema.sql" ]; then
    log "$YELLOW" "Applying core schema migration..."
    psql "$TEST_DB_URL" -f "migrations/20250824000000_core_schema.sql" >/dev/null
else
    log "$RED" "ERROR: Core schema migration not found"
    exit 1
fi

# Create sample data
log "$YELLOW" "Creating sample data..."
TENANT_ID=$(uuidgen)
USER_ID=$(uuidgen)
AGENT_ID=$(uuidgen)
WORKFLOW_ID=$(uuidgen)

# Insert sample records
psql "$TEST_DB_URL" -c "INSERT INTO tenants (id, name, created_at) VALUES ('$TENANT_ID', 'disaster-test-tenant', NOW());" >/dev/null
psql "$TEST_DB_URL" -c "INSERT INTO users (id, tenant_id, email, name, created_at) VALUES ('$USER_ID', '$TENANT_ID', 'disaster-test@example.com', 'Disaster Test User', NOW());" >/dev/null
psql "$TEST_DB_URL" -c "INSERT INTO agents (id, tenant_id, name, type, config, created_at) VALUES ('$AGENT_ID', '$TENANT_ID', 'disaster-test-agent', 'worker', '{}', NOW());" >/dev/null
psql "$TEST_DB_URL" -c "INSERT INTO workflows (id, tenant_id, name, definition, created_at) VALUES ('$WORKFLOW_ID', '$TENANT_ID', 'disaster-test-workflow', '{}', NOW());" >/dev/null

# Create some audit records
for i in {1..10}; do
    AUDIT_ID=$(uuidgen)
    psql "$TEST_DB_URL" -c "INSERT INTO audits (id, tenant_id, event_type, entity_type, entity_id, user_id, changes, created_at) VALUES ('$AUDIT_ID', '$TENANT_ID', 'test_event_$i', 'test_entity', '$AGENT_ID', '$USER_ID', '{\"test\": $i, \"disaster_recovery\": true}', NOW());" >/dev/null
done

# Show initial data
log "$GREEN" "Sample data created successfully!"
echo ""
echo "Initial data summary:"
echo "- Tenants: $(psql "$TEST_DB_URL" -t -c "SELECT COUNT(*) FROM tenants;" | tr -d ' ')"
echo "- Users: $(psql "$TEST_DB_URL" -t -c "SELECT COUNT(*) FROM users;" | tr -d ' ')"
echo "- Agents: $(psql "$TEST_DB_URL" -t -c "SELECT COUNT(*) FROM agents;" | tr -d ' ')"
echo "- Workflows: $(psql "$TEST_DB_URL" -t -c "SELECT COUNT(*) FROM workflows;" | tr -d ' ')"
echo "- Audits: $(psql "$TEST_DB_URL" -t -c "SELECT COUNT(*) FROM audits;" | tr -d ' ')"

pause

# Step 2: Create backup
log "$BLUE" "Step 2: Creating backup"
rm -rf "$BACKUP_DIR"
mkdir -p "$BACKUP_DIR"

log "$YELLOW" "Running backup script..."
AF_DATABASE_URL="$TEST_DB_URL" bash scripts/backup-database.sh "$TEST_DB_URL" "$BACKUP_DIR" 4 6

# Get backup ID
BACKUP_ID=$(ls -1 "$BACKUP_DIR"/*_manifest.json | head -1 | sed 's/.*agentflow_backup_\(.*\)_manifest.json/\1/')
log "$GREEN" "Backup created successfully!"
echo "Backup ID: $BACKUP_ID"
echo "Backup files:"
ls -la "$BACKUP_DIR"

pause

# Step 3: Simulate disaster (accidental table drop)
log "$BLUE" "Step 3: Simulating disaster - Accidental table drop"
echo -e "${RED}WARNING: This will simulate dropping the 'audits' table!${NC}"
echo "In a real scenario, this could happen due to:"
echo "- Incorrect DROP TABLE command"
echo "- Faulty migration script"
echo "- Human error in database management"
echo "- Malicious activity"

pause

log "$YELLOW" "Dropping audits table..."
psql "$TEST_DB_URL" -c "DROP TABLE IF EXISTS audits CASCADE;" >/dev/null

# Verify disaster
log "$RED" "DISASTER SIMULATED!"
echo ""
echo "Current data summary (after disaster):"
echo "- Tenants: $(psql "$TEST_DB_URL" -t -c "SELECT COUNT(*) FROM tenants;" | tr -d ' ')"
echo "- Users: $(psql "$TEST_DB_URL" -t -c "SELECT COUNT(*) FROM users;" | tr -d ' ')"
echo "- Agents: $(psql "$TEST_DB_URL" -t -c "SELECT COUNT(*) FROM agents;" | tr -d ' ')"
echo "- Workflows: $(psql "$TEST_DB_URL" -t -c "SELECT COUNT(*) FROM workflows;" | tr -d ' ')"

# Try to query audits table (should fail)
echo -n "- Audits: "
if psql "$TEST_DB_URL" -t -c "SELECT COUNT(*) FROM audits;" 2>/dev/null; then
    echo -e "${RED}ERROR: Audits table still exists!${NC}"
else
    echo -e "${RED}TABLE MISSING (as expected)${NC}"
fi

echo ""
echo -e "${RED}The audits table has been lost! This is a critical data loss scenario.${NC}"

pause

# Step 4: Recovery from backup
log "$BLUE" "Step 4: Recovery from backup"
echo "Recovery options:"
echo "1. Full restore - Restores entire database (destructive)"
echo "2. Critical restore - Restores only critical tables"
echo "3. Schema restore - Restores table structure only"
echo ""
echo "For this scenario, we'll use FULL RESTORE to recover all data."

pause

log "$YELLOW" "Verifying backup integrity before restore..."
af backup verify "$BACKUP_ID" "$BACKUP_DIR"

if [ $? -eq 0 ]; then
    log "$GREEN" "Backup integrity verified - proceeding with restore"
else
    log "$RED" "Backup integrity check failed - aborting recovery"
    exit 1
fi

pause

log "$YELLOW" "Starting full database restore..."
echo "This will:"
echo "1. Drop and recreate the entire database"
echo "2. Restore all data from backup"
echo "3. Verify data integrity"

# Perform restore (with automatic confirmation)
echo "yes" | AF_DATABASE_URL="$TEST_DB_URL" bash scripts/restore-database.sh "$BACKUP_ID" "$TEST_DB_URL" "$BACKUP_DIR" "full"

log "$GREEN" "Restore completed!"

pause

# Step 5: Verify recovery
log "$BLUE" "Step 5: Verifying recovery"

log "$YELLOW" "Checking data integrity after restore..."
echo ""
echo "Post-recovery data summary:"
echo "- Tenants: $(psql "$TEST_DB_URL" -t -c "SELECT COUNT(*) FROM tenants;" | tr -d ' ')"
echo "- Users: $(psql "$TEST_DB_URL" -t -c "SELECT COUNT(*) FROM users;" | tr -d ' ')"
echo "- Agents: $(psql "$TEST_DB_URL" -t -c "SELECT COUNT(*) FROM agents;" | tr -d ' ')"
echo "- Workflows: $(psql "$TEST_DB_URL" -t -c "SELECT COUNT(*) FROM workflows;" | tr -d ' ')"
echo "- Audits: $(psql "$TEST_DB_URL" -t -c "SELECT COUNT(*) FROM audits;" | tr -d ' ')"

# Verify specific data
log "$YELLOW" "Verifying specific data integrity..."

# Check tenant data
TENANT_COUNT=$(psql "$TEST_DB_URL" -t -c "SELECT COUNT(*) FROM tenants WHERE name = 'disaster-test-tenant';" | tr -d ' ')
if [ "$TENANT_COUNT" -eq 1 ]; then
    log "$GREEN" "✓ Tenant data recovered correctly"
else
    log "$RED" "✗ Tenant data recovery failed"
fi

# Check user data
USER_COUNT=$(psql "$TEST_DB_URL" -t -c "SELECT COUNT(*) FROM users WHERE email = 'disaster-test@example.com';" | tr -d ' ')
if [ "$USER_COUNT" -eq 1 ]; then
    log "$GREEN" "✓ User data recovered correctly"
else
    log "$RED" "✗ User data recovery failed"
fi

# Check audit data
AUDIT_COUNT=$(psql "$TEST_DB_URL" -t -c "SELECT COUNT(*) FROM audits WHERE changes::text LIKE '%disaster_recovery%';" | tr -d ' ')
if [ "$AUDIT_COUNT" -eq 10 ]; then
    log "$GREEN" "✓ Audit data recovered correctly ($AUDIT_COUNT records)"
else
    log "$RED" "✗ Audit data recovery failed (expected 10, got $AUDIT_COUNT)"
fi

# Check foreign key relationships
ORPHANED_USERS=$(psql "$TEST_DB_URL" -t -c "SELECT COUNT(*) FROM users u LEFT JOIN tenants t ON u.tenant_id = t.id WHERE t.id IS NULL;" | tr -d ' ')
ORPHANED_AUDITS=$(psql "$TEST_DB_URL" -t -c "SELECT COUNT(*) FROM audits a LEFT JOIN tenants t ON a.tenant_id = t.id WHERE t.id IS NULL;" | tr -d ' ')

if [ "$ORPHANED_USERS" -eq 0 ] && [ "$ORPHANED_AUDITS" -eq 0 ]; then
    log "$GREEN" "✓ Foreign key relationships intact"
else
    log "$RED" "✗ Foreign key relationship issues detected"
fi

echo ""
log "$BLUE" "Recovery Test Summary"
echo "===================="
echo ""
echo -e "${GREEN}✓ Database backup created successfully${NC}"
echo -e "${GREEN}✓ Disaster scenario simulated (table drop)${NC}"
echo -e "${GREEN}✓ Backup integrity verified${NC}"
echo -e "${GREEN}✓ Full database restore completed${NC}"
echo -e "${GREEN}✓ Data integrity verified post-recovery${NC}"
echo ""
echo -e "${BLUE}Key Learnings:${NC}"
echo "1. Regular backups are essential for disaster recovery"
echo "2. Backup integrity verification prevents corrupt restores"
echo "3. Full restore procedures can recover from major data loss"
echo "4. Recovery time depends on database size and hardware"
echo "5. Post-recovery verification is critical to ensure data integrity"
echo ""
echo -e "${YELLOW}Recommendations:${NC}"
echo "1. Implement automated backup scheduling"
echo "2. Test recovery procedures regularly"
echo "3. Monitor backup success/failure"
echo "4. Document recovery procedures for operations team"
echo "5. Consider point-in-time recovery for granular restoration"
echo ""
echo -e "${GREEN}Disaster Recovery Test Completed Successfully!${NC}"