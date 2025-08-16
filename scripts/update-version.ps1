# AgentFlow Version Update Script (PowerShell)
# Updates version across all relevant files in the project

param(
    [Parameter(Mandatory=$true)]
    [string]$NewVersion
)

$ErrorActionPreference = "Stop"

$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$ProjectRoot = Split-Path -Parent $ScriptDir

function Write-ColorOutput {
    param(
        [string]$Message,
        [string]$Color = "White"
    )
    Write-Host $Message -ForegroundColor $Color
}

function Test-VersionFormat {
    param([string]$Version)
    
    if ($Version -notmatch '^[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.-]+)?$') {
        Write-ColorOutput "Error: Invalid version format: $Version" "Red"
        Write-ColorOutput "Version must follow semantic versioning: MAJOR.MINOR.PATCH[-PRERELEASE]" "Yellow"
        exit 1
    }
}

function Update-FileContent {
    param(
        [string]$FilePath,
        [string]$Pattern,
        [string]$Replacement
    )
    
    if (Test-Path $FilePath) {
        try {
            $content = Get-Content $FilePath -Raw
            $newContent = $content -replace $Pattern, $Replacement
            if ($content -ne $newContent) {
                Set-Content -Path $FilePath -Value $newContent -NoNewline
                Write-ColorOutput "✓ Updated $FilePath" "Green"
            } else {
                Write-ColorOutput "⚠ No changes needed in $FilePath" "Yellow"
            }
        } catch {
            Write-ColorOutput "⚠ Could not update $FilePath`: $($_.Exception.Message)" "Yellow"
        }
    } else {
        Write-ColorOutput "⚠ File not found: $FilePath" "Yellow"
    }
}

function Main {
    Test-VersionFormat $NewVersion
    
    Write-ColorOutput "Updating AgentFlow version to: $NewVersion" "Green"
    Write-Host ""
    
    Set-Location $ProjectRoot
    
    # Update go.mod version comment
    Update-FileContent "go.mod" "// version: .*" "// version: v$NewVersion"
    
    # Update version in CLI (create if doesn't exist)
    $versionGoPath = "cmd/af/version.go"
    if (-not (Test-Path $versionGoPath)) {
        $null = New-Item -Path "cmd/af" -ItemType Directory -Force
        $versionContent = @"
package main

// Version information for AgentFlow CLI
const (
    Version = "$NewVersion"
    BuildDate = ""
    GitCommit = ""
)
"@
        Set-Content -Path $versionGoPath -Value $versionContent
        Write-ColorOutput "✓ Created $versionGoPath" "Green"
    } else {
        Update-FileContent $versionGoPath 'Version = ".*"' "Version = `"$NewVersion`""
    }
    
    # Update Dockerfile labels
    Update-FileContent "Dockerfile" 'LABEL version=".*"' "LABEL version=`"$NewVersion`""
    
    # Update docker-compose.yml if it exists
    if (Test-Path "docker-compose.yml") {
        Update-FileContent "docker-compose.yml" "image: agentflow.*" "image: agentflow:$NewVersion"
    }
    
    # Update Helm chart if it exists
    if (Test-Path "deploy/helm/Chart.yaml") {
        Update-FileContent "deploy/helm/Chart.yaml" "version: .*" "version: $NewVersion"
        Update-FileContent "deploy/helm/Chart.yaml" "appVersion: .*" "appVersion: $NewVersion"
    }
    
    # Update package.json files if they exist
    Get-ChildItem -Path . -Name "package.json" -Recurse | Where-Object { $_ -notlike "*node_modules*" } | ForEach-Object {
        Update-FileContent $_ '"version": ".*"' "`"version`": `"$NewVersion`""
    }
    
    # Update Python setup files if they exist
    @("setup.py", "pyproject.toml") | ForEach-Object {
        Get-ChildItem -Path . -Name $_ -Recurse | ForEach-Object {
            if ($_ -like "*setup.py") {
                Update-FileContent $_ 'version=".*"' "version=`"$NewVersion`""
            } else {
                Update-FileContent $_ 'version = ".*"' "version = `"$NewVersion`""
            }
        }
    }
    
    Write-Host ""
    Write-ColorOutput "Version update completed successfully!" "Green"
    Write-Host ""
    Write-ColorOutput "Next steps:" "Cyan"
    Write-ColorOutput "1. Review the changes: git diff" "White"
    Write-ColorOutput "2. Update CHANGELOG.md with release notes" "White"
    Write-ColorOutput "3. Commit the changes: git add . && git commit -m 'chore: bump version to v$NewVersion'" "White"
    Write-ColorOutput "4. Create release branch: git checkout -b release/v$NewVersion" "White"
}

Main