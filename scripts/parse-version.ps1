# AgentFlow Version Parsing Script (PowerShell)
# Parses and validates version strings according to semantic versioning

param(
    [Parameter(Mandatory=$true)]
    [string]$Version,
    
    [Parameter(Mandatory=$false)]
    [ValidateSet("parse", "increment", "compare", "validate")]
    [string]$Command = "parse",
    
    [Parameter(Mandatory=$false)]
    [string]$Parameter
)

$ErrorActionPreference = "Stop"

function Write-ColorOutput {
    param(
        [string]$Message,
        [string]$Color = "White"
    )
    Write-Host $Message -ForegroundColor $Color
}

function Parse-Version {
    param([string]$VersionString)
    
    # Remove 'v' prefix if present
    $cleanVersion = $VersionString -replace '^v', ''
    
    # Validate version format
    if ($cleanVersion -notmatch '^([0-9]+)\.([0-9]+)\.([0-9]+)(-([a-zA-Z0-9.-]+))?$') {
        Write-ColorOutput "Error: Invalid version format: $VersionString" "Red"
        Write-ColorOutput "Expected format: MAJOR.MINOR.PATCH[-PRERELEASE]" "Yellow"
        Write-ColorOutput "Examples: 1.2.3, 0.1.0-alpha.1, 2.0.0-beta.2" "Yellow"
        exit 1
    }
    
    $major = [int]$Matches[1]
    $minor = [int]$Matches[2]
    $patch = [int]$Matches[3]
    $prerelease = $Matches[5]
    
    # Output parsed components
    Write-Output "MAJOR=$major"
    Write-Output "MINOR=$minor"
    Write-Output "PATCH=$patch"
    Write-Output "PRERELEASE=$prerelease"
    Write-Output "FULL_VERSION=$cleanVersion"
    Write-Output "TAG_VERSION=v$cleanVersion"
    
    # Determine version type
    if ($prerelease) {
        Write-Output "VERSION_TYPE=prerelease"
        if ($prerelease -like "alpha*") {
            Write-Output "PRERELEASE_TYPE=alpha"
        } elseif ($prerelease -like "beta*") {
            Write-Output "PRERELEASE_TYPE=beta"
        } elseif ($prerelease -like "rc*") {
            Write-Output "PRERELEASE_TYPE=rc"
        } else {
            Write-Output "PRERELEASE_TYPE=other"
        }
    } else {
        Write-Output "VERSION_TYPE=release"
        Write-Output "PRERELEASE_TYPE="
    }
    
    # Determine stability
    if ($major -eq 0) {
        Write-Output "STABILITY=unstable"
        Write-Output "API_COMPATIBILITY=none"
    } else {
        Write-Output "STABILITY=stable"
        Write-Output "API_COMPATIBILITY=semantic"
    }
}

function Increment-Version {
    param(
        [string]$VersionString,
        [string]$IncrementType
    )
    
    # Parse current version
    $cleanVersion = $VersionString -replace '^v', ''
    if ($cleanVersion -notmatch '^([0-9]+)\.([0-9]+)\.([0-9]+)(-([a-zA-Z0-9.-]+))?$') {
        Write-ColorOutput "Error: Invalid version format for increment: $VersionString" "Red"
        exit 1
    }
    
    $major = [int]$Matches[1]
    $minor = [int]$Matches[2]
    $patch = [int]$Matches[3]
    
    switch ($IncrementType.ToLower()) {
        "major" {
            $major++
            $minor = 0
            $patch = 0
        }
        "minor" {
            $minor++
            $patch = 0
        }
        "patch" {
            $patch++
        }
        default {
            Write-ColorOutput "Error: Invalid increment type: $IncrementType" "Red"
            Write-ColorOutput "Valid types: major, minor, patch" "Yellow"
            exit 1
        }
    }
    
    Write-Output "NEXT_VERSION=$major.$minor.$patch"
    Write-Output "NEXT_TAG_VERSION=v$major.$minor.$patch"
}

function Compare-Versions {
    param(
        [string]$Version1,
        [string]$Version2
    )
    
    # Remove 'v' prefix if present
    $cleanVersion1 = $Version1 -replace '^v', ''
    $cleanVersion2 = $Version2 -replace '^v', ''
    
    # Parse version1
    if ($cleanVersion1 -notmatch '^([0-9]+)\.([0-9]+)\.([0-9]+)(-([a-zA-Z0-9.-]+))?$') {
        Write-ColorOutput "Error: Invalid version1 format: $Version1" "Red"
        exit 1
    }
    $major1 = [int]$Matches[1]; $minor1 = [int]$Matches[2]; $patch1 = [int]$Matches[3]; $pre1 = $Matches[5]
    
    # Parse version2
    if ($cleanVersion2 -notmatch '^([0-9]+)\.([0-9]+)\.([0-9]+)(-([a-zA-Z0-9.-]+))?$') {
        Write-ColorOutput "Error: Invalid version2 format: $Version2" "Red"
        exit 1
    }
    $major2 = [int]$Matches[1]; $minor2 = [int]$Matches[2]; $patch2 = [int]$Matches[3]; $pre2 = $Matches[5]
    
    # Compare major.minor.patch
    if ($major1 -gt $major2) {
        Write-Output "COMPARISON=greater"
    } elseif ($major1 -lt $major2) {
        Write-Output "COMPARISON=less"
    } elseif ($minor1 -gt $minor2) {
        Write-Output "COMPARISON=greater"
    } elseif ($minor1 -lt $minor2) {
        Write-Output "COMPARISON=less"
    } elseif ($patch1 -gt $patch2) {
        Write-Output "COMPARISON=greater"
    } elseif ($patch1 -lt $patch2) {
        Write-Output "COMPARISON=less"
    } else {
        # Same major.minor.patch, compare prerelease
        if (-not $pre1 -and -not $pre2) {
            Write-Output "COMPARISON=equal"
        } elseif (-not $pre1 -and $pre2) {
            Write-Output "COMPARISON=greater"  # Release > prerelease
        } elseif ($pre1 -and -not $pre2) {
            Write-Output "COMPARISON=less"     # Prerelease < release
        } else {
            # Both have prerelease, lexicographic comparison
            if ($pre1 -eq $pre2) {
                Write-Output "COMPARISON=equal"
            } elseif ($pre1 -gt $pre2) {
                Write-Output "COMPARISON=greater"
            } else {
                Write-Output "COMPARISON=less"
            }
        }
    }
}

function Test-VersionSequence {
    param(
        [string]$CurrentVersion,
        [string]$NextVersion
    )
    
    # Get comparison result
    $comparison = Compare-Versions $CurrentVersion $NextVersion
    $comparisonResult = ($comparison | Where-Object { $_ -like "COMPARISON=*" }) -replace "COMPARISON=", ""
    
    if ($comparisonResult -eq "greater" -or $comparisonResult -eq "equal") {
        Write-Output "SEQUENCE_VALID=true"
        Write-Output "SEQUENCE_TYPE=forward"
    } else {
        Write-Output "SEQUENCE_VALID=false"
        Write-Output "SEQUENCE_TYPE=backward"
        Write-ColorOutput "Warning: Version sequence goes backward" "Yellow"
        Write-ColorOutput "Current: $CurrentVersion, Next: $NextVersion" "Yellow"
    }
}

# Main execution
switch ($Command.ToLower()) {
    "parse" {
        Parse-Version $Version
    }
    "increment" {
        if (-not $Parameter) {
            Write-ColorOutput "Error: Increment type required" "Red"
            Write-ColorOutput "Usage: .\parse-version.ps1 <version> -Command increment -Parameter <major|minor|patch>" "Yellow"
            exit 1
        }
        Increment-Version $Version $Parameter
    }
    "compare" {
        if (-not $Parameter) {
            Write-ColorOutput "Error: Second version required for comparison" "Red"
            Write-ColorOutput "Usage: .\parse-version.ps1 <version1> -Command compare -Parameter <version2>" "Yellow"
            exit 1
        }
        Compare-Versions $Version $Parameter
    }
    "validate" {
        if (-not $Parameter) {
            Write-ColorOutput "Error: Next version required for validation" "Red"
            Write-ColorOutput "Usage: .\parse-version.ps1 <current_version> -Command validate -Parameter <next_version>" "Yellow"
            exit 1
        }
        Test-VersionSequence $Version $Parameter
    }
    default {
        Write-ColorOutput "Error: Unknown command: $Command" "Red"
        Write-ColorOutput "Available commands: parse, increment, compare, validate" "Yellow"
        exit 1
    }
}