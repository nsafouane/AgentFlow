# AgentFlow Architecture

## Overview

AgentFlow implements a distributed multi-agent framework with clear separation of concerns between control plane operations, data plane execution, and client interfaces.

## System Architecture

```
┌─────────────────────── CLIENT LAYER ───────────────────────┐
│  ┌──────────────────┐  ┌──────────────────┐  ┌─────────────┐  │
│  │   CLI Tool       │  │   Go SDK         │  │  Dashboard  │  │  
│  │   (cmd/af)       │  │   (sdk/go)       │  │  (Web UI)   │  │
│  └──────────────────┘  └──────────────────┘  └─────────────┘  │
└──────────────────────────────────────────────────────────────┘
                               │
┌─────────────────────── CONTROL PLANE ──────────────────────┐
│  ┌──────────────────┐  ┌──────────────────┐  ┌─────────────┐  │
│  │ Workflow Manager │  │  Plan Executor   │  │  Registry   │  │
│  │ • REST/gRPC API  │  │ • Orchestration  │  │ • Tools     │  │
│  │ • Authentication │  │ • State Machine  │  │ • Templates │  │
│  │ • Authorization  │  │ • Error Handling │  │ • Policies  │  │
│  └──────────────────┘  └──────────────────┘  └─────────────┘  │
└──────────────────────────────────────────────────────────────┘
                               │
┌─────────────────────── MESSAGE BUS ────────────────────────┐
│                    NATS JetStream                          │
│              (with Redis/RabbitMQ support)                 │
└──────────────────────────────────────────────────────────────┘
                               │
┌─────────────────────── DATA PLANE ─────────────────────────┐
│  ┌──────────────────┐  ┌──────────────────┐  ┌─────────────┐  │
│  │ Agent Runtime    │  │  Tool Executor   │  │ Memory Mgr  │  │
│  │ • Agent Lifecycle│  │ • Sandboxing     │  │ • Vector DB │  │
│  │ • Message Proc   │  │ • Security       │  │ • Cache     │  │
│  │ • Load Balancing │  │ • Monitoring     │  │ • State     │  │
│  └──────────────────┘  └──────────────────┘  └─────────────┘  │
└──────────────────────────────────────────────────────────────┘
```

## Module Architecture

### Control Plane (`cmd/control-plane`)

**Responsibilities**:
- Workflow lifecycle management
- Plan execution orchestration
- Tool and template registry
- Authentication and authorization
- API gateway functionality

**Key Components**:
- **API Server**: REST/gRPC endpoints for client interactions
- **Orchestrator**: Manages workflow execution state
- **Registry**: Stores and manages tools, templates, and policies
- **Scheduler**: Distributes work to data plane workers

### Data Plane (`cmd/worker`)

**Responsibilities**:
- Agent runtime execution
- Tool execution with security sandboxing
- Message processing and routing
- Memory and state management

**Key Components**:
- **Agent Runtime**: Executes agent logic and manages lifecycle
- **Tool Executor**: Secure execution environment for tools
- **Message Handler**: Processes messages from control plane
- **Memory Manager**: Handles short-term and long-term memory

### CLI Tool (`cmd/af`)

**Responsibilities**:
- Project initialization and scaffolding
- Configuration management
- Deployment automation
- Development and debugging utilities

**Key Features**:
- Template-based project creation
- Environment validation
- Workflow deployment
- Log aggregation and debugging

## Package Architecture

### Internal Packages (`internal/`)

#### Configuration (`internal/config`)
- Environment-based configuration loading
- Feature flag management
- Validation and defaults
- Hot-reload capabilities

#### Logging (`internal/logging`)
- Structured JSON logging
- Correlation ID tracking
- Log level management
- Multiple output destinations

#### Metrics (`internal/metrics`)
- Prometheus-compatible metrics
- Custom AgentFlow metrics
- Performance monitoring
- Resource utilization tracking

#### Security (`internal/security`)
- Authentication providers
- Authorization policies
- Token management
- Audit logging

### Public API Packages (`pkg/`)

#### Agent (`pkg/agent`)
- Agent interface definitions
- Execution context management
- Input/output specifications
- Registry interfaces

#### Planner (`pkg/planner`)
- Planning algorithm interfaces
- FSM (Finite State Machine) planner
- Behavior Tree planner
- LLM-based planner
- Plan validation and optimization

#### Tools (`pkg/tools`)
- Tool interface definitions
- Execution profiles and security
- Registry and discovery
- Sandboxing specifications

#### Memory (`pkg/memory`)
- Memory store abstractions
- Vector database interfaces
- Caching strategies
- State persistence

#### Messaging (`pkg/messaging`)
- Message bus abstractions
- Publisher/subscriber patterns
- Message routing and filtering
- Delivery guarantees

## SDK Architecture

### Go SDK (`sdk/go`)
- Native Go client library
- Type-safe API bindings
- Connection management
- Error handling and retries

### Python SDK (`sdk/python`) - Stub
- Python client library (planned)
- Async/await support
- Integration with popular ML libraries
- Jupyter notebook compatibility

### JavaScript SDK (`sdk/javascript`) - Stub
- TypeScript-first implementation (planned)
- Browser and Node.js support
- React/Vue component library
- WebSocket real-time updates

## Data Flow

### Workflow Execution Flow

1. **Client Request**: SDK/CLI submits workflow request to Control Plane
2. **Authentication**: Control Plane validates client credentials
3. **Planning**: Planner generates execution plan based on workflow definition
4. **Orchestration**: Orchestrator breaks plan into discrete tasks
5. **Distribution**: Tasks distributed to Data Plane workers via message bus
6. **Execution**: Workers execute agents and tools in secure sandboxes
7. **Coordination**: Results aggregated and next steps determined
8. **Completion**: Final results returned to client

### Message Flow

```
Client → Control Plane → Message Bus → Data Plane → Message Bus → Control Plane → Client
```

## Security Architecture

### Tenant Isolation
- All data scoped by tenant ID
- Message subjects include tenant prefix
- RBAC enforcement at API layer
- Resource quotas and limits

### Tool Sandboxing
- gVisor containers for isolation
- Execution profiles define permissions
- Resource limits (CPU, memory, network)
- Audit logging for all tool executions

### Authentication & Authorization
- JWT-based authentication
- Role-based access control (RBAC)
- API key management
- Service-to-service authentication

## Deployment Architecture

### Container Strategy
- Multi-architecture builds (amd64, arm64)
- Minimal base images for security
- Signed containers with Cosign
- SBOM and provenance attestation

### Kubernetes Deployment
- Helm charts for complex deployments
- Horizontal pod autoscaling
- Service mesh integration (optional)
- Persistent volume management

### Development Environment
- VS Code devcontainer support
- Docker Compose for local development
- Hot-reload capabilities
- Integrated debugging tools

## Performance Considerations

### Scalability Targets
- Message routing: p95 < 15ms
- Worker execution: p95 < 100ms
- Plan generation: p95 < 200ms
- Tool execution: p95 < 5s (varies by tool)

### Optimization Strategies
- Connection pooling and reuse
- Message batching and compression
- Caching at multiple layers
- Lazy loading of resources

## Monitoring and Observability

### Metrics Collection
- Prometheus-compatible metrics
- Custom AgentFlow dashboards
- SLA monitoring and alerting
- Resource utilization tracking

### Distributed Tracing
- OpenTelemetry integration
- Request correlation across services
- Performance bottleneck identification
- Error propagation tracking

### Logging Strategy
- Structured JSON logs
- Centralized log aggregation
- Correlation ID tracking
- Security event logging

## Future Architecture Considerations

### Planned Enhancements
- Multi-region deployment support
- Advanced scheduling algorithms
- Plugin architecture for extensibility
- GraphQL API layer
- Event sourcing for audit trails

### Scalability Roadmap
- Horizontal scaling of control plane
- Distributed consensus for high availability
- Edge deployment capabilities
- Advanced caching strategies