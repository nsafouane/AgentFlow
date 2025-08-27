# AgentFlow Backup/Restore Roundtrip Test (PowerShell)
# Tests complete backup and restore cycle with performance validation

param(
    [string]$TestDatabaseUrl = $(if ($env:AF_TEST_DATABASE_URL) { $env:AF_TEST_DATABASE_URL } else { "postgresql://agentflow:agentflow@localhost:5432/agentflow_test" }),
    [string]$BackupDir = "./test-backups",
    [int]$BaselineDatasetSize = 1000,
    [int]$MaxDurationSeconds = 300
)

$ErrorActionPreference = "Stop"

Write-Host "AgentFlow Backup/Restore Roundtrip Test"
Write-Host "======================================="
Write-Host "Test Database: $($TestDatabaseUrl -replace '@.*@', '@***@')"
Write-Host "Backup Directory: $BackupDir"
Write-Host "Baseline Dataset Size: $BaselineDatasetSize records"
Write-Host "Max Duration: $MaxDurationSeconds seconds"
Write-Host ""

# Function to log with timestamp
function Write-Log {
    param([string]$Message)
    $timestamp = Get-Date -Format "yyyy-MM-dd HH:mm:ss"
    Write-Host "[$timestamp] $Message"
}

# Function to cleanup test environment
function Invoke-Cleanup {
    Write-Log "Cleaning up test environment..."
    
    if (Test-Path $BackupDir) {
        Remove-Item -Recurse -Force $BackupDir
    }
    
    # Drop test database if it exists
    $DbName = ($TestDatabaseUrl -split '/')[-1] -replace '\?.*', ''
    $DbUrlWithoutDb = $TestDatabaseUrl -replace "/$DbName.*$", "/postgres"
    
    try {
        & psql $DbUrlWithoutDb -c "DROP DATABASE IF EXISTS $DbName;" 2>$null
    } catch {
        # Ignore errors during cleanup
    }
}

# Function to setup test database with baseline dataset
function Initialize-TestDatabase {
    Write-Log "Setting up test database with baseline dataset..."
    
    # Extract database name and create database
    $DbName = ($TestDatabaseUrl -split '/')[-1] -replace '\?.*', ''
    $DbUrlWithoutDb = $TestDatabaseUrl -replace "/$DbName.*$", "/postgres"
    
    # Drop and recreate test database
    & psql $DbUrlWithoutDb -c "DROP DATABASE IF EXISTS $DbName;" | Out-Null
    & psql $DbUrlWithoutDb -c "CREATE DATABASE $DbName;" | Out-Null
    
    # Run migrations to create schema
    if (Test-Path "migrations/20250824000000_core_schema.sql") {
        Write-Log "Applying core schema migration..."
        & psql $TestDatabaseUrl -f "migrations/20250824000000_core_schema.sql" | Out-Null
    } else {
        throw "Core schema migration not found"
    }
    
    # Generate baseline dataset
    Write-Log "Generating baseline dataset ($BaselineDatasetSize records)..."
    
    # Create test tenant
    $TenantId = [System.Guid]::NewGuid().ToString()
    & psql $TestDatabaseUrl -c "INSERT INTO tenants (id, name, created_at) VALUES ('$TenantId', 'test-tenant', NOW());" | Out-Null
    
    # Create test user
    $UserId = [System.Guid]::NewGuid().ToString()
    & psql $TestDatabaseUrl -c "INSERT INTO users (id, tenant_id, email, name, created_at) VALUES ('$UserId', '$TenantId', 'test@example.com', 'Test User', NOW());" | Out-Null
    
    # Create test agent
    $AgentId = [System.Guid]::NewGuid().ToString()
    & psql $TestDatabaseUrl -c "INSERT INTO agents (id, tenant_id, name, type, config, created_at) VALUES ('$AgentId', '$TenantId', 'test-agent', 'worker', '{}', NOW());" | Out-Null
    
    # Create test workflow
    $WorkflowId = [System.Guid]::NewGuid().ToString()
    & psql $TestDatabaseUrl -c "INSERT INTO workflows (id, tenant_id, name, definition, created_at) VALUES ('$WorkflowId', '$TenantId', 'test-workflow', '{}', NOW());" | Out-Null
    
    # Generate audit records (baseline dataset)
    Write-Log "Generating $BaselineDatasetSize audit records..."
    for ($i = 1; $i -le $BaselineDatasetSize; $i++) {
        $AuditId = [System.Guid]::NewGuid().ToString()
        & psql $TestDatabaseUrl -c "INSERT INTO audits (id, tenant_id, event_type, entity_type, entity_id, user_id, changes, created_at) VALUES ('$AuditId', '$TenantId', 'test_event', 'test_entity', '$AgentId', '$UserId', '{\"test\": $i}', NOW());" | Out-Null
        
        # Progress indicator
        if ($i % 100 -eq 0) {
            Write-Host -NoNewline "."
        }
    }
    Write-Host ""
    
    Write-Log "Baseline dataset created successfully"
}

# Function to verify data integrity
function Test-DataIntegrity {
    param([string]$Description)
    
    Write-Log "Verifying data integrity: $Description"
    
    # Count records in each table
    $TenantCount = (& psql $TestDatabaseUrl -t -c "SELECT COUNT(*) FROM tenants;").Trim()
    $UserCount = (& psql $TestDatabaseUrl -t -c "SELECT COUNT(*) FROM users;").Trim()
    $AgentCount = (& psql $TestDatabaseUrl -t -c "SELECT COUNT(*) FROM agents;").Trim()
    $WorkflowCount = (& psql $TestDatabaseUrl -t -c "SELECT COUNT(*) FROM workflows;").Trim()
    $AuditCount = (& psql $TestDatabaseUrl -t -c "SELECT COUNT(*) FROM audits;").Trim()
    
    Write-Log "Record counts: tenants=$TenantCount, users=$UserCount, agents=$AgentCount, workflows=$WorkflowCount, audits=$AuditCount"
    
    # Verify expected counts
    if ([int]$TenantCount -ne 1 -or [int]$UserCount -ne 1 -or [int]$AgentCount -ne 1 -or [int]$WorkflowCount -ne 1) {
        throw "Unexpected record counts for core entities"
    }
    
    if ([int]$AuditCount -ne $BaselineDatasetSize) {
        throw "Expected $BaselineDatasetSize audit records, found $AuditCount"
    }
    
    # Verify foreign key relationships
    $OrphanedUsers = (& psql $TestDatabaseUrl -t -c "SELECT COUNT(*) FROM users u LEFT JOIN tenants t ON u.tenant_id = t.id WHERE t.id IS NULL;").Trim()
    $OrphanedAgents = (& psql $TestDatabaseUrl -t -c "SELECT COUNT(*) FROM agents a LEFT JOIN tenants t ON a.tenant_id = t.id WHERE t.id IS NULL;").Trim()
    $OrphanedAudits = (& psql $TestDatabaseUrl -t -c "SELECT COUNT(*) FROM audits a LEFT JOIN tenants t ON a.tenant_id = t.id WHERE t.id IS NULL;").Trim()
    
    if ([int]$OrphanedUsers -ne 0 -or [int]$OrphanedAgents -ne 0 -or [int]$OrphanedAudits -ne 0) {
        throw "Found orphaned records (users=$OrphanedUsers, agents=$OrphanedAgents, audits=$OrphanedAudits)"
    }
    
    Write-Log "Data integrity verification passed"
}

# Function to run backup
function Invoke-Backup {
    Write-Log "Creating backup..."
    
    # Ensure backup directory is clean
    if (Test-Path $BackupDir) {
        Remove-Item -Recurse -Force $BackupDir
    }
    New-Item -ItemType Directory -Path $BackupDir -Force | Out-Null
    
    # Run backup script
    $env:AF_DATABASE_URL = $TestDatabaseUrl
    & powershell -ExecutionPolicy Bypass -File "scripts/backup-database.ps1" -DatabaseUrl $TestDatabaseUrl -BackupDir $BackupDir -Jobs 4 -CompressionLevel 6
    
    # Verify backup was created
    $BackupCount = (Get-ChildItem -Path $BackupDir -Filter "*_manifest.json").Count
    if ($BackupCount -eq 0) {
        throw "No backup manifest found"
    }
    
    Write-Log "Backup created successfully"
}

# Function to run restore
function Invoke-Restore {
    param([string]$BackupId)
    
    Write-Log "Restoring from backup: $BackupId"
    
    # Run restore script with automatic confirmation
    $env:AF_DATABASE_URL = $TestDatabaseUrl
    $input = "yes"
    $input | & powershell -ExecutionPolicy Bypass -File "scripts/restore-database.ps1" -BackupId $BackupId -DatabaseUrl $TestDatabaseUrl -BackupDir $BackupDir -RestoreType "full"
    
    Write-Log "Restore completed successfully"
}

# Function to simulate accidental table drop
function Invoke-DataLossSimulation {
    Write-Log "Simulating accidental table drop..."
    
    # Drop a critical table to simulate data loss
    & psql $TestDatabaseUrl -c "DROP TABLE IF EXISTS audits CASCADE;" | Out-Null
    
    # Verify table is gone
    $TableExists = (& psql $TestDatabaseUrl -t -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'audits';").Trim()
    if ([int]$TableExists -ne 0) {
        throw "Failed to simulate data loss"
    }
    
    Write-Log "Data loss simulated (audits table dropped)"
}

# Main test execution
try {
    $StartTime = Get-Date
    $TestPassed = $true
    
    # Check prerequisites
    Write-Log "Checking prerequisites..."
    if (-not (Get-Command "psql" -ErrorAction SilentlyContinue)) {
        throw "psql not found"
    }
    
    # Test Phase 1: Setup and Initial Backup
    Write-Log "Phase 1: Setup and Initial Backup"
    try {
        Initialize-TestDatabase
        Test-DataIntegrity "initial dataset"
        Invoke-Backup
    } catch {
        $TestPassed = $false
        Write-Error "Phase 1 failed: $_"
    }
    
    # Get backup ID for restore
    $BackupId = ""
    if ($TestPassed) {
        $ManifestFile = Get-ChildItem -Path $BackupDir -Filter "*_manifest.json" | Select-Object -First 1
        if ($ManifestFile) {
            $BackupId = $ManifestFile.Name -replace 'agentflow_backup_(.*)_manifest\.json', '$1'
            Write-Log "Backup ID: $BackupId"
        } else {
            $TestPassed = $false
            Write-Error "No backup manifest found"
        }
    }
    
    # Test Phase 2: Data Loss Simulation and Restore
    if ($TestPassed) {
        Write-Log "Phase 2: Data Loss Simulation and Restore"
        try {
            Invoke-DataLossSimulation
            Invoke-Restore $BackupId
        } catch {
            $TestPassed = $false
            Write-Error "Phase 2 failed: $_"
        }
    }
    
    # Test Phase 3: Post-Restore Verification
    if ($TestPassed) {
        Write-Log "Phase 3: Post-Restore Verification"
        try {
            Test-DataIntegrity "post-restore"
        } catch {
            $TestPassed = $false
            Write-Error "Phase 3 failed: $_"
        }
    }
    
    # Performance validation
    $EndTime = Get-Date
    $Duration = ($EndTime - $StartTime).TotalSeconds
    
    Write-Log "Test Duration: $([math]::Round($Duration, 2)) seconds"
    
    if ($Duration -gt $MaxDurationSeconds) {
        Write-Warning "Test took longer than expected ($([math]::Round($Duration, 2))s > ${MaxDurationSeconds}s)"
        Write-Warning "Consider optimizing backup size or database performance"
    }
    
    # Final result
    Write-Host ""
    Write-Host "======================================="
    if ($TestPassed) {
        Write-Host "✓ Backup/Restore Roundtrip Test PASSED" -ForegroundColor Green
        Write-Host "✓ Duration: $([math]::Round($Duration, 2))s (target: <${MaxDurationSeconds}s)" -ForegroundColor Green
        Write-Host "✓ Dataset: $BaselineDatasetSize records" -ForegroundColor Green
        Write-Host "✓ Data integrity verified" -ForegroundColor Green
        exit 0
    } else {
        Write-Host "✗ Backup/Restore Roundtrip Test FAILED" -ForegroundColor Red
        Write-Host "✗ Duration: $([math]::Round($Duration, 2))s" -ForegroundColor Red
        exit 1
    }

} catch {
    Write-Error "Test execution failed: $_"
    exit 1
} finally {
    Invoke-Cleanup
}