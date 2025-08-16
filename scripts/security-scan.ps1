# Security Scanning Script (PowerShell)
# Runs comprehensive security scans with configurable severity thresholds
# Fails on High/Critical vulnerabilities by default

param(
    [string]$Threshold = "high",
    [string]$OutputDir = "",
    [string]$ConfigFile = "",
    [switch]$InstallTools,
    [switch]$SkipInstall,
    [switch]$Verbose,
    [switch]$Help
)

# Set error action preference
$ErrorActionPreference = "Stop"

# Get script and repository root directories
$ScriptDir = Split-Path -Parent $MyInvocation.MyCommand.Path
$RepoRoot = Split-Path -Parent $ScriptDir

# Default configuration
$DefaultOutputDir = Join-Path $RepoRoot "security-reports"
$DefaultConfigFile = Join-Path $RepoRoot ".security-config.yml"

# Configuration
if (-not $OutputDir) { $OutputDir = $DefaultOutputDir }
if (-not $ConfigFile) { $ConfigFile = $DefaultConfigFile }

# Create output directory
if (-not (Test-Path $OutputDir)) {
    New-Item -ItemType Directory -Path $OutputDir -Force | Out-Null
}

# Security scan results
$ScanResults = @()
$TotalVulnerabilities = 0
$HighCriticalVulnerabilities = 0
$ScanFailed = $false

# Logging functions
function Write-Info {
    param([string]$Message)
    Write-Host "[INFO] $Message" -ForegroundColor Green
}

function Write-Warn {
    param([string]$Message)
    Write-Host "[WARN] $Message" -ForegroundColor Yellow
}

function Write-Error-Custom {
    param([string]$Message)
    Write-Host "[ERROR] $Message" -ForegroundColor Red
}

function Write-Debug-Custom {
    param([string]$Message)
    if ($Verbose) {
        Write-Host "[DEBUG] $Message" -ForegroundColor Blue
    }
}

# Parse severity threshold to numeric value for comparison
function Get-SeverityNumeric {
    param([string]$Severity)
    
    switch ($Severity.ToLower()) {
        "critical" { return 4 }
        "high" { return 3 }
        "medium" { return 2 }
        "low" { return 1 }
        "info" { return 0 }
        default { return 2 } # Default to medium
    }
}

# Check if severity meets threshold
function Test-SeverityThreshold {
    param([string]$Severity)
    
    $ThresholdNumeric = Get-SeverityNumeric $Threshold
    $SeverityNumeric = Get-SeverityNumeric $Severity
    
    return $SeverityNumeric -ge $ThresholdNumeric
}

# Check if command exists
function Test-Command {
    param([string]$Command)
    
    try {
        Get-Command $Command -ErrorAction Stop | Out-Null
        return $true
    }
    catch {
        return $false
    }
}

# Install security tools if not available
function Install-SecurityTools {
    Write-Info "Checking security tool availability..."
    
    # Check gosec
    if (-not (Test-Command "gosec")) {
        Write-Warn "gosec not found, installing..."
        go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
    }
    
    # Check govulncheck
    if (-not (Test-Command "govulncheck")) {
        Write-Warn "govulncheck not found, installing..."
        go install golang.org/x/vuln/cmd/govulncheck@latest
    }
    
    # Check gitleaks
    if (-not (Test-Command "gitleaks")) {
        Write-Warn "gitleaks not found. Please install manually from: https://github.com/gitleaks/gitleaks#installing"
    }
    
    # Check syft
    if (-not (Test-Command "syft")) {
        Write-Warn "syft not found. Installing via PowerShell..."
        try {
            $SyftUrl = "https://github.com/anchore/syft/releases/latest/download/syft_windows_amd64.zip"
            $TempPath = Join-Path $env:TEMP "syft.zip"
            $ExtractPath = Join-Path $env:TEMP "syft"
            
            Invoke-WebRequest -Uri $SyftUrl -OutFile $TempPath
            Expand-Archive -Path $TempPath -DestinationPath $ExtractPath -Force
            
            $SyftExe = Join-Path $ExtractPath "syft.exe"
            $LocalBin = Join-Path $env:USERPROFILE "bin"
            if (-not (Test-Path $LocalBin)) {
                New-Item -ItemType Directory -Path $LocalBin -Force | Out-Null
            }
            Copy-Item $SyftExe (Join-Path $LocalBin "syft.exe") -Force
            
            # Add to PATH if not already there
            $CurrentPath = [Environment]::GetEnvironmentVariable("PATH", "User")
            if ($CurrentPath -notlike "*$LocalBin*") {
                [Environment]::SetEnvironmentVariable("PATH", "$CurrentPath;$LocalBin", "User")
                $env:PATH += ";$LocalBin"
            }
            
            Write-Info "syft installed to $LocalBin"
        }
        catch {
            Write-Warn "Failed to install syft automatically. Please install manually."
        }
    }
    
    # Check grype
    if (-not (Test-Command "grype")) {
        Write-Warn "grype not found. Installing via PowerShell..."
        try {
            $GrypeUrl = "https://github.com/anchore/grype/releases/latest/download/grype_windows_amd64.zip"
            $TempPath = Join-Path $env:TEMP "grype.zip"
            $ExtractPath = Join-Path $env:TEMP "grype"
            
            Invoke-WebRequest -Uri $GrypeUrl -OutFile $TempPath
            Expand-Archive -Path $TempPath -DestinationPath $ExtractPath -Force
            
            $GrypeExe = Join-Path $ExtractPath "grype.exe"
            $LocalBin = Join-Path $env:USERPROFILE "bin"
            if (-not (Test-Path $LocalBin)) {
                New-Item -ItemType Directory -Path $LocalBin -Force | Out-Null
            }
            Copy-Item $GrypeExe (Join-Path $LocalBin "grype.exe") -Force
            
            # Add to PATH if not already there
            $CurrentPath = [Environment]::GetEnvironmentVariable("PATH", "User")
            if ($CurrentPath -notlike "*$LocalBin*") {
                [Environment]::SetEnvironmentVariable("PATH", "$CurrentPath;$LocalBin", "User")
                $env:PATH += ";$LocalBin"
            }
            
            Write-Info "grype installed to $LocalBin"
        }
        catch {
            Write-Warn "Failed to install grype automatically. Please install manually."
        }
    }
}

# Run gosec security scanner
function Invoke-Gosec {
    Write-Info "Running gosec security scanner..."
    
    $OutputFile = Join-Path $OutputDir "gosec-report.json"
    $SarifFile = Join-Path $OutputDir "gosec-report.sarif"
    
    try {
        $Process = Start-Process -FilePath "gosec" -ArgumentList @("-fmt", "json", "-out", $OutputFile, "-fmt", "sarif", "-out", $SarifFile, "./...") -Wait -PassThru -NoNewWindow
        
        if ($Process.ExitCode -eq 0) {
            Write-Info "✓ gosec scan completed successfully"
            
            # Parse results
            if (Test-Path $OutputFile) {
                $Report = Get-Content $OutputFile | ConvertFrom-Json
                $Issues = if ($Report.Issues) { $Report.Issues.Count } else { 0 }
                $HighCritical = if ($Report.Issues) { ($Report.Issues | Where-Object { $_.severity -eq "HIGH" -or $_.severity -eq "CRITICAL" }).Count } else { 0 }
                
                $script:ScanResults += "gosec: $Issues total issues, $HighCritical high/critical"
                $script:TotalVulnerabilities += $Issues
                $script:HighCriticalVulnerabilities += $HighCritical
                
                if ($HighCritical -gt 0 -and (Test-SeverityThreshold "high")) {
                    Write-Error-Custom "gosec found $HighCritical high/critical security issues"
                    $script:ScanFailed = $true
                }
            }
        }
        else {
            Write-Error-Custom "gosec scan failed"
            $script:ScanResults += "gosec: FAILED"
            $script:ScanFailed = $true
        }
    }
    catch {
        Write-Error-Custom "gosec scan failed: $($_.Exception.Message)"
        $script:ScanResults += "gosec: FAILED"
        $script:ScanFailed = $true
    }
}

# Run govulncheck vulnerability scanner
function Invoke-Govulncheck {
    Write-Info "Running govulncheck vulnerability scanner..."
    
    $OutputFile = Join-Path $OutputDir "govulncheck-report.json"
    
    try {
        $Process = Start-Process -FilePath "govulncheck" -ArgumentList @("-json", "./...") -Wait -PassThru -NoNewWindow -RedirectStandardOutput $OutputFile
        
        if ($Process.ExitCode -eq 0) {
            Write-Info "✓ govulncheck scan completed successfully"
            
            # Parse results
            if (Test-Path $OutputFile) {
                $Content = Get-Content $OutputFile -Raw
                $Lines = $Content -split "`n" | Where-Object { $_.Trim() -ne "" }
                $Vulns = 0
                
                foreach ($Line in $Lines) {
                    try {
                        $Json = $Line | ConvertFrom-Json
                        if ($Json.finding) {
                            $Vulns++
                        }
                    }
                    catch {
                        # Skip invalid JSON lines
                    }
                }
                
                $script:ScanResults += "govulncheck: $Vulns vulnerabilities found"
                $script:TotalVulnerabilities += $Vulns
                
                if ($Vulns -gt 0 -and (Test-SeverityThreshold "high")) {
                    Write-Error-Custom "govulncheck found $Vulns vulnerabilities"
                    $script:HighCriticalVulnerabilities += $Vulns
                    $script:ScanFailed = $true
                }
            }
        }
        else {
            Write-Error-Custom "govulncheck scan failed"
            $script:ScanResults += "govulncheck: FAILED"
            $script:ScanFailed = $true
        }
    }
    catch {
        Write-Error-Custom "govulncheck scan failed: $($_.Exception.Message)"
        $script:ScanResults += "govulncheck: FAILED"
        $script:ScanFailed = $true
    }
}

# Run gitleaks secret scanner
function Invoke-Gitleaks {
    Write-Info "Running gitleaks secret scanner..."
    
    $OutputFile = Join-Path $OutputDir "gitleaks-report.json"
    
    if (-not (Test-Command "gitleaks")) {
        Write-Warn "gitleaks not available, skipping..."
        $script:ScanResults += "gitleaks: NOT_AVAILABLE"
        return
    }
    
    try {
        $Process = Start-Process -FilePath "gitleaks" -ArgumentList @("detect", "--source=$RepoRoot", "--report-format", "json", "--report-path", $OutputFile, "--verbose") -Wait -PassThru -NoNewWindow
        
        if ($Process.ExitCode -eq 0) {
            Write-Info "✓ gitleaks scan completed - no secrets found"
            $script:ScanResults += "gitleaks: no secrets detected"
        }
        elseif ($Process.ExitCode -eq 1) {
            # Exit code 1 means secrets were found
            $SecretsCount = 0
            if (Test-Path $OutputFile) {
                $Report = Get-Content $OutputFile | ConvertFrom-Json
                $SecretsCount = if ($Report -is [array]) { $Report.Count } else { 1 }
            }
            Write-Error-Custom "gitleaks found $SecretsCount secrets"
            $script:ScanResults += "gitleaks: $SecretsCount secrets found"
            $script:HighCriticalVulnerabilities += $SecretsCount
            $script:ScanFailed = $true
        }
        else {
            Write-Error-Custom "gitleaks scan failed with exit code $($Process.ExitCode)"
            $script:ScanResults += "gitleaks: FAILED"
            $script:ScanFailed = $true
        }
    }
    catch {
        Write-Error-Custom "gitleaks scan failed: $($_.Exception.Message)"
        $script:ScanResults += "gitleaks: FAILED"
        $script:ScanFailed = $true
    }
}

# Run syft SBOM generation
function Invoke-Syft {
    Write-Info "Running syft SBOM generation..."
    
    $SbomFile = Join-Path $OutputDir "sbom.spdx.json"
    $CycloneFile = Join-Path $OutputDir "sbom.cyclonedx.json"
    
    if (-not (Test-Command "syft")) {
        Write-Warn "syft not available, skipping..."
        $script:ScanResults += "syft: NOT_AVAILABLE"
        return
    }
    
    try {
        $Process1 = Start-Process -FilePath "syft" -ArgumentList @("packages", ".", "-o", "spdx-json=$SbomFile") -Wait -PassThru -NoNewWindow
        $Process2 = Start-Process -FilePath "syft" -ArgumentList @("packages", ".", "-o", "cyclonedx-json=$CycloneFile") -Wait -PassThru -NoNewWindow
        
        if ($Process1.ExitCode -eq 0 -and $Process2.ExitCode -eq 0) {
            Write-Info "✓ syft SBOM generation completed"
            
            # Count packages
            $PackageCount = "unknown"
            if (Test-Path $SbomFile) {
                try {
                    $Sbom = Get-Content $SbomFile | ConvertFrom-Json
                    $PackageCount = if ($Sbom.packages) { $Sbom.packages.Count } else { 0 }
                }
                catch {
                    $PackageCount = "unknown"
                }
            }
            $script:ScanResults += "syft: $PackageCount packages cataloged"
        }
        else {
            Write-Error-Custom "syft SBOM generation failed"
            $script:ScanResults += "syft: FAILED"
        }
    }
    catch {
        Write-Error-Custom "syft SBOM generation failed: $($_.Exception.Message)"
        $script:ScanResults += "syft: FAILED"
    }
}

# Run grype vulnerability scanner
function Invoke-Grype {
    Write-Info "Running grype vulnerability scanner..."
    
    $OutputFile = Join-Path $OutputDir "grype-report.json"
    $SarifFile = Join-Path $OutputDir "grype-report.sarif"
    
    if (-not (Test-Command "grype")) {
        Write-Warn "grype not available, skipping..."
        $script:ScanResults += "grype: NOT_AVAILABLE"
        return
    }
    
    try {
        $Process1 = Start-Process -FilePath "grype" -ArgumentList @(".", "-o", "json", "--file", $OutputFile) -Wait -PassThru -NoNewWindow
        $Process2 = Start-Process -FilePath "grype" -ArgumentList @(".", "-o", "sarif", "--file", $SarifFile) -Wait -PassThru -NoNewWindow
        
        if ($Process1.ExitCode -eq 0) {
            Write-Info "✓ grype scan completed successfully"
            
            # Parse results
            if (Test-Path $OutputFile) {
                try {
                    $Report = Get-Content $OutputFile | ConvertFrom-Json
                    $TotalVulns = if ($Report.matches) { $Report.matches.Count } else { 0 }
                    $HighCriticalVulns = if ($Report.matches) { ($Report.matches | Where-Object { $_.vulnerability.severity -eq "High" -or $_.vulnerability.severity -eq "Critical" }).Count } else { 0 }
                    
                    $script:ScanResults += "grype: $TotalVulns total vulnerabilities, $HighCriticalVulns high/critical"
                    $script:TotalVulnerabilities += $TotalVulns
                    $script:HighCriticalVulnerabilities += $HighCriticalVulns
                    
                    if ($HighCriticalVulns -gt 0 -and (Test-SeverityThreshold "high")) {
                        Write-Error-Custom "grype found $HighCriticalVulns high/critical vulnerabilities"
                        $script:ScanFailed = $true
                    }
                }
                catch {
                    Write-Warn "Failed to parse grype results"
                    $script:ScanResults += "grype: PARSE_ERROR"
                }
            }
        }
        else {
            Write-Warn "grype scan failed, continuing..."
            $script:ScanResults += "grype: FAILED"
        }
    }
    catch {
        Write-Warn "grype scan failed: $($_.Exception.Message)"
        $script:ScanResults += "grype: FAILED"
    }
}

# Generate summary report
function New-SummaryReport {
    $SummaryFile = Join-Path $OutputDir "security-summary.json"
    $Timestamp = (Get-Date).ToUniversalTime().ToString("yyyy-MM-ddTHH:mm:ssZ")
    
    $Summary = @{
        timestamp = $Timestamp
        severity_threshold = $Threshold
        scan_status = if ($ScanFailed) { "FAILED" } else { "PASSED" }
        total_vulnerabilities = $TotalVulnerabilities
        high_critical_vulnerabilities = $HighCriticalVulnerabilities
        scan_results = $ScanResults
        reports = @{
            gosec = Join-Path $OutputDir "gosec-report.json"
            govulncheck = Join-Path $OutputDir "govulncheck-report.json"
            gitleaks = Join-Path $OutputDir "gitleaks-report.json"
            grype = Join-Path $OutputDir "grype-report.json"
            sbom_spdx = Join-Path $OutputDir "sbom.spdx.json"
            sbom_cyclonedx = Join-Path $OutputDir "sbom.cyclonedx.json"
        }
    }
    
    $Summary | ConvertTo-Json -Depth 10 | Out-File -FilePath $SummaryFile -Encoding UTF8
    Write-Info "Security summary report generated: $SummaryFile"
}

# Print scan results
function Show-Results {
    Write-Host ""
    Write-Info "=== Security Scan Results ==="
    Write-Host ""
    
    foreach ($Result in $ScanResults) {
        Write-Host "  • $Result"
    }
    
    Write-Host ""
    Write-Info "Total vulnerabilities found: $TotalVulnerabilities"
    Write-Info "High/Critical vulnerabilities: $HighCriticalVulnerabilities"
    Write-Info "Severity threshold: $Threshold"
    
    if ($ScanFailed) {
        Write-Host ""
        Write-Error-Custom "❌ Security scan FAILED - vulnerabilities above threshold detected"
        Write-Error-Custom "Review reports in: $OutputDir"
        exit 1
    }
    else {
        Write-Host ""
        Write-Info "✅ Security scan PASSED - no vulnerabilities above threshold"
        exit 0
    }
}

# Show usage information
function Show-Usage {
    Write-Host @"
Usage: .\security-scan.ps1 [OPTIONS]

Security scanning script with configurable severity thresholds.

OPTIONS:
    -Threshold LEVEL         Set severity threshold (critical|high|medium|low) [default: high]
    -OutputDir DIR           Set output directory for reports [default: ./security-reports]
    -ConfigFile FILE         Use custom configuration file
    -InstallTools            Install missing security tools
    -SkipInstall             Skip automatic tool installation
    -Verbose                 Enable verbose output
    -Help                    Show this help message

EXAMPLES:
    .\security-scan.ps1                    # Run with default settings (fail on high/critical)
    .\security-scan.ps1 -Threshold critical   # Only fail on critical vulnerabilities
    .\security-scan.ps1 -OutputDir C:\temp\reports   # Use custom output directory
    .\security-scan.ps1 -InstallTools         # Install missing tools before scanning
    .\security-scan.ps1 -Verbose              # Enable verbose output

EXIT CODES:
    0    All scans passed or no vulnerabilities above threshold
    1    Vulnerabilities above threshold detected or scan failed
    2    Invalid arguments or configuration error
"@
}

# Validate arguments
function Test-Arguments {
    # Validate severity threshold
    $ValidThresholds = @("critical", "high", "medium", "low", "info")
    if ($Threshold.ToLower() -notin $ValidThresholds) {
        Write-Error-Custom "Invalid severity threshold: $Threshold"
        Write-Error-Custom "Valid options: $($ValidThresholds -join ', ')"
        exit 2
    }
}

# Main execution
function Main {
    if ($Help) {
        Show-Usage
        exit 0
    }
    
    Test-Arguments
    
    Write-Info "Starting security scan with threshold: $Threshold"
    Write-Info "Output directory: $OutputDir"
    
    Set-Location $RepoRoot
    
    # Install tools if requested or needed
    if ($InstallTools -or -not $SkipInstall) {
        Install-SecurityTools
    }
    
    # Run security scans
    Invoke-Gosec
    Invoke-Govulncheck
    Invoke-Gitleaks
    Invoke-Syft
    Invoke-Grype
    
    # Generate reports
    New-SummaryReport
    
    # Print results and exit
    Show-Results
}

# Execute main function
Main