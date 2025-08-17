# Validation script for pkg/messaging
# Runs unit tests with coverage and prints summary.

param(
    [string]$Package = './pkg/messaging'
)

Write-Host "Running tests for $Package"
$coverFile = "coverage-messaging.out"

$testCmd = "go test $Package -coverprofile=$coverFile -v"
Write-Host "Executing: $testCmd"

# Run go test directly so the cover profile is written to the expected path
try {
    & go test $Package -coverprofile=$coverFile -v
    $exitCode = $LASTEXITCODE
} catch {
    Write-Host "go test invocation failed: $_" -ForegroundColor Red
    exit 1
}

if ($exitCode -ne 0) {
    Write-Host "Tests failed with exit code $exitCode" -ForegroundColor Red
    exit $exitCode
}

# Print coverage summary
if (Test-Path $coverFile) {
    $coverageLine = go tool cover -func=$coverFile | Select-String "total:" -SimpleMatch
    Write-Host "Coverage: $coverageLine"
} else {
    Write-Host "Coverage file not found: $coverFile" -ForegroundColor Yellow
}

Write-Host "Messaging validation completed successfully." -ForegroundColor Green
exit 0
