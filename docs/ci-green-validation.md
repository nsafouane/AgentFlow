# CI Green Validation Procedures

This document describes the procedures for validating that all CI workflows pass with no High/Critical vulnerabilities, as part of the Foundations & Project Governance specification.

## Overview

The CI Green validation ensures that:
- All GitHub Actions workflows complete successfully
- Security scans pass with no High/Critical vulnerabilities
- Security tools are properly configured with appropriate thresholds
- SARIF reports are uploaded to GitHub Security tab
- Local security validation can reproduce CI results

## Validation Components

### 1. Workflow Status Validation

The validation checks the status of key workflows:

- **CI Pipeline** (`ci.yml`): Build, test, lint, and basic security scans
- **Security Scan** (`security-scan.yml`): Comprehensive security scanning
- **Container Build** (`container-build.yml`): Multi-arch container builds with signing

#### Success Criteria
- Latest workflow runs must have `conclusion: success`
- No workflows should be consistently failing
- In-progress workflows are acceptable but flagged as warnings

### 2. Security Threshold Validation

Validates that security tools are configured with appropriate severity thresholds:

#### Required Thresholds
- **gosec**: Fail on HIGH or CRITICAL severity issues
- **grype**: Fail on HIGH or CRITICAL vulnerabilities (`--fail-on high`)
- **trivy**: Fail on HIGH or CRITICAL vulnerabilities
- **gitleaks**: Fail on any secrets detected
- **govulncheck**: Fail on any vulnerabilities found

#### Configuration Locations
- Workflow files: `.github/workflows/*.yml`
- Security config: `.security-config.yml`
- Tool-specific configs: `.gosec.json`, `.gitleaks.toml`

### 3. Security Tools Validation

Ensures all required security tools are present in workflows:

#### Required Tools
1. **gosec** - Go security analyzer
2. **gitleaks** - Secret detection
3. **grype** - Container vulnerability scanning
4. **syft** - SBOM generation
5. **govulncheck** - Go vulnerability database

#### Validation Method
- Searches workflow files for tool usage
- Verifies tools are called with appropriate parameters
- Checks for proper error handling and reporting

### 4. SARIF Upload Validation

Validates that security scan results are uploaded to GitHub Security tab:

#### Requirements
- Use `github/codeql-action/upload-sarif@v3` action
- Generate SARIF format outputs from security tools
- Upload results with appropriate categories
- Handle upload failures gracefully

### 5. Local Security Validation

Runs local security scans to validate CI configuration:

#### Process
1. Execute `scripts/security-scan.sh` with high threshold
2. Compare results with CI workflow expectations
3. Validate that local scans would fail on High/Critical issues
4. Generate local security report for comparison

## Usage

### Running Validation

#### Basic Usage
```bash
# Run full validation
./scripts/validate-ci-green.sh

# Check specific branch
./scripts/validate-ci-green.sh --branch develop

# Skip local security scan (faster)
./scripts/validate-ci-green.sh --skip-local-scan

# Skip GitHub workflow checks (offline mode)
./scripts/validate-ci-green.sh --skip-github-check
```

#### Advanced Options
```bash
# Custom security threshold
./scripts/validate-ci-green.sh --threshold critical

# Custom output directory
./scripts/validate-ci-green.sh --output /tmp/ci-reports

# Verbose output
./scripts/validate-ci-green.sh --verbose

# Help
./scripts/validate-ci-green.sh --help
```

### Prerequisites

#### GitHub CLI (for workflow status checks)
```bash
# Install GitHub CLI
# macOS
brew install gh

# Ubuntu/Debian
sudo apt install gh

# Windows
winget install GitHub.cli

# Authenticate
gh auth login
```

#### Security Tools (for local validation)
```bash
# Install required tools
go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
go install golang.org/x/vuln/cmd/govulncheck@latest

# Install gitleaks
brew install gitleaks  # macOS
# or download from https://github.com/gitleaks/gitleaks/releases

# Install syft and grype
curl -sSfL https://raw.githubusercontent.com/anchore/syft/main/install.sh | sh -s -- -b /usr/local/bin
curl -sSfL https://raw.githubusercontent.com/anchore/grype/main/install.sh | sh -s -- -b /usr/local/bin
```

## Validation Report

The validation generates a comprehensive JSON report:

```json
{
  "timestamp": "2024-01-01T00:00:00Z",
  "validation_status": "PASSED",
  "security_threshold": "high",
  "total_validations": 8,
  "validations": [
    "Security Thresholds: PASSED - High/Critical severity thresholds configured",
    "Security Tools: PASSED - 5/5 required security tools found",
    "SARIF Upload: PASSED - SARIF upload configuration found",
    "CI Pipeline: PASSED - Latest run completed successfully",
    "Security Scan: PASSED - Latest run completed successfully",
    "Container Build: PASSED - Latest run completed successfully",
    "Local Security Scan: PASSED - No high/critical vulnerabilities found"
  ],
  "reports_directory": "./ci-validation-reports",
  "github_repository": "https://github.com/org/repo",
  "git_commit": "abc123def456",
  "git_branch": "main"
}
```

### Report Fields

- **timestamp**: ISO 8601 timestamp of validation
- **validation_status**: Overall status (PASSED/FAILED)
- **security_threshold**: Security severity threshold used
- **total_validations**: Number of validation checks performed
- **validations**: Array of individual validation results
- **reports_directory**: Directory containing detailed reports
- **github_repository**: Repository URL
- **git_commit**: Git commit hash
- **git_branch**: Git branch name

## Troubleshooting

### Common Issues

#### 1. Workflow Failures
```bash
# Check workflow logs
gh run list --workflow=ci.yml --limit=5
gh run view <run-id> --log

# Common causes:
# - Dependency vulnerabilities
# - New security issues in code
# - Tool configuration changes
# - Infrastructure issues
```

#### 2. Security Threshold Mismatches
```bash
# Verify threshold configuration
grep -r "fail-on\|severity" .github/workflows/
cat .security-config.yml

# Update thresholds if needed
# Edit workflow files or security config
```

#### 3. Missing Security Tools
```bash
# Check tool availability
which gosec gitleaks grype syft govulncheck

# Install missing tools
./scripts/validate-security-tools.sh
```

#### 4. SARIF Upload Failures
```bash
# Check SARIF file generation
ls -la security-reports/*.sarif

# Validate SARIF format
# Use online SARIF validator or GitHub's sarif-sdk
```

#### 5. Local vs CI Discrepancies
```bash
# Run local security scan
./scripts/security-scan.sh --threshold high

# Compare with CI results
# Check for environment differences
# Verify tool versions match CI
```

### Debugging Steps

1. **Enable Verbose Output**
   ```bash
   ./scripts/validate-ci-green.sh --verbose
   ```

2. **Check Individual Components**
   ```bash
   # Test security tools
   ./scripts/validate-security-tools.sh
   
   # Run security scan locally
   ./scripts/security-scan.sh
   
   # Check workflow syntax
   gh workflow list
   ```

3. **Review Generated Reports**
   ```bash
   # Check validation report
   cat ci-validation-reports/ci-green-validation-report.json
   
   # Review security scan results
   ls -la ci-validation-reports/local-security/
   ```

4. **Verify GitHub Integration**
   ```bash
   # Test GitHub CLI
   gh auth status
   gh run list --limit=5
   
   # Check repository permissions
   gh api repos/:owner/:repo --jq '.permissions'
   ```

## Integration with CI/CD

### Pre-commit Hook
```bash
# Add to .pre-commit-config.yaml
repos:
  - repo: local
    hooks:
      - id: ci-green-validation
        name: CI Green Validation
        entry: ./scripts/validate-ci-green.sh --skip-github-check
        language: system
        pass_filenames: false
```

### GitHub Actions Integration
```yaml
# Add to workflow
- name: Validate CI Green Status
  run: |
    ./scripts/validate-ci-green.sh --skip-local-scan
  env:
    GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
```

### Manual Testing Checklist

Before marking CI Green validation as complete:

- [ ] All workflows pass on main branch
- [ ] Security scans detect no High/Critical vulnerabilities
- [ ] Local security validation reproduces CI results
- [ ] SARIF reports are uploaded to GitHub Security tab
- [ ] Validation script runs without errors
- [ ] Unit tests pass for validation logic
- [ ] Documentation is complete and accurate

## Maintenance

### Regular Tasks

1. **Weekly**: Review validation reports for trends
2. **Monthly**: Update security tool versions
3. **Quarterly**: Review and update security thresholds
4. **As needed**: Update validation logic for new tools

### Version Updates

When updating security tools:

1. Update tool versions in workflows
2. Test validation with new versions
3. Update documentation if behavior changes
4. Verify backward compatibility

### Threshold Adjustments

When adjusting security thresholds:

1. Document rationale for changes
2. Update all relevant configuration files
3. Test with current codebase
4. Communicate changes to team

## References

- [GitHub Actions Security Hardening](https://docs.github.com/en/actions/security-guides/security-hardening-for-github-actions)
- [SARIF Format Specification](https://docs.oasis-open.org/sarif/sarif/v2.1.0/sarif-v2.1.0.html)
- [Security Scanning Tools Documentation](../security-baseline.md)
- [CI/CD Pipeline Documentation](../ci-policy.md)