# Implementation Plan

## Overview

This implementation plan converts the Foundations & Project Governance design into a series of discrete, manageable coding tasks that exactly match the 10 tasks specified in Spec Q1.1. Each task follows the Implementation/Unit/Manual/Documentation structure and builds incrementally on previous tasks, ensuring early testing and validation of core functionality.

## Tasks

- [x] 1. Repository & Module Layout (control plane, worker, cli, sdk stubs, dashboard stub)





  - **Implementation**: Create Go modules for control plane, worker, cli, sdk stubs, and dashboard stub with shared internal packages layout and root Makefile/Taskfile
  - **Unit Tests**: Implement lint config test (golangci-lint) and placeholder unit test runs for each module
  - **Manual Testing**: Run build on Linux + Windows + WSL2 and verify task runner parity
  - **Documentation**: Create architecture README section describing repo conventions
  - _Requirements: 1.1, 1.2, 1.3, 1.4_

- [x] 2. Dev Container & Toolchain Standardization





  - **Implementation**: Create .devcontainer with pinned Go, NATS, Postgres clients, and pre-commit hooks
  - **Unit Tests**: Write script validating required binaries versions
  - **Manual Testing**: Open in VS Code devcontainer and run `af validate` stub
  - **Documentation**: Create dev environment guide with Windows fallback notes
  - _Requirements: 2.1, 2.2, 2.3, 2.4_

- [x] 3. CI Pipeline (Build, Lint, Test, SBOM, SAST, Dependencies, Secrets, License, Container Scan)






  - **Implementation**: Create GitHub Actions workflows with cache strategy and provenance attestation
  - **Unit Tests**: Implement workflow dry-run using act/minimal branch test and config schema lint
  - **Manual Testing**: Force failing job for dependency vulnerability and confirm block
  - **Documentation**: Create CI policy & gating doc
  - _Requirements: 3.1, 3.2, 3.3, 3.4_

- [x] 4. Security Tooling Integration (gosec, osv-scanner, gitleaks, syft/grype)








  - **Implementation**: Create scripts and severity thresholds (fail High/Critical)
  - **Unit Tests**: Parse mock reports and test threshold logic
  - **Manual Testing**: Introduce benign vulnerable lib in branch and ensure failure
  - **Documentation**: Create security baseline & exception process
  - _Requirements: 4.1, 4.2, 4.3, 4.4_

- [x] 5. Migration Tooling Decision & Policy (goose + sqlc)





  - **Implementation**: Pin versions, add initial empty migration, and configure sqlc
  - **Unit Tests**: Implement migration linter test and verify sqlc code compiles
  - **Manual Testing**: Run up/down locally with Windows path validation
  - **Documentation**: Create migration policy (naming, reversibility stance)
  - _Requirements: 5.1, 5.2, 5.3, 5.4_

- [x] 6. CLI `af validate` Stub





  - **Implementation**: Create CLI that outputs JSON skeleton with environment probes placeholders
  - **Unit Tests**: Implement JSON schema validation test
  - **Manual Testing**: Run on host vs devcontainer and verify warning displayed
  - **Documentation**: Create CLI usage quickstart
  - _Requirements: 6.1, 6.2, 6.3, 6.4_

- [x] 7. Versioning & Release Engineering Baseline






  - **Implementation**: Define semantic version scheme (pre-1.0 minor for breaking changes), tagging policy, and CHANGELOG template
  - **Unit Tests**: Implement tag parsing & increment script tests
  - **Manual Testing**: Execute dry-run release workflow producing signed artifacts
  - **Documentation**: Create RELEASE.md (versioning & branching model)
  - _Requirements: 7.1, 7.2, 7.3, 7.4_

- [x] 8. Multi-Arch Container Build & Signing (Foundational)





  - **Implementation**: Build amd64 + arm64 images (linux) for core services with cosign keyless signing + SBOM attestation integrated in CI
  - **Unit Tests**: Implement manifest list inspection test and signature presence test
  - **Manual Testing**: Pull signed image and verify cosign signature
  - **Documentation**: Create supply chain security section (extends security baseline doc)
  - _Requirements: 8.1, 8.2, 8.3, 8.4_

- [x] 9. Initial Risk Register & ADR Template





  - **Implementation**: Create /docs/risk-register.yaml with top ≥8 risks (id, desc, severity, mitigation link) and /docs/adr/ template committed + first ADR (architecture baseline)
  - **Unit Tests**: Implement risk YAML schema lint test and ADR filename pattern test
  - **Manual Testing**: Record review sign-off in PR comments
  - **Documentation**: Update CONTRIBUTING.md referencing ADR & risk processes
  - _Requirements: 9.1, 9.2, 9.3, 9.4_

- [x] 10. Operational Runbook Seed





  - **Implementation**: Create /docs/runbooks/index.md with placeholders (build failure, message backlog, cost spike) linking to future specs
  - **Unit Tests**: Implement link checker that passes
  - **Manual Testing**: Validate discoverability from root README
  - **Documentation**: Create runbook index (living document)
  - _Requirements: 10.1, 10.2, 10.3, 10.4_

## Exit Criteria Validation Tasks

The following tasks ensure all Gate G0 criteria are met:

- [x] 11. CI Green Including Security Scans Validation








  - **Implementation**: Ensure all workflows pass with no High/Critical vulnerabilities
  - **Unit Tests**: Validate CI workflow success and security scan thresholds
  - **Manual Testing**: Verify complete CI pipeline execution without security failures
  - **Documentation**: Document CI green validation procedures
  - _Requirements: 11.1_

- [x] 12. Cross-Platform Builds Validation





  - **Implementation**: Ensure Linux + Windows + WSL2 builds succeed
  - **Unit Tests**: Automated cross-platform build validation tests
  - **Manual Testing**: Execute builds on all supported platforms
  - **Documentation**: Cross-platform build troubleshooting guide
  - _Requirements: 11.2_

- [x] 13. Devcontainer Adoption Validation





  - **Implementation**: Ensure `af validate` warns outside container
  - **Unit Tests**: Test warning logic for container vs host execution
  - **Manual Testing**: Run `af validate` on host and verify warning display
  - **Documentation**: Devcontainer adoption guide
  - _Requirements: 11.3_

- [x] 14. SBOM & Provenance Validation










  - **Implementation**: Ensure artifacts published per build include SBOM & provenance
  - **Unit Tests**: Validate SBOM generation and provenance attestation
  - **Manual Testing**: Verify published artifacts contain required metadata
  - **Documentation**: SBOM and provenance verification procedures
  - _Requirements: 11.4_

- [x] 15. Final Gate G0 Validation & Spec Completion








  - **Implementation**: Complete all remaining Gate G0 criteria validation including signed multi-arch images (amd64+arm64 with cosign verification), risk register & ADR baseline merge, RELEASE.md publication with CI references, interface documentation snapshot (/docs/interfaces), and threat modeling session scheduling
  - **Unit Tests**: Comprehensive validation suite covering signature verification, governance artifact compliance, CI policy references, interface documentation completeness, and threat modeling entry validation
  - **Manual Testing**: End-to-end validation of all Gate G0 criteria including image signature verification, governance artifact accessibility, release process compliance, interface documentation accuracy, and threat modeling session confirmation
  - **Documentation**: Complete validation procedures covering image signing verification, governance artifact maintenance, release policy compliance, interface documentation maintenance, and threat modeling process
  - _Requirements: 11.5, 11.6, 11.7, 11.8, 11.9_

## Implementation Notes

### Task Execution Approach
- Each task must complete its Implementation, Unit Tests, Manual Testing, and Documentation components before proceeding
- Tasks 1-10 correspond exactly to the original Spec Q1.1 tasks
- Tasks 11-19 ensure all Gate G0 exit criteria are validated
- All tasks reference specific requirements for traceability

### Quality Standards
- Unit tests must achieve minimum 80% coverage for critical path code
- Manual testing must be reproducible with documented steps
- Documentation must be discoverable and linked from appropriate locations
- All code must pass linting and security scans

### Integration Requirements
- Each task builds incrementally on previous completed tasks
- No task should be considered complete until all four components (Impl/Unit/Manual/Docs) are finished
- Exit criteria validation tasks ensure the foundation meets all specified requirements
- Complete end-to-end validation must pass before considering the spec complete