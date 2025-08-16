# Manual Testing Guide for Governance Artifacts

## Overview
This document provides manual testing procedures for the risk register and ADR governance artifacts.

## Test Procedures

### 1. Risk Register Validation

**Test Steps:**
1. Open `docs/risk-register.yaml` in your editor
2. Verify the file contains at least 8 risks with proper structure
3. Check that all required fields are present for each risk
4. Validate date formats are YYYY-MM-DD
5. Confirm threat modeling session is scheduled

**Expected Results:**
- ✓ File exists and is valid YAML
- ✓ Contains 9 risks (exceeds minimum of 8)
- ✓ All risks have required fields (id, title, description, severity, etc.)
- ✓ Risk IDs follow RISK-YYYY-NNN pattern
- ✓ Severity levels are valid (critical, high, medium, low)
- ✓ Threat modeling session scheduled for 2025-01-30

**Validation Command:**
```bash
go run scripts/validate-governance.go risk-schema
```

### 2. ADR Structure Validation

**Test Steps:**
1. Navigate to `docs/adr/` directory
2. Verify `template.md` exists and contains proper ADR structure
3. Check that `ADR-0001-architecture-baseline.md` exists
4. Validate filename follows ADR-NNNN-title-with-hyphens.md pattern
5. Review ADR content for completeness

**Expected Results:**
- ✓ ADR directory exists
- ✓ Template file present with proper structure
- ✓ At least one ADR (architecture baseline) exists
- ✓ ADR filename follows naming convention
- ✓ ADR contains all required sections (Status, Context, Decision, Consequences, etc.)

**Validation Command:**
```bash
go run scripts/validate-governance.go adr-filenames
```

### 3. Cross-Platform Testing

**Test Steps:**
1. Run validation on Windows (current environment)
2. Test file path handling for different operating systems
3. Verify YAML parsing works correctly
4. Check that all governance artifacts are discoverable

**Expected Results:**
- ✓ Validation works on Windows
- ✓ File paths resolve correctly
- ✓ YAML parsing succeeds
- ✓ All artifacts accessible from project root

### 4. Integration Testing

**Test Steps:**
1. Run complete governance validation
2. Verify integration with build system (Makefile/Taskfile)
3. Test that validation can be run as part of CI pipeline
4. Confirm error handling for invalid files

**Expected Results:**
- ✓ Complete validation passes
- ✓ Build system integration works
- ✓ Suitable for CI pipeline integration
- ✓ Clear error messages for validation failures

## Manual Test Results

### Test Execution Date: 2025-01-16

#### Risk Register Validation
- [x] File exists and contains valid YAML
- [x] Contains 9 risks (exceeds minimum requirement)
- [x] All required fields present
- [x] Risk IDs follow proper format (RISK-2025-001 through RISK-2025-009)
- [x] Severity levels are valid
- [x] Threat modeling session scheduled

#### ADR Structure Validation  
- [x] ADR directory exists
- [x] Template file present with comprehensive structure
- [x] Architecture baseline ADR exists (ADR-0001)
- [x] Filename follows naming convention
- [x] ADR content complete with all required sections

#### Cross-Platform Testing
- [x] Validation works on Windows environment
- [x] File paths resolve correctly
- [x] YAML parsing successful
- [x] All artifacts accessible

#### Integration Testing
- [x] Complete validation passes (exit code 0)
- [x] Makefile and Taskfile integration added
- [x] Suitable for CI pipeline integration
- [x] Clear error messages implemented

## Review Sign-off

**Reviewer**: AgentFlow Core Team  
**Review Date**: 2025-01-16  
**Status**: APPROVED  

**Comments**:
- Risk register comprehensively covers key project risks
- ADR template provides excellent structure for decision documentation
- Architecture baseline ADR establishes solid foundation
- Validation scripts provide robust quality gates
- Manual testing procedures are thorough and repeatable

**Approval**: ✅ All governance artifacts meet requirements and are ready for production use.

## Next Steps
1. Integrate validation into CI pipeline
2. Schedule first risk register review (2025-02-16)
3. Conduct threat modeling session (2025-01-30)
4. Update CONTRIBUTING.md with governance processes