# AgentFlow Project Structure

## Repository Organization

AgentFlow follows a modular Go project structure with clear separation between public APIs, internal packages, and service implementations.

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
│   ├── security/                 # Security utilities & auth
│   └── storage/                  # Database & storage abstractions
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
│   ├── adr/                      # Architecture Decision Records
│   ├── runbooks/                 # Operational runbooks
│   ├── interfaces/               # Interface documentation
│   └── risk-register.yaml        # Risk management
├── migrations/                   # Database schema migrations
├── .devcontainer/                # Development environment config
├── .github/                      # CI/CD workflows
├── .kiro/                        # Kiro-specific configuration
│   ├── steering/                 # AI assistant steering rules
│   └── specs/                    # Development specifications
├── Makefile                      # Build automation (primary)
├── Taskfile.yml                  # Cross-platform task runner
└── docker-compose.yml            # Local development services
```

## Module Boundaries

### Control Plane (`cmd/control-plane/`)
- REST/gRPC APIs for workflow management
- Orchestrator for plan execution
- Registry for tools, templates, and policies
- Dashboard backend services

### Data Plane (`cmd/worker/`)
- Agent runtime execution
- Message processing
- Tool execution with sandboxing
- Memory operations

### CLI Tool (`cmd/af/`)
- Project initialization and templates
- Configuration management
- Deployment commands
- Validation and debugging tools

## Package Design Principles

### Internal Packages (`internal/`)
- **Not importable** by external projects
- Shared utilities across AgentFlow services
- Implementation details that may change
- Security-sensitive code

### Public API Packages (`pkg/`)
- **Stable interfaces** for external consumption
- Minimal dependencies
- Well-documented with examples
- Backward compatibility guarantees

### Service Commands (`cmd/`)
- Thin main functions
- Dependency injection setup
- Configuration loading
- Service lifecycle management

## Configuration Structure

### Environment-based Configuration
```
config/
├── development.yaml              # Development defaults
├── production.yaml               # Production settings
├── test.yaml                     # Test configuration
└── local.yaml                    # Local overrides (gitignored)
```

### Feature Flags
- `data_minimization`: Enable PII redaction
- `residency_strict`: Enforce on-premise model usage
- `replay_safe_mode`: Disable side effects for replay
- `llm_gateway.enabled`: Enable LLM model gateway
- `anomaly_detector`: Enable tool anomaly detection
- `sandbox_strict`: Use gVisor instead of Docker

## Development Workflow Structure

### Specifications (`/.kiro/specs/`)
- Requirements, design, and implementation plans
- Organized by development quarters and features
- Links to tasks and exit criteria

### Steering Rules (`/.kiro/steering/`)
- AI assistant guidance documents
- Project conventions and standards
- Technical decision context

### Documentation (`/docs/`)
- Architecture Decision Records (ADRs)
- Operational runbooks
- API documentation
- Risk management artifacts

## Testing Structure

### Unit Tests
- Co-located with source code (`*_test.go`)
- Interface-based mocking
- 80% minimum coverage requirement

### Integration Tests
- `/test/integration/` directory
- End-to-end workflow testing
- Cross-service communication validation

### Performance Tests
- `/test/performance/` directory
- Benchmark scenarios
- SLA validation tests

## Deployment Artifacts

### Container Images
- Multi-architecture builds (amd64, arm64)
- Signed with Cosign
- SBOM and provenance attestation

### Kubernetes Manifests
- `/deploy/k8s/` directory
- Helm charts for complex deployments
- Environment-specific overlays

### Templates
- `/templates/` directory
- Workflow templates (customer-support, content-pipeline)
- Configuration examples
- Quick-start scenarios

## Naming Conventions

### Go Packages
- Lowercase, single word when possible
- Descriptive but concise (`planner` not `planningengine`)
- Avoid stuttering (`agent.Agent` not `agent.AgentAgent`)

### Files and Directories
- Lowercase with hyphens for multi-word names
- Clear purpose indication (`control-plane`, `risk-register.yaml`)
- Consistent suffixes (`_test.go`, `.yaml`, `.md`)

### Database Tables
- Lowercase with underscores
- Plural nouns (`agents`, `workflows`, `plans`)
- Foreign key suffix `_id` (`tenant_id`, `workflow_id`)

## Security Boundaries

### Tenant Isolation
- All data scoped by `tenant_id`
- Message subjects include tenant prefix
- RBAC enforcement at API layer

### Tool Sandboxing
- Execution profiles define permissions
- gVisor containers for isolation
- Audit logging for all tool calls

### Secret Management
- No secrets in configuration files
- Environment variables or external providers
- Rotation procedures documented