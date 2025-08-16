# Test script for version management functionality (PowerShell)

param(
    [switch]$Verbose
)

$ErrorActionPreference = "Continue"

function Write-ColorOutput {
    param(
        [string]$Message,
        [string]$Color = "White"
    )
    Write-Host $Message -ForegroundColor $Color
}

function Test-VersionParsing {
    Write-ColorOutput "`nTest 1: Version Parsing Script" "Yellow"
    
    if (Test-Path "parse-version.ps1") {
        Write-Host "Testing valid version parsing..."
        
        $testVersions = @("1.2.3", "0.1.0", "2.0.0-alpha.1", "0.5.0-beta.2", "1.0.0-rc.1")
        
        foreach ($version in $testVersions) {
            Write-Host "  Testing $version... " -NoNewline
            try {
                $output = & .\parse-version.ps1 -Version $version -Command parse 2>$null
                if ($output) {
                    Write-ColorOutput "✓" "Green"
                } else {
                    Write-ColorOutput "✗" "Red"
                }
            } catch {
                Write-ColorOutput "✗" "Red"
                if ($Verbose) {
                    Write-Host "    Error: $($_.Exception.Message)"
                }
            }
        }
        
        Write-Host "Testing invalid version rejection..."
        $invalidVersions = @("1.2", "1.2.3.4", "1.2.3a", "invalid")
        
        foreach ($version in $invalidVersions) {
            Write-Host "  Testing $version (should fail)... " -NoNewline
            try {
                $output = & .\parse-version.ps1 -Version $version -Command parse 2>$null
                if ($output) {
                    Write-ColorOutput "✗ (should have failed)" "Red"
                } else {
                    Write-ColorOutput "✓" "Green"
                }
            } catch {
                Write-ColorOutput "✓" "Green"
            }
        }
    } else {
        Write-ColorOutput "✗ parse-version.ps1 not found" "Red"
    }
}

function Test-VersionIncrement {
    Write-ColorOutput "`nTest 2: Version Increment" "Yellow"
    
    if (Test-Path "parse-version.ps1") {
        $testCases = @(
            @{Version="1.2.3"; Increment="patch"; Expected="1.2.4"},
            @{Version="1.2.3"; Increment="minor"; Expected="1.3.0"},
            @{Version="1.2.3"; Increment="major"; Expected="2.0.0"},
            @{Version="0.1.0"; Increment="patch"; Expected="0.1.1"}
        )
        
        foreach ($testCase in $testCases) {
            $version = $testCase.Version
            $increment = $testCase.Increment
            $expected = $testCase.Expected
            
            Write-Host "  Testing $version + $increment = $expected... " -NoNewline
            
            try {
                $output = & .\parse-version.ps1 -Version $version -Command increment -Parameter $increment 2>$null
                $actual = ($output | Where-Object { $_ -like "NEXT_VERSION=*" }) -replace "NEXT_VERSION=", ""
                
                if ($actual -eq $expected) {
                    Write-ColorOutput "✓" "Green"
                } else {
                    Write-ColorOutput "✗ (got $actual)" "Red"
                }
            } catch {
                Write-ColorOutput "✗ (script failed)" "Red"
                if ($Verbose) {
                    Write-Host "    Error: $($_.Exception.Message)"
                }
            }
        }
    } else {
        Write-ColorOutput "✗ parse-version.ps1 not found" "Red"
    }
}

function Test-VersionComparison {
    Write-ColorOutput "`nTest 3: Version Comparison" "Yellow"
    
    if (Test-Path "parse-version.ps1") {
        $testCases = @(
            @{Version1="1.2.3"; Version2="1.2.3"; Expected="equal"},
            @{Version1="1.2.4"; Version2="1.2.3"; Expected="greater"},
            @{Version1="1.2.3"; Version2="1.2.4"; Expected="less"},
            @{Version1="1.2.3"; Version2="1.2.3-alpha.1"; Expected="greater"},
            @{Version1="1.2.3-alpha.1"; Version2="1.2.3"; Expected="less"}
        )
        
        foreach ($testCase in $testCases) {
            $version1 = $testCase.Version1
            $version2 = $testCase.Version2
            $expected = $testCase.Expected
            
            Write-Host "  Testing $version1 vs $version2 = $expected... " -NoNewline
            
            try {
                $output = & .\parse-version.ps1 -Version $version1 -Command compare -Parameter $version2 2>$null
                $actual = ($output | Where-Object { $_ -like "COMPARISON=*" }) -replace "COMPARISON=", ""
                
                if ($actual -eq $expected) {
                    Write-ColorOutput "✓" "Green"
                } else {
                    Write-ColorOutput "✗ (got $actual)" "Red"
                }
            } catch {
                Write-ColorOutput "✗ (script failed)" "Red"
                if ($Verbose) {
                    Write-Host "    Error: $($_.Exception.Message)"
                }
            }
        }
    } else {
        Write-ColorOutput "✗ parse-version.ps1 not found" "Red"
    }
}

function Test-VersionUpdate {
    Write-ColorOutput "`nTest 4: Version Update Script" "Yellow"
    
    if (Test-Path "update-version.ps1") {
        # Create temporary test environment
        $tempDir = New-TemporaryFile | ForEach-Object { Remove-Item $_; New-Item -ItemType Directory -Path $_ }
        Write-Host "  Using temporary directory: $tempDir"
        
        try {
            # Create test files
            $null = New-Item -Path "$tempDir\cmd\af" -ItemType Directory -Force
            
            Set-Content -Path "$tempDir\go.mod" -Value @"
module github.com/agentflow/agentflow

go 1.22

// version: v0.0.1
"@
            
            Set-Content -Path "$tempDir\cmd\af\version.go" -Value @"
package main

const (
    Version = "0.0.1"
    BuildDate = ""
    GitCommit = ""
)
"@
            
            Set-Content -Path "$tempDir\Dockerfile" -Value @"
FROM golang:1.22-alpine
LABEL version="0.0.1"
COPY . .
"@
            
            # Test version update
            $oldLocation = Get-Location
            Set-Location $tempDir
            
            Write-Host "  Testing version update to 0.2.0... " -NoNewline
            
            try {
                & "$PSScriptRoot\update-version.ps1" -NewVersion "0.2.0" > $null 2>&1
                
                # Verify updates
                $goModContent = Get-Content "go.mod" -Raw
                $versionGoContent = Get-Content "cmd\af\version.go" -Raw
                $dockerfileContent = Get-Content "Dockerfile" -Raw
                
                if ($goModContent -match "// version: v0\.2\.0" -and
                    $versionGoContent -match 'Version = "0\.2\.0"' -and
                    $dockerfileContent -match 'LABEL version="0\.2\.0"') {
                    Write-ColorOutput "✓" "Green"
                } else {
                    Write-ColorOutput "✗ (files not updated correctly)" "Red"
                }
            } catch {
                Write-ColorOutput "✗ (script failed)" "Red"
                if ($Verbose) {
                    Write-Host "    Error: $($_.Exception.Message)"
                }
            } finally {
                Set-Location $oldLocation
            }
        } finally {
            # Cleanup
            Remove-Item -Path $tempDir -Recurse -Force -ErrorAction SilentlyContinue
        }
    } else {
        Write-ColorOutput "✗ update-version.ps1 not found" "Red"
    }
}

function Test-Documentation {
    Write-ColorOutput "`nTest 5: Documentation Validation" "Yellow"
    
    # Check RELEASE.md
    Write-Host "  Checking RELEASE.md... " -NoNewline
    if (Test-Path "..\RELEASE.md") {
        $releaseContent = Get-Content "..\RELEASE.md" -Raw
        $requiredSections = @(
            "Versioning Scheme",
            "Tagging Policy", 
            "Branching Model",
            "Release Process",
            "Hotfix Process"
        )
        
        $allFound = $true
        foreach ($section in $requiredSections) {
            if ($releaseContent -notmatch [regex]::Escape($section)) {
                $allFound = $false
                break
            }
        }
        
        if ($allFound) {
            Write-ColorOutput "✓" "Green"
        } else {
            Write-ColorOutput "✗ (missing required sections)" "Red"
        }
    } else {
        Write-ColorOutput "✗ (file not found)" "Red"
    }
    
    # Check CHANGELOG.md
    Write-Host "  Checking CHANGELOG.md... " -NoNewline
    if (Test-Path "..\CHANGELOG.md") {
        $changelogContent = Get-Content "..\CHANGELOG.md" -Raw
        $requiredSections = @(
            "# Changelog",
            "## [Unreleased]",
            "### Added",
            "### Changed", 
            "### Fixed",
            "### Security"
        )
        
        $allFound = $true
        foreach ($section in $requiredSections) {
            if ($changelogContent -notmatch [regex]::Escape($section)) {
                $allFound = $false
                break
            }
        }
        
        if ($allFound) {
            Write-ColorOutput "✓" "Green"
        } else {
            Write-ColorOutput "✗ (missing required sections)" "Red"
        }
    } else {
        Write-ColorOutput "✗ (file not found)" "Red"
    }
    
    # Check version.go
    Write-Host "  Checking cmd/af/version.go... " -NoNewline
    if (Test-Path "..\cmd\af\version.go") {
        $versionContent = Get-Content "..\cmd\af\version.go" -Raw
        $requiredFunctions = @(
            "func GetVersionInfo",
            "func GetVersionString",
            "func IsPreRelease",
            "func GetMajorVersion",
            "func IsStableAPI"
        )
        
        $allFound = $true
        foreach ($func in $requiredFunctions) {
            if ($versionContent -notmatch [regex]::Escape($func)) {
                $allFound = $false
                break
            }
        }
        
        if ($allFound) {
            Write-ColorOutput "✓" "Green"
        } else {
            Write-ColorOutput "✗ (missing required functions)" "Red"
        }
    } else {
        Write-ColorOutput "✗ (file not found)" "Red"
    }
}

# Main execution
Write-ColorOutput "Testing AgentFlow Version Management" "Blue"
Write-ColorOutput "========================================" "Blue"

$currentLocation = Get-Location
try {
    Set-Location $PSScriptRoot
    
    Test-VersionParsing
    Test-VersionIncrement  
    Test-VersionComparison
    Test-VersionUpdate
    Test-Documentation
    
} finally {
    Set-Location $currentLocation
}

Write-ColorOutput "`nVersion Management Tests Complete" "Blue"
Write-ColorOutput "========================================" "Blue"