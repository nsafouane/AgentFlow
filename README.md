# AgentFlow

AgentFlow is a production-ready multi-agent framework designed to eliminate the complexity gap between AI prototypes and production systems. It delivers enterprise-grade multi-agent orchestration with deterministic planning, cost-aware execution, and security-first architecture.

## Architecture Overview

AgentFlow follows a modular architecture with clear separation between control plane, data plane, and client interfaces.

### Repository Structure

```
agentflow/
├── cmd/                          # CLI applications & service entry points
│   ├── af/                       # Main CLI tool
│   ├── control-plane/            # Control plane service
│   └── worker/                   # Worker service
├── internal/                     # Shared internal packages (not importable)
│   ├── config/                   # Configuration management
│   ├── logging/                  # Structured logging utilities
│   ├── metrics/                  # Observability & metrics
│   └── security/                 # Security utilities & auth
├── pkg/                          # Public API packages (importable)
│   ├── agent/                    # Agent interfaces & runtime
│   ├── planner/                  # Planning interfaces (FSM, BT, LLM)
│   ├── tools/                    # Tool interfaces & registry
│   ├── memory/                   # Memory store interfaces
│   └── messaging/                # Message bus abstractions
├── sdk/                          # Language SDKs
│   ├── go/                       # Go SDK (primary)
│   ├── python/                   # Python SDK (stub)
│   └── javascript/               # JavaScript SDK (stub)
├── dashboard/                    # Web dashboard (stub)
├── docs/                         # Documentation
├── .devcontainer/                # Development environment config
├── .github/                      # CI/CD workflows
├── Makefile                      # Build automation (primary)
└── Taskfile.yml                  # Cross-platform task runner
```

### Module Boundaries

#### Control Plane (`cmd/control-plane/`)
- REST/gRPC APIs for workflow management
- Orchestrator for plan execution
- Registry for tools, templates, and policies
- Dashboard backend services

#### Data Plane (`cmd/worker/`)
- Agent runtime execution
- Message processing
- Tool execution with sandboxing
- Memory operations

#### CLI Tool (`cmd/af/`)
- Project initialization and templates
- Configuration management
- Deployment commands
- Validation and debugging tools

### Package Design Principles

#### Internal Packages (`internal/`)
- **Not importable** by external projects
- Shared utilities across AgentFlow services
- Implementation details that may change
- Security-sensitive code

#### Public API Packages (`pkg/`)
- **Stable interfaces** for external consumption
- Minimal dependencies
- Well-documented with examples
- Backward compatibility guarantees

#### Service Commands (`cmd/`)
- Thin main functions
- Dependency injection setup
- Configuration loading
- Service lifecycle management

## Repository Conventions

### Go Module Structure
- Each service (`cmd/*`) has its own `go.mod` for independent versioning
- SDK modules (`sdk/*`) have separate `go.mod` files for language-specific dependencies
- Root `go.mod` contains shared dependencies and internal packages
- All modules use replace directives to reference the root module during development

### Build System
- **Makefile**: Primary build system for Unix-like systems
- **Taskfile.yml**: Cross-platform task runner for Windows compatibility
- Both build systems provide equivalent functionality:
  - `build`: Compile all components
  - `test`: Run all tests
  - `lint`: Run code quality checks
  - `clean`: Remove build artifacts
  - `deps`: Update dependencies

### Cross-Platform Support
- All build scripts work on Linux, Windows, and WSL2
- Platform-specific binary extensions handled automatically
- Path normalization for Windows compatibility
- Consistent behavior across different development environments

### Code Quality Standards
- golangci-lint configuration in `.golangci.yml`
- Minimum test coverage requirements
- Consistent code formatting with gofmt/goimports
- Security scanning with gosec integration

### Naming Conventions
- **Go Packages**: Lowercase, single word when possible
- **Files**: Lowercase with hyphens for multi-word names
- **Directories**: Clear purpose indication
- **Modules**: Follow Go module naming conventions

## Getting Started

### Development Environment Setup

The fastest way to get started is using our pre-configured VS Code devcontainer:

1. **Prerequisites**:
   - [VS Code](https://code.visualstudio.com/)
   - [Docker Desktop](https://www.docker.com/products/docker-desktop/)
   - [Dev Containers extension](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers)

2. **Quick Setup**:
   ```bash
   git clone https://github.com/agentflow/agentflow.git
   cd agentflow
   code .
   # Click "Reopen in Container" when prompted
   ```

3. **Verify Environment**:
   ```bash
   af validate
   ```

For detailed setup instructions, including Windows/macOS host setup, see [Development Environment Guide](docs/dev-environment.md). For devcontainer adoption guidance, see [Devcontainer Adoption Guide](docs/devcontainer-adoption-guide.md).

### CLI Usage

The AgentFlow CLI (`af`) provides essential development and validation tools:

- **Environment Validation**: `af validate` - Comprehensive environment check
- **JSON Output**: Structured validation results for automation
- **Cross-Platform**: Works on Linux, Windows, macOS, and containers

For complete CLI documentation, see [CLI Quickstart Guide](docs/cli-quickstart.md).

### Quick Start

1. **Build all components**:
   ```bash
   make build
   # or
   task build
   ```

2. **Run tests**:
   ```bash
   make test
   # or
   task test
   ```

3. **Start development services**:
   ```bash
   make dev
   # or
   task dev
   ```

### Database Migrations

AgentFlow uses goose for database schema management and sqlc for type-safe query generation:

1. **Create new migration**:
   ```bash
   make migrate-create NAME=add_user_table
   # or
   task migrate-create NAME=add_user_table
   ```

2. **Run migrations**:
   ```bash
   make migrate-up
   # or
   task migrate-up
   ```

3. **Generate type-safe queries**:
   ```bash
   make sqlc-generate
   # or
   task sqlc-generate
   ```

For detailed migration policies and procedures, see [Migration Policy](docs/migration-policy.md).

### Development Workflow

1. **Code Quality**: Run linting before committing
   ```bash
   make lint
   ```

2. **Testing**: Ensure all tests pass
   ```bash
   make test
   ```

3. **Cross-Platform**: Test builds on multiple platforms
   ```bash
   make build-all
   ```

## Documentation

### Architecture & Design
- [Architecture Documentation](docs/ARCHITECTURE.md) - System architecture and design decisions
- [ADR Directory](docs/adr/) - Architecture Decision Records
- [Risk Register](docs/risk-register.yaml) - Project risk management

### Development Guides
- [Development Environment Guide](docs/dev-environment.md) - Detailed setup instructions
- [CLI Quickstart Guide](docs/cli-quickstart.md) - Command-line tool usage
- [Migration Policy](docs/migration-policy.md) - Database schema management

### Operations & Troubleshooting
- [Operational Runbooks](docs/runbooks/) - Troubleshooting and maintenance procedures
- [Security Baseline](docs/security-baseline.md) - Security policies and procedures
- [CI Policy & Gating](docs/ci-policy.md) - Continuous integration guidelines

## Contributing

Please read our contributing guidelines and ensure all builds pass on supported platforms before submitting pull requests.

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

Apache 2.0 is an OSI-approved open source license that allows:
- ✅ Commercial use
- ✅ Modification  
- ✅ Distribution
- ✅ Private use
- ✅ Patent use

The license requires:
- 📄 License and copyright notice
- 📝 State changes made to the code