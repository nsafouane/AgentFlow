# AgentFlow Database Backup Script (PowerShell)
# Provides pg_dump with compression, parallel options, and integrity validation

param(
    [string]$DatabaseUrl = $env:AF_DATABASE_URL ?? "postgresql://agentflow:agentflow@localhost:5432/agentflow",
    [string]$BackupDir = "./backups",
    [int]$Jobs = 4,
    [int]$CompressionLevel = 6
)

$ErrorActionPreference = "Stop"

# Generate timestamp for backup files
$Timestamp = Get-Date -Format "yyyyMMdd_HHmmss"
$BackupPrefix = "agentflow_backup_$Timestamp"

# Create backup directory
New-Item -ItemType Directory -Path $BackupDir -Force | Out-Null

Write-Host "AgentFlow Database Backup Starting..."
Write-Host "Database URL: $($DatabaseUrl -replace '@.*@', '@***@')"
Write-Host "Backup Directory: $BackupDir"
Write-Host "Parallel Jobs: $Jobs"
Write-Host "Compression Level: $CompressionLevel"
Write-Host "Timestamp: $Timestamp"

# Function to generate integrity hash
function Get-FileHash256 {
    param([string]$FilePath)
    return (Get-FileHash -Path $FilePath -Algorithm SHA256).Hash.ToLower()
}

# Function to run pg_dump with error handling
function Invoke-PgDump {
    param([string[]]$Arguments)
    
    $process = Start-Process -FilePath "pg_dump" -ArgumentList $Arguments -NoNewWindow -Wait -PassThru
    if ($process.ExitCode -ne 0) {
        throw "pg_dump failed with exit code $($process.ExitCode)"
    }
}

try {
    # 1. Schema-only backup (structure without data)
    Write-Host "Creating schema-only backup..."
    $SchemaFile = Join-Path $BackupDir "${BackupPrefix}_schema.sql"
    
    Invoke-PgDump @(
        $DatabaseUrl,
        "--schema-only",
        "--no-owner", 
        "--no-privileges",
        "--verbose",
        "--file=$SchemaFile"
    )

    # Compress schema backup using 7-Zip or built-in compression
    $SchemaCompressed = "$SchemaFile.gz"
    if (Get-Command "7z" -ErrorAction SilentlyContinue) {
        & 7z a -tgzip -mx$CompressionLevel "$SchemaCompressed" "$SchemaFile" | Out-Null
    } else {
        # Fallback to PowerShell compression
        Compress-Archive -Path $SchemaFile -DestinationPath "$SchemaFile.zip" -CompressionLevel Optimal
        $SchemaCompressed = "$SchemaFile.zip"
    }
    Remove-Item $SchemaFile

    # Generate schema hash
    $SchemaHash = Get-FileHash256 $SchemaCompressed
    "$SchemaHash  $(Split-Path $SchemaCompressed -Leaf)" | Out-File -FilePath (Join-Path $BackupDir "${BackupPrefix}_schema.sha256") -Encoding utf8

    Write-Host "Schema backup completed: $(Split-Path $SchemaCompressed -Leaf)"
    Write-Host "Schema hash: $SchemaHash"

    # 2. Full data backup with parallel jobs
    Write-Host "Creating full data backup with $Jobs parallel jobs..."
    $DataDir = Join-Path $BackupDir "${BackupPrefix}_data"
    New-Item -ItemType Directory -Path $DataDir -Force | Out-Null

    Invoke-PgDump @(
        $DatabaseUrl,
        "--format=directory",
        "--jobs=$Jobs",
        "--compress=$CompressionLevel",
        "--no-owner",
        "--no-privileges", 
        "--verbose",
        "--file=$DataDir"
    )

    # Create tarball of data directory
    $DataTarball = Join-Path $BackupDir "${BackupPrefix}_data.tar.gz"
    if (Get-Command "tar" -ErrorAction SilentlyContinue) {
        & tar -czf $DataTarball -C $BackupDir (Split-Path $DataDir -Leaf)
    } else {
        # Fallback to PowerShell compression
        $DataZip = Join-Path $BackupDir "${BackupPrefix}_data.zip"
        Compress-Archive -Path $DataDir -DestinationPath $DataZip -CompressionLevel Optimal
        $DataTarball = $DataZip
    }

    # Generate data hash
    $DataHash = Get-FileHash256 $DataTarball
    "$DataHash  $(Split-Path $DataTarball -Leaf)" | Out-File -FilePath (Join-Path $BackupDir "${BackupPrefix}_data.sha256") -Encoding utf8

    Write-Host "Data backup completed: $(Split-Path $DataTarball -Leaf)"
    Write-Host "Data hash: $DataHash"

    # 3. Selective critical tables backup
    Write-Host "Creating selective backup of critical tables..."
    $SelectiveFile = Join-Path $BackupDir "${BackupPrefix}_critical.sql"

    # Critical tables for AgentFlow operations
    $CriticalTables = @("tenants", "users", "rbac_roles", "rbac_bindings", "audits")
    $TableArgs = $CriticalTables | ForEach-Object { "--table=$_" }

    $PgDumpArgs = @($DatabaseUrl, "--data-only", "--no-owner", "--no-privileges", "--verbose") + $TableArgs + @("--file=$SelectiveFile")
    Invoke-PgDump $PgDumpArgs

    # Compress selective backup
    $SelectiveCompressed = "$SelectiveFile.gz"
    if (Get-Command "7z" -ErrorAction SilentlyContinue) {
        & 7z a -tgzip -mx$CompressionLevel "$SelectiveCompressed" "$SelectiveFile" | Out-Null
    } else {
        Compress-Archive -Path $SelectiveFile -DestinationPath "$SelectiveFile.zip" -CompressionLevel Optimal
        $SelectiveCompressed = "$SelectiveFile.zip"
    }
    Remove-Item $SelectiveFile

    # Generate selective hash
    $SelectiveHash = Get-FileHash256 $SelectiveCompressed
    "$SelectiveHash  $(Split-Path $SelectiveCompressed -Leaf)" | Out-File -FilePath (Join-Path $BackupDir "${BackupPrefix}_critical.sha256") -Encoding utf8

    Write-Host "Critical tables backup completed: $(Split-Path $SelectiveCompressed -Leaf)"
    Write-Host "Critical hash: $SelectiveHash"

    # 4. Generate backup manifest
    $ManifestFile = Join-Path $BackupDir "${BackupPrefix}_manifest.json"
    $CriticalTablesJson = ($CriticalTables | ForEach-Object { "`"$_`"" }) -join ","
    
    $Manifest = @"
{
  "backup_id": "$BackupPrefix",
  "timestamp": "$Timestamp",
  "database_url": "$($DatabaseUrl -replace '@.*@', '@***@')",
  "compression_level": $CompressionLevel,
  "parallel_jobs": $Jobs,
  "files": {
    "schema": {
      "filename": "$(Split-Path $SchemaCompressed -Leaf)",
      "hash": "$SchemaHash",
      "type": "schema_only"
    },
    "data": {
      "filename": "$(Split-Path $DataTarball -Leaf)",
      "hash": "$DataHash",
      "type": "full_data"
    },
    "critical": {
      "filename": "$(Split-Path $SelectiveCompressed -Leaf)",
      "hash": "$SelectiveHash",
      "type": "critical_tables"
    }
  },
  "critical_tables": [$CriticalTablesJson]
}
"@

    $Manifest | Out-File -FilePath $ManifestFile -Encoding utf8

    # Generate manifest hash
    $ManifestHash = Get-FileHash256 $ManifestFile
    "$ManifestHash  $(Split-Path $ManifestFile -Leaf)" | Out-File -FilePath (Join-Path $BackupDir "${BackupPrefix}_manifest.sha256") -Encoding utf8

    # Clean up temporary data directory
    Remove-Item -Recurse -Force $DataDir

    # Summary
    Write-Host ""
    Write-Host "=== Backup Summary ==="
    Write-Host "Backup ID: $BackupPrefix"
    Write-Host "Schema: $(Split-Path $SchemaCompressed -Leaf) ($SchemaHash)"
    Write-Host "Data: $(Split-Path $DataTarball -Leaf) ($DataHash)"
    Write-Host "Critical: $(Split-Path $SelectiveCompressed -Leaf) ($SelectiveHash)"
    Write-Host "Manifest: $(Split-Path $ManifestFile -Leaf) ($ManifestHash)"
    Write-Host ""
    Write-Host "Backup completed successfully in: $BackupDir"
    $FileCount = (Get-ChildItem -Path $BackupDir -Filter "${BackupPrefix}*").Count
    Write-Host "Total files: $FileCount"
    $TotalSize = [math]::Round((Get-ChildItem -Path $BackupDir -Recurse | Measure-Object -Property Length -Sum).Sum / 1MB, 2)
    Write-Host "Total size: ${TotalSize}MB"

} catch {
    Write-Error "Backup failed: $_"
    exit 1
}