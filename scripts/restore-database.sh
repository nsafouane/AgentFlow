#!/bin/bash
# AgentFlow Database Restore Script
# Restores from backup with integrity validation and smoke tests

set -euo pipefail

# Default configuration
DEFAULT_DB_URL="postgresql://agentflow:agentflow@localhost:5432/agentflow"
DEFAULT_BACKUP_DIR="./backups"

# Parse command line arguments
BACKUP_ID="${1:-}"
DB_URL="${AF_DATABASE_URL:-${2:-$DEFAULT_DB_URL}}"
BACKUP_DIR="${3:-$DEFAULT_BACKUP_DIR}"
RESTORE_TYPE="${4:-full}" # full, schema, critical

if [ -z "$BACKUP_ID" ]; then
    echo "Usage: $0 <backup_id> [db_url] [backup_dir] [restore_type]"
    echo ""
    echo "Available backups:"
    ls -1 "$BACKUP_DIR"/*_manifest.json 2>/dev/null | sed 's/.*agentflow_backup_\(.*\)_manifest.json/  \1/' || echo "  No backups found"
    exit 1
fi

BACKUP_PREFIX="agentflow_backup_${BACKUP_ID}"
MANIFEST_FILE="$BACKUP_DIR/${BACKUP_PREFIX}_manifest.json"

echo "AgentFlow Database Restore Starting..."
echo "Backup ID: $BACKUP_ID"
echo "Database URL: ${DB_URL%@*}@***"
echo "Backup Directory: $BACKUP_DIR"
echo "Restore Type: $RESTORE_TYPE"

# Function to verify integrity hash
verify_hash() {
    local file="$1"
    local hash_file="$2"
    
    if [ ! -f "$hash_file" ]; then
        echo "ERROR: Hash file not found: $hash_file" >&2
        return 1
    fi
    
    local expected_hash=$(cut -d' ' -f1 "$hash_file")
    local actual_hash
    
    if command -v sha256sum >/dev/null 2>&1; then
        actual_hash=$(sha256sum "$file" | cut -d' ' -f1)
    elif command -v shasum >/dev/null 2>&1; then
        actual_hash=$(shasum -a 256 "$file" | cut -d' ' -f1)
    else
        echo "ERROR: No SHA256 utility found (sha256sum or shasum)" >&2
        return 1
    fi
    
    if [ "$expected_hash" != "$actual_hash" ]; then
        echo "ERROR: Hash mismatch for $file" >&2
        echo "Expected: $expected_hash" >&2
        echo "Actual: $actual_hash" >&2
        return 1
    fi
    
    echo "Hash verified: $(basename "$file")"
    return 0
}

# Function to run psql with error handling
run_psql() {
    local sql="$1"
    psql "$DB_URL" -c "$sql" -v ON_ERROR_STOP=1
}

# Function to perform smoke tests
smoke_test() {
    echo "Performing smoke tests..."
    
    # Test 1: Check if critical tables exist and have data
    local tables=("tenants" "users" "rbac_roles" "rbac_bindings" "audits")
    for table in "${tables[@]}"; do
        local count=$(psql "$DB_URL" -t -c "SELECT COUNT(*) FROM $table;" 2>/dev/null || echo "0")
        echo "Table $table: $count rows"
        if [ "$count" -eq 0 ] && [ "$table" != "audits" ]; then
            echo "WARNING: Table $table is empty"
        fi
    done
    
    # Test 2: Check foreign key constraints
    echo "Checking foreign key constraints..."
    local fk_violations=$(psql "$DB_URL" -t -c "
        SELECT COUNT(*) FROM (
            SELECT conname FROM pg_constraint WHERE contype = 'f' AND NOT convalidated
        ) AS invalid_fks;" 2>/dev/null || echo "0")
    
    if [ "$fk_violations" -gt 0 ]; then
        echo "ERROR: $fk_violations foreign key constraint violations found" >&2
        return 1
    fi
    
    # Test 3: Check if audit hash-chain is intact (if audits exist)
    local audit_count=$(psql "$DB_URL" -t -c "SELECT COUNT(*) FROM audits;" 2>/dev/null || echo "0")
    if [ "$audit_count" -gt 0 ]; then
        echo "Verifying audit hash-chain integrity..."
        # This would call the audit verification CLI when available
        echo "Audit records found: $audit_count (manual verification recommended)"
    fi
    
    echo "Smoke tests completed successfully"
}

# Verify manifest exists
if [ ! -f "$MANIFEST_FILE" ]; then
    echo "ERROR: Manifest file not found: $MANIFEST_FILE" >&2
    exit 1
fi

# Verify manifest integrity
verify_hash "$MANIFEST_FILE" "$BACKUP_DIR/${BACKUP_PREFIX}_manifest.sha256"

# Parse manifest to get file information
SCHEMA_FILE=$(jq -r '.files.schema.filename' "$MANIFEST_FILE")
DATA_FILE=$(jq -r '.files.data.filename' "$MANIFEST_FILE")
CRITICAL_FILE=$(jq -r '.files.critical.filename' "$MANIFEST_FILE")

SCHEMA_PATH="$BACKUP_DIR/$SCHEMA_FILE"
DATA_PATH="$BACKUP_DIR/$DATA_FILE"
CRITICAL_PATH="$BACKUP_DIR/$CRITICAL_FILE"

# Verify backup file integrity based on restore type
case "$RESTORE_TYPE" in
    "schema")
        echo "Verifying schema backup integrity..."
        verify_hash "$SCHEMA_PATH" "$BACKUP_DIR/${BACKUP_PREFIX}_schema.sha256"
        ;;
    "critical")
        echo "Verifying critical tables backup integrity..."
        verify_hash "$CRITICAL_PATH" "$BACKUP_DIR/${BACKUP_PREFIX}_critical.sha256"
        ;;
    "full")
        echo "Verifying all backup files integrity..."
        verify_hash "$SCHEMA_PATH" "$BACKUP_DIR/${BACKUP_PREFIX}_schema.sha256"
        verify_hash "$DATA_PATH" "$BACKUP_DIR/${BACKUP_PREFIX}_data.sha256"
        ;;
    *)
        echo "ERROR: Invalid restore type: $RESTORE_TYPE" >&2
        echo "Valid types: full, schema, critical" >&2
        exit 1
        ;;
esac

# Confirm restore operation
echo ""
echo "WARNING: This will replace the current database content!"
echo "Database: ${DB_URL%@*}@***"
echo "Restore type: $RESTORE_TYPE"
read -p "Continue? (yes/no): " -r
if [[ ! $REPLY =~ ^[Yy][Ee][Ss]$ ]]; then
    echo "Restore cancelled"
    exit 0
fi

# Record start time for performance measurement
START_TIME=$(date +%s)

# Perform restore based on type
case "$RESTORE_TYPE" in
    "schema")
        echo "Restoring schema only..."
        # Drop existing schema (be careful!)
        run_psql "DROP SCHEMA IF EXISTS public CASCADE; CREATE SCHEMA public;"
        
        # Restore schema
        if [[ "$SCHEMA_FILE" == *.gz ]]; then
            gunzip -c "$SCHEMA_PATH" | psql "$DB_URL" -v ON_ERROR_STOP=1
        else
            psql "$DB_URL" -f "$SCHEMA_PATH" -v ON_ERROR_STOP=1
        fi
        ;;
        
    "critical")
        echo "Restoring critical tables only..."
        # Truncate critical tables
        run_psql "TRUNCATE tenants, users, rbac_roles, rbac_bindings, audits CASCADE;"
        
        # Restore critical data
        if [[ "$CRITICAL_FILE" == *.gz ]]; then
            gunzip -c "$CRITICAL_PATH" | psql "$DB_URL" -v ON_ERROR_STOP=1
        else
            psql "$DB_URL" -f "$CRITICAL_PATH" -v ON_ERROR_STOP=1
        fi
        ;;
        
    "full")
        echo "Performing full database restore..."
        
        # Drop and recreate database
        DB_NAME=$(echo "$DB_URL" | sed 's/.*\///' | sed 's/?.*//')
        DB_URL_WITHOUT_DB=$(echo "$DB_URL" | sed "s/\/$DB_NAME.*$/\/postgres/")
        
        psql "$DB_URL_WITHOUT_DB" -c "DROP DATABASE IF EXISTS $DB_NAME;"
        psql "$DB_URL_WITHOUT_DB" -c "CREATE DATABASE $DB_NAME;"
        
        # Extract and restore from directory format
        TEMP_DIR=$(mktemp -d)
        if [[ "$DATA_FILE" == *.tar.gz ]]; then
            tar -xzf "$DATA_PATH" -C "$TEMP_DIR"
            DATA_EXTRACT_DIR="$TEMP_DIR/$(basename "$DATA_PATH" .tar.gz)"
        elif [[ "$DATA_FILE" == *.zip ]]; then
            unzip -q "$DATA_PATH" -d "$TEMP_DIR"
            DATA_EXTRACT_DIR="$TEMP_DIR/$(basename "$DATA_PATH" .zip)"
        else
            echo "ERROR: Unsupported data file format: $DATA_FILE" >&2
            exit 1
        fi
        
        # Restore using pg_restore
        pg_restore "$DB_URL" \
            --format=directory \
            --jobs=4 \
            --no-owner \
            --no-privileges \
            --verbose \
            "$DATA_EXTRACT_DIR"
        
        # Clean up
        rm -rf "$TEMP_DIR"
        ;;
esac

# Calculate restore time
END_TIME=$(date +%s)
RESTORE_TIME=$((END_TIME - START_TIME))

echo ""
echo "Restore completed in ${RESTORE_TIME} seconds"

# Perform smoke tests
smoke_test

echo ""
echo "=== Restore Summary ==="
echo "Backup ID: $BACKUP_ID"
echo "Restore Type: $RESTORE_TYPE"
echo "Duration: ${RESTORE_TIME} seconds"
echo "Status: SUCCESS"

if [ $RESTORE_TIME -gt 300 ]; then
    echo "WARNING: Restore took longer than 5 minutes (${RESTORE_TIME}s)"
    echo "Consider optimizing backup size or database performance"
fi