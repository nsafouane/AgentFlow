# Requirements Document

## Introduction

The Foundations & Project Governance spec establishes the foundational infrastructure, tooling, and governance processes required for the AgentFlow multi-agent framework development. This spec creates the essential development environment, CI/CD pipeline, security baseline, and project governance structures that will enable all subsequent development phases.

AgentFlow is a production-ready multi-agent framework with deterministic planning, enterprise security, and cost-aware execution. The foundations must support a 3-quarter development plan targeting MVP foundation (Q1), enterprise readiness (Q2), and scale & advanced capabilities (Q3).

This spec relies on none (root) and enables all subsequent specs.

## Requirements

### Requirement 1

**User Story:** As a developer, I want a standardized repository structure and module layout so that I can work consistently across the AgentFlow codebase.

#### Acceptance Criteria

1. WHEN the repository structure is implemented THEN it SHALL create Go modules for control plane, worker, cli, sdk stubs, and dashboard stub with shared internal packages layout and root Makefile/Taskfile
2. WHEN lint configuration tests run THEN golangci-lint SHALL pass and placeholder unit test runs SHALL execute for each module
3. WHEN builds are manually executed THEN they SHALL succeed on Linux + Windows + WSL2 and verify task runner parity
4. WHEN architecture documentation is created THEN it SHALL include README section describing repo conventions

### Requirement 2

**User Story:** As a developer, I want a standardized development environment so that I can contribute consistently across different machines and operating systems.

#### Acceptance Criteria

1. WHEN the devcontainer is implemented THEN it SHALL include .devcontainer with pinned Go, NATS, Postgres clients, and pre-commit hooks
2. WHEN binary validation script tests run THEN they SHALL validate required binaries versions
3. WHEN the devcontainer is manually tested THEN opening in VS Code SHALL provision the environment and running `af validate` stub SHALL work
4. WHEN dev environment documentation is created THEN it SHALL include dev environment guide with Windows fallback notes

### Requirement 3

**User Story:** As a developer, I want a comprehensive CI pipeline so that code quality, security, and build integrity are automatically enforced.

#### Acceptance Criteria

1. WHEN CI pipeline is implemented THEN it SHALL include GitHub Actions workflows for Build, Lint, Test, SBOM, SAST, Dependencies, Secrets, License, Container Scan with cache strategy and provenance attestation
2. WHEN workflow tests run THEN workflow dry-run using act/minimal branch test SHALL pass and config schema lint SHALL validate
3. WHEN CI is manually tested THEN forcing failing job for dependency vulnerability SHALL confirm block
4. WHEN CI documentation is created THEN it SHALL include CI policy & gating doc

### Requirement 4

**User Story:** As a security engineer, I want integrated security tooling so that vulnerabilities and secrets are detected early in the development process.

#### Acceptance Criteria

1. WHEN security tooling is implemented THEN it SHALL include scripts and severity thresholds (fail High/Critical) for gosec, osv-scanner, gitleaks, syft/grype
2. WHEN security tooling tests run THEN mock reports SHALL be parsed and threshold logic SHALL be tested
3. WHEN security tooling is manually tested THEN introducing benign vulnerable lib in branch SHALL ensure failure
4. WHEN security documentation is created THEN it SHALL include security baseline & exception process

### Requirement 5

**User Story:** As a database administrator, I want reliable migration tooling so that database schema changes can be managed safely.

#### Acceptance Criteria

1. WHEN migration tooling is implemented THEN it SHALL pin versions, add initial empty migration, and configure sqlc for goose + sqlc
2. WHEN migration tests run THEN migration linter test SHALL pass and sqlc code SHALL compile
3. WHEN migrations are manually tested THEN up/down SHALL run locally with Windows path validation
4. WHEN migration documentation is created THEN it SHALL include migration policy (naming, reversibility stance)

### Requirement 6

**User Story:** As a developer, I want a CLI validation tool so that I can quickly verify my development environment setup.

#### Acceptance Criteria

1. WHEN CLI `af validate` stub is implemented THEN it SHALL output JSON skeleton with environment probes placeholders
2. WHEN CLI tests run THEN JSON schema validation test SHALL pass
3. WHEN CLI is manually tested THEN running on host vs devcontainer SHALL display warning
4. WHEN CLI documentation is created THEN it SHALL include CLI usage quickstart

### Requirement 7

**User Story:** As a release manager, I want standardized versioning and release processes so that releases are consistent and traceable.

#### Acceptance Criteria

1. WHEN versioning & release engineering baseline is implemented THEN it SHALL define semantic version scheme (pre-1.0 minor for breaking changes), tagging policy, and CHANGELOG template
2. WHEN versioning tests run THEN tag parsing & increment script tests SHALL pass
3. WHEN release process is manually tested THEN dry-run release workflow SHALL produce signed artifacts
4. WHEN release documentation is created THEN it SHALL include RELEASE.md (versioning & branching model)

### Requirement 8

**User Story:** As a security engineer, I want multi-architecture container builds with signing so that supply chain security is established.

#### Acceptance Criteria

1. WHEN multi-arch container build is implemented THEN it SHALL build amd64 + arm64 images (linux) for core services with cosign keyless signing + SBOM attestation integrated in CI
2. WHEN container build tests run THEN manifest list inspection test SHALL pass and signature presence test SHALL validate
3. WHEN container builds are manually tested THEN pulling signed image and verifying cosign signature SHALL succeed
4. WHEN supply chain documentation is created THEN it SHALL include supply chain security section (extends security baseline doc)

### Requirement 9

**User Story:** As a project stakeholder, I want comprehensive risk management and decision documentation so that project governance is established.

#### Acceptance Criteria

1. WHEN initial risk register & ADR template are implemented THEN /docs/risk-register.yaml SHALL contain top ≥8 risks (id, desc, severity, mitigation link) and /docs/adr/ template SHALL be committed + first ADR (architecture baseline)
2. WHEN governance tests run THEN risk YAML schema lint test SHALL pass and ADR filename pattern test SHALL validate
3. WHEN governance is manually tested THEN review sign-off SHALL be recorded in PR comments
4. WHEN governance documentation is created THEN CONTRIBUTING.md SHALL be updated referencing ADR & risk processes

### Requirement 10

**User Story:** As an operations engineer, I want operational runbooks so that I can effectively troubleshoot and maintain the system.

#### Acceptance Criteria

1. WHEN operational runbook seed is implemented THEN it SHALL create /docs/runbooks/index.md with placeholders (build failure, message backlog, cost spike) linking to future specs
2. WHEN runbook tests run THEN link checker SHALL pass
3. WHEN runbooks are manually tested THEN discoverability from root README SHALL be validated
4. WHEN runbook documentation is created THEN it SHALL include runbook index (living document)

### Requirement 11 - Exit Criteria Compliance

**User Story:** As a project manager, I want to ensure all Gate G0 criteria are met so that the foundation is properly established for subsequent development phases.

#### Acceptance Criteria

1. WHEN CI workflows are complete THEN all workflows SHALL pass with no High/Critical vulnerabilities (CI green incl. security scans)
2. WHEN cross-platform builds are tested THEN Linux + Windows + WSL2 builds SHALL succeed (Cross-platform builds)
3. WHEN devcontainer adoption is verified THEN `af validate` SHALL warn outside container (Devcontainer adoption)
4. WHEN build artifacts are generated THEN SBOM & provenance SHALL be published per build (SBOM & provenance)
5. WHEN container images are built THEN amd64+arm64 images SHALL be pushed and cosign verify SHALL pass (Signed multi-arch images)
6. WHEN governance artifacts are complete THEN risk-register.yaml + first ADR SHALL be merged (Risk register & ADR baseline)
7. WHEN release process is established THEN RELEASE.md SHALL be published & referenced by CI (Release versioning policy)
8. WHEN interface documentation is complete THEN /docs/interfaces (core Q1 interfaces) SHALL be committed & referenced (Interface freeze snapshot)
9. WHEN threat modeling is scheduled THEN threat modeling session date & owner SHALL be logged in risk register (Threat model kickoff scheduled)