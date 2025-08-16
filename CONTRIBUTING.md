# Contributing to AgentFlow

Welcome to AgentFlow! We're excited to have you contribute to building a production-ready multi-agent framework. This guide will help you understand our development process, governance, and contribution standards.

## Table of Contents

- [Getting Started](#getting-started)
- [Development Environment](#development-environment)
- [Contribution Process](#contribution-process)
- [Architecture Decision Records (ADRs)](#architecture-decision-records-adrs)
- [Risk Management](#risk-management)
- [Code Standards](#code-standards)
- [Testing Requirements](#testing-requirements)
- [Security Guidelines](#security-guidelines)
- [Documentation Standards](#documentation-standards)

## Getting Started

### Prerequisites

- Go 1.22 or later
- Docker and Docker Compose
- Git
- VS Code (recommended for devcontainer support)

### Quick Start

1. **Clone the repository**
   ```bash
   git clone https://github.com/agentflow/agentflow.git
   cd agentflow
   ```

2. **Set up development environment**
   ```bash
   # Using devcontainer (recommended)
   code .  # Open in VS Code and reopen in container
   
   # Or local setup
   go mod tidy
   ```

3. **Validate your environment**
   ```bash
   go run cmd/af/main.go validate
   ```

4. **Run tests**
   ```bash
   go test ./...
   ```

## Development Environment

### DevContainer (Recommended)

AgentFlow provides a complete development environment via VS Code devcontainer:

- Pre-configured Go 1.22+ environment
- NATS and PostgreSQL clients
- All required development tools
- Pre-commit hooks configured

### Local Development

If you prefer local development, ensure you have:

- Go 1.22+
- golangci-lint for code quality
- goose for database migrations
- sqlc for type-safe SQL generation

## Contribution Process

### 1. Issue Creation

- Check existing issues before creating new ones
- Use issue templates when available
- Provide clear reproduction steps for bugs
- Include relevant context and environment details

### 2. Branch Strategy

- Create feature branches from `main`
- Use descriptive branch names: `feature/add-tool-registry`, `fix/memory-leak`
- Keep branches focused on single features or fixes

### 3. Development Workflow

1. **Create your branch**
   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make your changes**
   - Follow code standards (see below)
   - Add tests for new functionality
   - Update documentation as needed

3. **Validate your changes**
   ```bash
   # Run tests
   go test ./...
   
   # Run linting
   golangci-lint run ./...
   
   # Validate governance artifacts (if modified)
   go run scripts/validate-governance.go all
   ```

4. **Commit your changes**
   ```bash
   git add .
   git commit -m "feat: add tool registry implementation"
   ```

5. **Push and create PR**
   ```bash
   git push origin feature/your-feature-name
   ```

### 4. Pull Request Requirements

- **Clear description** of changes and motivation
- **Link to related issues** using GitHub keywords
- **Tests included** for new functionality
- **Documentation updated** for user-facing changes
- **Security review** for security-sensitive changes
- **ADR created** for architectural decisions (see below)

## Architecture Decision Records (ADRs)

AgentFlow uses ADRs to document significant architectural decisions. This ensures transparency and helps future contributors understand the reasoning behind design choices.

### When to Create an ADR

Create an ADR when making decisions about:

- System architecture and design patterns
- Technology choices (databases, message queues, etc.)
- Security models and approaches
- API design and interfaces
- Performance and scalability strategies
- Development processes and tooling

### ADR Process

1. **Use the template**: Copy `/docs/adr/template.md` to create new ADRs
2. **Follow naming convention**: `ADR-NNNN-title-with-hyphens.md`
3. **Include all sections**: Status, Context, Decision, Consequences, Alternatives
4. **Get review**: ADRs require review from relevant team members
5. **Update status**: Mark as Accepted, Deprecated, or Superseded as appropriate

### ADR Example

```markdown
# ADR-0002: Message Queue Selection

## Status
Proposed

## Context
AgentFlow needs reliable message passing between control plane and workers...

## Decision
We will use NATS JetStream as our primary message queue...

## Consequences
### Positive
- Built-in persistence and clustering
- Excellent Go client library

### Negative  
- Additional operational complexity
- Learning curve for team
```

## Risk Management

AgentFlow maintains a comprehensive risk register to proactively identify and mitigate project risks.

### Risk Register

The risk register is maintained in `/docs/risk-register.yaml` and includes:

- **Risk identification** with unique IDs
- **Severity and probability** assessments
- **Impact descriptions** and mitigation strategies
- **Ownership** and review schedules
- **Links** to related ADRs and documentation

### Risk Process

1. **Identify risks** during development and planning
2. **Document in risk register** with proper classification
3. **Assign ownership** and mitigation strategies
4. **Regular reviews** (monthly) to update status
5. **Escalate high-severity risks** to project leadership

### Contributing to Risk Management

- **Report new risks** by updating the risk register
- **Validate schema** using `go run scripts/validate-governance.go risk-schema`
- **Participate in reviews** when assigned as risk owner
- **Link risks to ADRs** for architectural decisions

## Code Standards

### Go Code Style

- Follow standard Go conventions and idioms
- Use `gofmt` and `goimports` for formatting
- Pass `golangci-lint` with project configuration
- Write clear, self-documenting code with appropriate comments

### Package Organization

```
agentflow/
├── cmd/           # Application entry points
├── internal/      # Private packages (not importable)
├── pkg/           # Public API packages
├── sdk/           # Language SDKs
└── docs/          # Documentation
```

### Naming Conventions

- **Packages**: lowercase, single word when possible
- **Files**: lowercase with hyphens for multi-word names
- **Functions**: camelCase, exported functions start with capital
- **Constants**: UPPER_CASE for package-level constants

### Error Handling

- Use explicit error handling, avoid panic in library code
- Wrap errors with context using `fmt.Errorf` or error wrapping libraries
- Return meaningful error messages for user-facing operations

## Testing Requirements

### Test Coverage

- **Minimum 80% coverage** for critical path code
- **Unit tests** for all public functions and methods
- **Integration tests** for cross-component interactions
- **End-to-end tests** for complete workflows

### Test Organization

- Co-locate unit tests with source code (`*_test.go`)
- Place integration tests in `/test/integration/`
- Use table-driven tests for multiple test cases
- Mock external dependencies using interfaces

### Test Commands

```bash
# Run all tests
go test ./...

# Run tests with coverage
go test -cover ./...

# Run specific test package
go test ./pkg/agent/...

# Run integration tests
go test ./test/integration/...
```

## Security Guidelines

### Security-First Development

- **Threat modeling** for new features and components
- **Input validation** for all external inputs
- **Secure defaults** in configuration and APIs
- **Principle of least privilege** for permissions and access

### Security Review Process

1. **Self-review** using security checklist
2. **Peer review** for security-sensitive changes
3. **Security team review** for authentication, authorization, and cryptography
4. **Penetration testing** for major security features

### Security Tools

AgentFlow uses automated security scanning:

- **gosec**: Go security analyzer
- **osv-scanner**: Vulnerability database scanning
- **gitleaks**: Secret detection
- **syft/grype**: Container vulnerability scanning

Run security scans locally:
```bash
# Full security scan (when implemented)
make security-scan

# Individual tools
gosec ./...
gitleaks detect
```

## Documentation Standards

### Documentation Types

- **Code documentation**: Go doc comments for public APIs
- **Architecture documentation**: ADRs for design decisions
- **User documentation**: README files and usage guides
- **Operational documentation**: Runbooks and troubleshooting guides

### Documentation Requirements

- **Public APIs**: Must have Go doc comments
- **Complex algorithms**: Inline comments explaining logic
- **Configuration**: Document all configuration options
- **Examples**: Include usage examples for public APIs

### Documentation Tools

- Use Go doc comments for API documentation
- Write README files in Markdown
- Include code examples that compile and run
- Link related documentation and ADRs

## Getting Help

### Communication Channels

- **GitHub Issues**: Bug reports and feature requests
- **GitHub Discussions**: Questions and general discussion
- **Security Issues**: Use GitHub Security Advisories for vulnerabilities

### Resources

- [Architecture Documentation](/docs/ARCHITECTURE.md)
- [Development Environment Setup](/docs/dev-environment.md)
- [Security Baseline](/docs/security-baseline.md)
- [Risk Register](/docs/risk-register.yaml)
- [ADR Index](/docs/adr/)

## Recognition

We appreciate all contributions to AgentFlow! Contributors will be recognized in:

- Release notes for significant contributions
- CONTRIBUTORS.md file (coming soon)
- GitHub contributor statistics

Thank you for helping make AgentFlow better! 🚀

---

## Governance Validation

This document and related governance artifacts are validated using:

```bash
# Validate all governance artifacts
go run scripts/validate-governance.go all

# Validate risk register only
go run scripts/validate-governance.go risk-schema

# Validate ADR structure only  
go run scripts/validate-governance.go adr-filenames
```

For questions about governance processes, please refer to the [Risk Register](/docs/risk-register.yaml) or create an issue.