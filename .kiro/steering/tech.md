# AgentFlow Technical Stack

## Core Technology Stack

**Language**: Go 1.22+ (primary), with WASM runtime support for Python, Rust, C#, TypeScript agents

**Architecture**: Control Plane + Data Plane separation with pluggable components

**Message Bus**: NATS JetStream (default), with support for Redis Streams, RabbitMQ

**Storage**: 
- State Store: PostgreSQL (production), Badger/SQLite (development)
- Vector DB: Pinecone (default), Weaviate, local embeddings
- Cache: Redis for short-term memory and response caching

**Container Runtime**: Docker with gVisor sandboxing for secure tool execution

## Build System & Tools

**Build Automation**: 
- Makefile + Taskfile.yml for cross-platform builds
- Multi-architecture container builds (amd64, arm64)

**Database Management**:
- goose for schema migrations
- sqlc for type-safe SQL code generation

**Code Quality**:
- golangci-lint for Go linting
- gosec for security analysis
- Pre-commit hooks for automated checks

**Security Scanning**:
- osv-scanner for vulnerability detection
- gitleaks for secret scanning
- syft/grype for container scanning

## Development Environment

**Containerized Development**: VS Code devcontainer with pinned tool versions

**Required Tools**:
- Go 1.22+
- Docker & Docker Compose
- NATS client tools
- PostgreSQL client
- Redis client

## Common Commands

### Development
```bash
# Environment validation
af validate

# Start development environment
make dev
# or
task dev

# Run tests
make test
task test

# Build all components
make build
task build
```

### Database Operations
```bash
# Run migrations
goose -dir migrations postgres "connection_string" up

# Rollback migration
goose -dir migrations postgres "connection_string" down

# Generate type-safe queries
sqlc generate
```

### Security & Quality
```bash
# Run security scans
make security-scan
task security-scan

# Run linting
golangci-lint run

# Check for secrets
gitleaks detect
```

### Deployment
```bash
# Build containers
make containers
task containers

# Deploy to development
af deploy --env=dev

# Deploy to production
af deploy --env=prod --provider=aws
```

## Performance Targets

- Message routing: p95 < 15ms
- Worker execution: p95 < 100ms
- Plan generation: p95 < 200ms
- Tool execution: p95 < 5s (varies by tool)
- Cold start recovery: < 5 seconds
- Cross-platform build: < 10 minutes

## Observability Stack

**Metrics**: Prometheus with custom AgentFlow metrics
**Tracing**: OpenTelemetry with Jaeger backend
**Logging**: Structured JSON logs with correlation IDs
**Dashboards**: Grafana with pre-built AgentFlow dashboards