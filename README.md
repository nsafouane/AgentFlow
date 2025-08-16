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

### Prerequisites
- Go 1.22+
- Docker (for development environment)
- make or Task runner

### Quick Start

1. **Clone the repository**:
   ```bash
   git clone https://github.com/agentflow/agentflow.git
   cd agentflow
   ```

2. **Install dependencies**:
   ```bash
   make deps
   # or
   task deps
   ```

3. **Build all components**:
   ```bash
   make build
   # or
   task build
   ```

4. **Run tests**:
   ```bash
   make test
   # or
   task test
   ```

5. **Start development environment**:
   ```bash
   make dev
   # or
   task dev
   ```

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

## Contributing

Please read our contributing guidelines and ensure all builds pass on supported platforms before submitting pull requests.

## License

[License information will be added]