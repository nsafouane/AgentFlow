# Gate G0 Validation Procedures

## Overview

This document provides comprehensive procedures for validating all Gate G0 exit criteria for the AgentFlow Foundations & Project Governance specification. Gate G0 represents the foundational milestone that must be achieved before proceeding to Q1.2 development.

**Validation Date**: 2025-01-16  
**Spec Version**: Q1.1 Foundations & Project Governance  
**Validation Scripts**: `scripts/validate-gate-g0.sh` (Linux/macOS), `scripts/validate-gate-g0.ps1` (Windows)

## Gate G0 Exit Criteria

### G0.1: CI Green Including Security Scans
**Requirement**: All CI workflows must pass with no High/Critical vulnerabilities

**Validation Procedure**:
1. **Automated Check**: Run `scripts/validate-gate-g0.sh` or `scripts/validate-gate-g0.ps1`
2. **Manual Verification**:
   - Navigate to GitHub Actions tab
   - Verify latest CI run shows all green checkmarks
   - Check security scan results for High/Critical findings
   - Confirm all required security tools are running (gosec, gitleaks, osv-scanner, grype)

**Expected Artifacts**:
- ✅ `.github/workflows/ci.yml` with security scans
- ✅ `.github/workflows/security-scan.yml` with severity thresholds
- ✅ All security tools integrated and passing

**Troubleshooting**:
- If security scans fail, review SARIF reports in Security tab
- Check security tool configurations for proper thresholds
- Verify no hardcoded secrets in codebase

### G0.2: Cross-Platform Builds
**Requirement**: Linux + Windows + WSL2 builds must succeed

**Validation Procedure**:
1. **Automated Check**: Validation script checks for cross-platform configuration
2. **Manual Verification**:
   - Run `make build` on Linux
   - Run `task build` on Windows
   - Run `make build` on WSL2
   - Verify CI matrix builds for multiple platforms

**Expected Artifacts**:
- ✅ `Makefile` with cross-platform targets
- ✅ `Taskfile.yml` with Windows compatibility
- ✅ CI workflow with platform matrix
- ✅ Cross-platform troubleshooting documentation

**Troubleshooting**:
- Check path separators for Windows compatibility
- Verify Go cross-compilation settings (GOOS/GOARCH)
- Review cross-platform build troubleshooting guide

### G0.3: Devcontainer Adoption
**Requirement**: `af validate` must warn when run outside container

**Validation Procedure**:
1. **Automated Check**: Validation script verifies devcontainer configuration
2. **Manual Verification**:
   - Open project in VS Code devcontainer
   - Run `af validate` inside container (should pass)
   - Run `af validate` on host system (should show warning)
   - Verify all required tools are available in container

**Expected Artifacts**:
- ✅ `.devcontainer/devcontainer.json` configuration
- ✅ CLI tool with container detection
- ✅ Devcontainer adoption guide documentation

**Troubleshooting**:
- Check devcontainer.json syntax and tool versions
- Verify CLI container detection logic
- Review devcontainer adoption guide for setup issues

### G0.4: SBOM & Provenance
**Requirement**: Artifacts must include SBOM and provenance per build

**Validation Procedure**:
1. **Automated Check**: Validation script checks CI configuration
2. **Manual Verification**:
   - Trigger CI build and check artifacts
   - Verify SBOM files are generated (SPDX, CycloneDX formats)
   - Confirm provenance attestation is created
   - Check container registry for signed artifacts

**Expected Artifacts**:
- ✅ SBOM generation in CI workflows
- ✅ Provenance attestation configuration
- ✅ SBOM verification documentation

**Troubleshooting**:
- Verify Syft tool installation and configuration
- Check GitHub Actions attestation permissions
- Review SBOM generation logs for errors

### G0.5: Signed Multi-Arch Images
**Requirement**: amd64+arm64 images must be signed and cosign verify must pass

**Validation Procedure**:
1. **Automated Check**: Validation script checks container build configuration
2. **Manual Verification**:
   ```bash
   # Pull and verify signed image
   docker pull ghcr.io/agentflow/agentflow/control-plane:latest
   cosign verify --certificate-identity-regexp="https://github.com/agentflow/agentflow" \
     --certificate-oidc-issuer="https://token.actions.githubusercontent.com" \
     ghcr.io/agentflow/agentflow/control-plane:latest
   ```
3. **Architecture Verification**:
   ```bash
   # Check multi-arch manifest
   docker manifest inspect ghcr.io/agentflow/agentflow/control-plane:latest
   ```

**Expected Artifacts**:
- ✅ Multi-architecture container builds (amd64, arm64)
- ✅ Cosign keyless signing configuration
- ✅ Signature verification in CI
- ✅ Supply chain security documentation

**Troubleshooting**:
- Check Docker Buildx configuration for multi-arch
- Verify Cosign installation and keyless signing setup
- Review container build logs for signing errors

### G0.6: Risk Register & ADR Baseline
**Requirement**: Risk register with ≥8 risks and ADR baseline must be merged

**Validation Procedure**:
1. **Automated Check**: Validation script counts risks and validates structure
2. **Manual Verification**:
   - Review `docs/risk-register.yaml` for completeness
   - Verify ≥8 risks with proper severity levels
   - Check `docs/adr/ADR-0001-architecture-baseline.md` exists
   - Confirm ADR template is available
   - Verify CONTRIBUTING.md references ADR process

**Expected Artifacts**:
- ✅ `docs/risk-register.yaml` with ≥8 risks
- ✅ `docs/adr/ADR-0001-architecture-baseline.md`
- ✅ `docs/adr/template.md`
- ✅ CONTRIBUTING.md with ADR references

**Troubleshooting**:
- Validate YAML syntax in risk register
- Ensure all risks have required fields (id, severity, mitigation)
- Check ADR structure follows template format

### G0.7: Release Versioning Policy
**Requirement**: RELEASE.md must be published and referenced by CI

**Validation Procedure**:
1. **Automated Check**: Validation script checks for required sections
2. **Manual Verification**:
   - Review `RELEASE.md` for completeness
   - Verify semantic versioning policy is documented
   - Check release workflow references versioning
   - Confirm version management scripts exist

**Expected Artifacts**:
- ✅ `RELEASE.md` with versioning scheme
- ✅ `.github/workflows/release.yml` with version logic
- ✅ Version management scripts
- ✅ Semantic versioning compliance

**Troubleshooting**:
- Check RELEASE.md contains all required sections
- Verify release workflow syntax and logic
- Test version management scripts

### G0.8: Interface Freeze Snapshot
**Requirement**: /docs/interfaces must be committed and referenced

**Validation Procedure**:
1. **Automated Check**: Validation script checks interface documentation
2. **Manual Verification**:
   - Review `docs/interfaces/README.md` for completeness
   - Verify all core interfaces are documented
   - Check interface freeze date is recorded
   - Confirm interfaces are referenced in main documentation

**Expected Artifacts**:
- ✅ `docs/interfaces/README.md` with core interfaces
- ✅ Interface freeze date documented
- ✅ References in README.md and ARCHITECTURE.md
- ✅ Interface stability guarantees defined

**Troubleshooting**:
- Ensure all core interface categories are covered
- Verify interface documentation is up-to-date
- Check cross-references in main documentation

### G0.9: Threat Model Kickoff Scheduled
**Requirement**: Threat modeling session must be logged in risk register

**Validation Procedure**:
1. **Automated Check**: Validation script checks threat modeling section
2. **Manual Verification**:
   - Review threat modeling section in risk register
   - Verify session date, owner, and participants are specified
   - Confirm scope and deliverables are defined
   - Check session is scheduled within reasonable timeframe

**Expected Artifacts**:
- ✅ Threat modeling section in risk register
- ✅ Session date and owner specified
- ✅ Participants and scope defined
- ✅ Deliverables outlined

**Troubleshooting**:
- Ensure threat modeling section has all required fields
- Verify session date is properly formatted
- Check participant availability and expertise

## Validation Execution

### Automated Validation

**Linux/macOS**:
```bash
# Make script executable
chmod +x scripts/validate-gate-g0.sh

# Run validation
./scripts/validate-gate-g0.sh
```

**Windows PowerShell**:
```powershell
# Run validation
.\scripts\validate-gate-g0.ps1

# Run with verbose output
.\scripts\validate-gate-g0.ps1 -Verbose
```

### Validation Results Summary

**Last Validation**: 2025-08-16  
**Status**: ✅ ALL GATE G0 CRITERIA PASSED  
**Foundation Status**: Ready for Q1.2 development

#### Detailed Results:
- ✅ **G0.1**: CI green including security scans - PASSED
- ✅ **G0.2**: Cross-platform builds (Linux + Windows + WSL2) - PASSED  
- ✅ **G0.3**: Devcontainer adoption (af validate warns outside container) - PASSED
- ✅ **G0.4**: SBOM & provenance (artifacts published per build) - PASSED
- ✅ **G0.5**: Signed multi-arch images (amd64+arm64, cosign verify passes) - PASSED
- ✅ **G0.6**: Risk register & ADR baseline (merged) - PASSED
- ✅ **G0.7**: Release versioning policy (RELEASE.md published & CI referenced) - PASSED
- ✅ **G0.8**: Interface freeze snapshot (/docs/interfaces committed & referenced) - PASSED
- ✅ **G0.9**: Threat model kickoff scheduled (logged in risk register) - PASSED

#### Validation Artifacts Generated:
- Comprehensive validation test suite (`scripts/validate-gate-g0_test.go`)
- Cross-platform validation scripts (PowerShell and Bash)
- Automated CI integration for continuous validation
- Complete documentation of validation procedures

### Manual Validation Checklist

Use this checklist for comprehensive manual validation:

- [ ] **G0.1**: All CI workflows green, security scans passing
- [ ] **G0.2**: Cross-platform builds successful on Linux, Windows, WSL2
- [ ] **G0.3**: Devcontainer working, CLI warns outside container
- [ ] **G0.4**: SBOM and provenance artifacts generated per build
- [ ] **G0.5**: Multi-arch images signed and verifiable with cosign
- [ ] **G0.6**: Risk register complete, ADR baseline merged
- [ ] **G0.7**: RELEASE.md published, CI references versioning
- [ ] **G0.8**: Interface documentation committed and referenced
- [ ] **G0.9**: Threat modeling session scheduled in risk register

### Validation Results

**Success Criteria**:
- All 9 Gate G0 criteria must pass
- Warnings are acceptable but should be addressed
- Validation scripts exit with code 0

**Failure Response**:
- Address all errors before proceeding to Q1.2
- Update documentation and configurations as needed
- Re-run validation until all criteria pass

## Continuous Validation

### CI Integration

Add Gate G0 validation to CI pipeline:

```yaml
# .github/workflows/gate-validation.yml
name: Gate Validation

on:
  push:
    branches: [ main ]
  schedule:
    - cron: '0 6 * * *'  # Daily at 6 AM UTC

jobs:
  validate-gate-g0:
    runs-on: ubuntu-latest
    steps:
    - uses: actions/checkout@v4
    - name: Validate Gate G0
      run: ./scripts/validate-gate-g0.sh
```

### Monitoring

- **Daily Validation**: Automated validation runs daily
- **PR Validation**: Gate validation on pull requests to main
- **Release Validation**: Mandatory validation before releases
- **Drift Detection**: Alert on validation failures

## Troubleshooting Guide

### Common Issues

1. **Security Scan Failures**:
   - Review vulnerability reports in GitHub Security tab
   - Update dependencies to resolve known vulnerabilities
   - Configure security tool thresholds appropriately

2. **Cross-Platform Build Issues**:
   - Check path separators and file permissions
   - Verify Go cross-compilation environment variables
   - Review platform-specific build configurations

3. **Container Signing Issues**:
   - Verify GitHub Actions permissions for attestations
   - Check Cosign installation and configuration
   - Review keyless signing setup

4. **Documentation Issues**:
   - Validate YAML syntax in configuration files
   - Check markdown formatting and links
   - Ensure all required sections are present

### Support Resources

- **Documentation**: Review all docs/ files for detailed guidance
- **Scripts**: Use validation scripts for automated checking
- **CI Logs**: Check GitHub Actions logs for detailed error information
- **Community**: Consult AgentFlow community for assistance

## Maintenance

### Regular Updates

- **Monthly**: Review and update risk register
- **Quarterly**: Update interface documentation
- **Per Release**: Validate all criteria before release
- **Annual**: Review and update validation procedures

### Version Control

- All validation artifacts are version controlled
- Changes require pull request review
- Validation scripts are tested before deployment
- Documentation is kept synchronized with implementation

---

**Document Version**: 1.0  
**Last Updated**: 2025-01-16  
**Next Review**: 2025-02-16  
**Maintained By**: AgentFlow Core Team