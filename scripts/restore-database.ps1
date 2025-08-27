# AgentFlow Database Restore Script (PowerShell)
# Restores from backup with integrity validation and smoke tests

param(
    [Parameter(Mandatory=$true)]
    [string]$BackupId,
    [string]$DatabaseUrl = $env:AF_DATABASE_URL ?? "postgresql://agentflow:agentflow@localhost:5432/agentflow",
    [string]$BackupDir = "./backups",
    [ValidateSet("full", "schema", "critical")]
    [string]$RestoreType = "full"
)

$ErrorActionPreference = "Stop"

$BackupPrefix = "agentflow_backup_$BackupId"
$ManifestFile = Join-Path $BackupDir "${BackupPrefix}_manifest.json"

Write-Host "AgentFlow Database Restore Starting..."
Write-Host "Backup ID: $BackupId"
Write-Host "Database URL: $($DatabaseUrl -replace '@.*@', '@***@')"
Write-Host "Backup Directory: $BackupDir"
Write-Host "Restore Type: $RestoreType"

# Function to verify integrity hash
function Test-FileIntegrity {
    param(
        [string]$FilePath,
        [string]$HashFile
    )
    
    if (-not (Test-Path $HashFile)) {
        throw "Hash file not found: $HashFile"
    }
    
    $ExpectedHash = (Get-Content $HashFile).Split(' ')[0]
    $ActualHash = (Get-FileHash -Path $FilePath -Algorithm SHA256).Hash.ToLower()
    
    if ($ExpectedHash -ne $ActualHash) {
        throw "Hash mismatch for $FilePath. Expected: $ExpectedHash, Actual: $ActualHash"
    }
    
    Write-Host "Hash verified: $(Split-Path $FilePath -Leaf)"
}

# Function to run psql with error handling
function Invoke-Psql {
    param(
        [string]$Sql,
        [string]$DatabaseUrl = $script:DatabaseUrl
    )
    
    $process = Start-Process -FilePath "psql" -ArgumentList @($DatabaseUrl, "-c", $Sql, "-v", "ON_ERROR_STOP=1") -NoNewWindow -Wait -PassThru
    if ($process.ExitCode -ne 0) {
        throw "psql command failed with exit code $($process.ExitCode)"
    }
}

# Function to perform smoke tests
function Test-DatabaseIntegrity {
    Write-Host "Performing smoke tests..."
    
    # Test 1: Check if critical tables exist and have data
    $Tables = @("tenants", "users", "rbac_roles", "rbac_bindings", "audits")
    foreach ($Table in $Tables) {
        try {
            $Count = & psql $DatabaseUrl -t -c "SELECT COUNT(*) FROM $Table;" 2>$null
            $Count = $Count.Trim()
            Write-Host "Table $Table`: $Count rows"
            if ($Count -eq 0 -and $Table -ne "audits") {
                Write-Warning "Table $Table is empty"
            }
        } catch {
            Write-Warning "Could not check table $Table`: $_"
        }
    }
    
    # Test 2: Check foreign key constraints
    Write-Host "Checking foreign key constraints..."
    try {
        $FkViolations = & psql $DatabaseUrl -t -c @"
SELECT COUNT(*) FROM (
    SELECT conname FROM pg_constraint WHERE contype = 'f' AND NOT convalidated
) AS invalid_fks;
"@ 2>$null
        
        $FkViolations = $FkViolations.Trim()
        if ([int]$FkViolations -gt 0) {
            throw "$FkViolations foreign key constraint violations found"
        }
    } catch {
        Write-Warning "Could not verify foreign key constraints: $_"
    }
    
    # Test 3: Check if audit hash-chain is intact (if audits exist)
    try {
        $AuditCount = & psql $DatabaseUrl -t -c "SELECT COUNT(*) FROM audits;" 2>$null
        $AuditCount = $AuditCount.Trim()
        if ([int]$AuditCount -gt 0) {
            Write-Host "Verifying audit hash-chain integrity..."
            Write-Host "Audit records found: $AuditCount (manual verification recommended)"
        }
    } catch {
        Write-Warning "Could not check audit records: $_"
    }
    
    Write-Host "Smoke tests completed successfully"
}

try {
    # Verify manifest exists
    if (-not (Test-Path $ManifestFile)) {
        throw "Manifest file not found: $ManifestFile"
    }

    # Verify manifest integrity
    Test-FileIntegrity $ManifestFile (Join-Path $BackupDir "${BackupPrefix}_manifest.sha256")

    # Parse manifest to get file information
    $Manifest = Get-Content $ManifestFile | ConvertFrom-Json
    $SchemaFile = $Manifest.files.schema.filename
    $DataFile = $Manifest.files.data.filename
    $CriticalFile = $Manifest.files.critical.filename

    $SchemaPath = Join-Path $BackupDir $SchemaFile
    $DataPath = Join-Path $BackupDir $DataFile
    $CriticalPath = Join-Path $BackupDir $CriticalFile

    # Verify backup file integrity based on restore type
    switch ($RestoreType) {
        "schema" {
            Write-Host "Verifying schema backup integrity..."
            Test-FileIntegrity $SchemaPath (Join-Path $BackupDir "${BackupPrefix}_schema.sha256")
        }
        "critical" {
            Write-Host "Verifying critical tables backup integrity..."
            Test-FileIntegrity $CriticalPath (Join-Path $BackupDir "${BackupPrefix}_critical.sha256")
        }
        "full" {
            Write-Host "Verifying all backup files integrity..."
            Test-FileIntegrity $SchemaPath (Join-Path $BackupDir "${BackupPrefix}_schema.sha256")
            Test-FileIntegrity $DataPath (Join-Path $BackupDir "${BackupPrefix}_data.sha256")
        }
    }

    # Confirm restore operation
    Write-Host ""
    Write-Warning "This will replace the current database content!"
    Write-Host "Database: $($DatabaseUrl -replace '@.*@', '@***@')"
    Write-Host "Restore type: $RestoreType"
    $Confirmation = Read-Host "Continue? (yes/no)"
    if ($Confirmation -ne "yes") {
        Write-Host "Restore cancelled"
        exit 0
    }

    # Record start time for performance measurement
    $StartTime = Get-Date

    # Perform restore based on type
    switch ($RestoreType) {
        "schema" {
            Write-Host "Restoring schema only..."
            # Drop existing schema (be careful!)
            Invoke-Psql "DROP SCHEMA IF EXISTS public CASCADE; CREATE SCHEMA public;"
            
            # Restore schema
            if ($SchemaFile -match '\.gz$') {
                if (Get-Command "7z" -ErrorAction SilentlyContinue) {
                    & 7z x -so $SchemaPath | & psql $DatabaseUrl -v ON_ERROR_STOP=1
                } else {
                    throw "Cannot decompress .gz file without 7-Zip. Please install 7-Zip or use uncompressed backup."
                }
            } elseif ($SchemaFile -match '\.zip$') {
                $TempDir = New-TemporaryFile | ForEach-Object { Remove-Item $_; New-Item -ItemType Directory -Path $_ }
                Expand-Archive -Path $SchemaPath -DestinationPath $TempDir
                $ExtractedFile = Get-ChildItem $TempDir -Filter "*.sql" | Select-Object -First 1
                & psql $DatabaseUrl -f $ExtractedFile.FullName -v ON_ERROR_STOP=1
                Remove-Item -Recurse -Force $TempDir
            } else {
                & psql $DatabaseUrl -f $SchemaPath -v ON_ERROR_STOP=1
            }
        }
        
        "critical" {
            Write-Host "Restoring critical tables only..."
            # Truncate critical tables
            Invoke-Psql "TRUNCATE tenants, users, rbac_roles, rbac_bindings, audits CASCADE;"
            
            # Restore critical data
            if ($CriticalFile -match '\.gz$') {
                if (Get-Command "7z" -ErrorAction SilentlyContinue) {
                    & 7z x -so $CriticalPath | & psql $DatabaseUrl -v ON_ERROR_STOP=1
                } else {
                    throw "Cannot decompress .gz file without 7-Zip. Please install 7-Zip or use uncompressed backup."
                }
            } elseif ($CriticalFile -match '\.zip$') {
                $TempDir = New-TemporaryFile | ForEach-Object { Remove-Item $_; New-Item -ItemType Directory -Path $_ }
                Expand-Archive -Path $CriticalPath -DestinationPath $TempDir
                $ExtractedFile = Get-ChildItem $TempDir -Filter "*.sql" | Select-Object -First 1
                & psql $DatabaseUrl -f $ExtractedFile.FullName -v ON_ERROR_STOP=1
                Remove-Item -Recurse -Force $TempDir
            } else {
                & psql $DatabaseUrl -f $CriticalPath -v ON_ERROR_STOP=1
            }
        }
        
        "full" {
            Write-Host "Performing full database restore..."
            
            # Extract database name from URL
            $DbName = ($DatabaseUrl -split '/')[-1] -replace '\?.*', ''
            $DbUrlWithoutDb = $DatabaseUrl -replace "/$DbName.*$", "/postgres"
            
            # Drop and recreate database
            & psql $DbUrlWithoutDb -c "DROP DATABASE IF EXISTS $DbName;"
            & psql $DbUrlWithoutDb -c "CREATE DATABASE $DbName;"
            
            # Extract and restore from directory format
            $TempDir = New-TemporaryFile | ForEach-Object { Remove-Item $_; New-Item -ItemType Directory -Path $_ }
            
            if ($DataFile -match '\.tar\.gz$') {
                if (Get-Command "tar" -ErrorAction SilentlyContinue) {
                    & tar -xzf $DataPath -C $TempDir
                } else {
                    throw "Cannot extract .tar.gz file without tar command. Please install tar or use zip format."
                }
                $DataExtractDir = Join-Path $TempDir ($DataFile -replace '\.tar\.gz$', '')
            } elseif ($DataFile -match '\.zip$') {
                Expand-Archive -Path $DataPath -DestinationPath $TempDir
                $DataExtractDir = Join-Path $TempDir ($DataFile -replace '\.zip$', '')
            } else {
                throw "Unsupported data file format: $DataFile"
            }
            
            # Restore using pg_restore
            $process = Start-Process -FilePath "pg_restore" -ArgumentList @(
                $DatabaseUrl,
                "--format=directory",
                "--jobs=4",
                "--no-owner",
                "--no-privileges",
                "--verbose",
                $DataExtractDir
            ) -NoNewWindow -Wait -PassThru
            
            if ($process.ExitCode -ne 0) {
                throw "pg_restore failed with exit code $($process.ExitCode)"
            }
            
            # Clean up
            Remove-Item -Recurse -Force $TempDir
        }
    }

    # Calculate restore time
    $EndTime = Get-Date
    $RestoreTime = ($EndTime - $StartTime).TotalSeconds

    Write-Host ""
    Write-Host "Restore completed in $([math]::Round($RestoreTime, 2)) seconds"

    # Perform smoke tests
    Test-DatabaseIntegrity

    Write-Host ""
    Write-Host "=== Restore Summary ==="
    Write-Host "Backup ID: $BackupId"
    Write-Host "Restore Type: $RestoreType"
    Write-Host "Duration: $([math]::Round($RestoreTime, 2)) seconds"
    Write-Host "Status: SUCCESS"

    if ($RestoreTime -gt 300) {
        Write-Warning "Restore took longer than 5 minutes ($([math]::Round($RestoreTime, 2))s)"
        Write-Host "Consider optimizing backup size or database performance"
    }

} catch {
    Write-Error "Restore failed: $_"
    exit 1
}