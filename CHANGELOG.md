# Changelog

All notable changes to AgentFlow will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Initial project structure and foundations
- Development environment with devcontainer
- CI/CD pipeline with security scanning
- Database migration tooling with goose and sqlc
- CLI validation tool (`af validate`)
 - Structured Logging Baseline (2025-08-17): JSON-structured logger with deterministic field ordering; automatic correlation enrichment (`trace_id`, `span_id`, `message_id`, `workflow_id`, `agent_id`); reserved-field validation to prevent accidental overrides; integrated across messaging operations (publish/consume/replay). Includes unit and integration tests (`pkg/messaging/logging_integration_test.go`), a manual ping-pong verification (`pkg/messaging/ping_pong_manual_test.go`), and documentation updates in `/docs/messaging.md`.

### Changed

### Deprecated

### Removed

### Fixed

### Security

## [0.1.0] - 2024-01-01

### Added
- Initial release of AgentFlow foundations
- Repository structure with Go modules for control plane, worker, and CLI
- Standardized development environment with VS Code devcontainer
- Comprehensive CI/CD pipeline with GitHub Actions
- Security tooling integration (gosec, osv-scanner, gitleaks, syft/grype)
- Database migration system with goose and sqlc
- CLI validation tool for environment verification
- Multi-architecture container builds with Cosign signing
- Project governance with risk register and ADR templates
- Operational runbook structure
- Cross-platform build support (Linux, Windows, WSL2)

### Security
- Implemented security baseline with vulnerability scanning
- Added secret detection with gitleaks
- Container image vulnerability scanning with syft/grype
- Supply chain security with SBOM generation and artifact signing

---

## Template for Future Releases

```markdown
## [X.Y.Z] - YYYY-MM-DD

### Added
- New features and capabilities

### Changed
- Changes to existing functionality

### Deprecated
- Features that will be removed in future versions

### Removed
- Features that have been removed

### Fixed
- Bug fixes and corrections

### Security
- Security improvements and vulnerability fixes
```

## Changelog Guidelines

### Categories

- **Added**: New features, capabilities, or enhancements
- **Changed**: Changes to existing functionality that don't break compatibility
- **Deprecated**: Features that will be removed in future versions (with timeline)
- **Removed**: Features that have been removed (breaking changes)
- **Fixed**: Bug fixes and error corrections
- **Security**: Security improvements, vulnerability fixes, and security-related changes

### Writing Guidelines

1. **User-Focused**: Write from the user's perspective, not internal implementation details
2. **Clear and Concise**: Use simple, direct language
3. **Actionable**: Include migration steps for breaking changes
4. **Linked**: Reference GitHub issues/PRs where relevant
5. **Grouped**: Group related changes together

### Examples

#### Good Changelog Entries

```markdown
### Added
- New `af deploy` command for simplified deployment to Kubernetes clusters (#123)
- Support for custom tool execution timeouts in workflow configuration (#145)
- Real-time cost tracking dashboard for LLM usage monitoring (#167)

### Changed
- Improved error messages in CLI validation tool with specific remediation steps (#134)
- Updated default memory limits for worker containers from 512MB to 1GB (#156)

### Fixed
- Fixed Windows path handling in migration scripts that caused deployment failures (#142)
- Resolved race condition in message processing that could cause duplicate executions (#158)

### Security
- Updated Go dependencies to address CVE-2024-1234 in crypto library (#171)
- Added input validation for tool parameters to prevent injection attacks (#183)
```

#### Poor Changelog Entries

```markdown
### Changed
- Refactored internal code structure
- Updated dependencies
- Fixed bugs
- Improved performance
```

### Version Links

At the bottom of the changelog, maintain links to version comparisons:

```markdown
[Unreleased]: https://github.com/agentflow/agentflow/compare/v0.1.0...HEAD
[0.1.0]: https://github.com/agentflow/agentflow/releases/tag/v0.1.0
```

### Automation

The changelog should be updated:
- **During development**: Add entries to [Unreleased] section
- **At release time**: Move [Unreleased] entries to new version section
- **Automatically**: CI can validate changelog format and completeness