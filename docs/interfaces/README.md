# AgentFlow Core Interfaces Documentation

## Overview

This document provides a snapshot of the core interfaces that define AgentFlow's Q1 foundation architecture. These interfaces represent the stable API contracts that will be maintained throughout the development lifecycle.

**Interface Freeze Date**: 2025-01-16  
**Spec Version**: Q1.1 Foundations & Project Governance  
**Status**: Baseline Established

## Core Interface Categories

### 1. Agent Runtime Interfaces

#### Agent Interface
```go
// pkg/agent/agent.go
type Agent interface {
    // Execute runs the agent with the given context and input
    Execute(ctx context.Context, input AgentInput) (AgentOutput, error)
    
    // GetCapabilities returns the agent's declared capabilities
    GetCapabilities() AgentCapabilities
    
    // Validate checks if the agent configuration is valid
    Validate() error
}

type AgentInput struct {
    TenantID    string                 `json:"tenant_id"`
    WorkflowID  string                 `json:"workflow_id"`
    PlanID      string                 `json:"plan_id"`
    StepID      string                 `json:"step_id"`
    Context     map[string]interface{} `json:"context"`
    Tools       []ToolReference        `json:"tools"`
    Memory      MemoryContext          `json:"memory"`
}

type AgentOutput struct {
    Result      interface{}            `json:"result"`
    Context     map[string]interface{} `json:"context"`
    ToolCalls   []ToolCall            `json:"tool_calls"`
    TokenUsage  TokenUsage            `json:"token_usage"`
    Error       *AgentError           `json:"error,omitempty"`
}
```

#### Agent Registry Interface
```go
// pkg/agent/registry.go
type Registry interface {
    // Register adds an agent to the registry
    Register(agent Agent) error
    
    // Get retrieves an agent by name
    Get(name string) (Agent, error)
    
    // List returns all registered agents
    List() []AgentInfo
    
    // Unregister removes an agent from the registry
    Unregister(name string) error
}
```

### 2. Planning Interfaces

#### Planner Interface
```go
// pkg/planner/planner.go
type Planner interface {
    // Plan generates an execution plan for the given request
    Plan(ctx context.Context, request PlanRequest) (Plan, error)
    
    // Validate checks if a plan is valid
    Validate(plan Plan) error
    
    // GetType returns the planner type (FSM, BehaviorTree, LLM)
    GetType() PlannerType
}

type PlanRequest struct {
    TenantID     string                 `json:"tenant_id"`
    WorkflowID   string                 `json:"workflow_id"`
    Goal         string                 `json:"goal"`
    Context      map[string]interface{} `json:"context"`
    Constraints  PlanConstraints        `json:"constraints"`
    Tools        []ToolReference        `json:"tools"`
}

type Plan struct {
    ID           string      `json:"id"`
    TenantID     string      `json:"tenant_id"`
    WorkflowID   string      `json:"workflow_id"`
    Type         PlannerType `json:"type"`
    Steps        []PlanStep  `json:"steps"`
    Dependencies []Dependency `json:"dependencies"`
    EstimatedCost CostEstimate `json:"estimated_cost"`
    CreatedAt    time.Time   `json:"created_at"`
}
```

#### Finite State Machine Planner
```go
// pkg/planner/fsm.go
type FSMPlanner interface {
    Planner
    
    // AddState adds a state to the FSM
    AddState(state State) error
    
    // AddTransition adds a transition between states
    AddTransition(from, to string, condition Condition) error
    
    // GetCurrentState returns the current state
    GetCurrentState() State
}
```

### 3. Tool Execution Interfaces

#### Tool Interface
```go
// pkg/tools/tool.go
type Tool interface {
    // Execute runs the tool with the given input
    Execute(ctx context.Context, input ToolInput) (ToolOutput, error)
    
    // GetSchema returns the tool's input/output schema
    GetSchema() ToolSchema
    
    // GetPermissions returns required permissions
    GetPermissions() ToolPermissions
    
    // Validate checks if the tool input is valid
    Validate(input ToolInput) error
}

type ToolInput struct {
    TenantID   string                 `json:"tenant_id"`
    Parameters map[string]interface{} `json:"parameters"`
    Context    ExecutionContext       `json:"context"`
}

type ToolOutput struct {
    Result     interface{}   `json:"result"`
    Artifacts  []Artifact    `json:"artifacts"`
    Logs       []LogEntry    `json:"logs"`
    Cost       Cost          `json:"cost"`
    Duration   time.Duration `json:"duration"`
}
```

#### Tool Registry Interface
```go
// pkg/tools/registry.go
type ToolRegistry interface {
    // Register adds a tool to the registry
    Register(tool Tool) error
    
    // Get retrieves a tool by name
    Get(name string) (Tool, error)
    
    // List returns all registered tools
    List() []ToolInfo
    
    // Search finds tools matching criteria
    Search(criteria SearchCriteria) []ToolInfo
}
```

### 4. Memory Interfaces

#### Memory Store Interface
```go
// pkg/memory/store.go
type Store interface {
    // Store saves data to memory
    Store(ctx context.Context, key string, data interface{}) error
    
    // Retrieve gets data from memory
    Retrieve(ctx context.Context, key string) (interface{}, error)
    
    // Search performs semantic search
    Search(ctx context.Context, query string, limit int) ([]SearchResult, error)
    
    // Delete removes data from memory
    Delete(ctx context.Context, key string) error
}

type SearchResult struct {
    Key        string      `json:"key"`
    Data       interface{} `json:"data"`
    Score      float64     `json:"score"`
    Metadata   Metadata    `json:"metadata"`
}
```

### 5. Messaging Interfaces

#### Message Bus Interface
```go
// pkg/messaging/bus.go
type MessageBus interface {
    // Publish sends a message to a subject
    Publish(ctx context.Context, subject string, message Message) error
    
    // Subscribe listens for messages on a subject
    Subscribe(subject string, handler MessageHandler) (Subscription, error)
    
    // Request sends a request and waits for response
    Request(ctx context.Context, subject string, message Message) (Message, error)
    
    // Close closes the message bus connection
    Close() error
}

type Message struct {
    ID        string                 `json:"id"`
    TenantID  string                 `json:"tenant_id"`
    Subject   string                 `json:"subject"`
    Data      []byte                 `json:"data"`
    Headers   map[string]string      `json:"headers"`
    Timestamp time.Time              `json:"timestamp"`
}
```

### 6. Storage Interfaces

#### State Store Interface
```go
// pkg/storage/state.go
type StateStore interface {
    // SaveWorkflow persists workflow state
    SaveWorkflow(ctx context.Context, workflow Workflow) error
    
    // GetWorkflow retrieves workflow by ID
    GetWorkflow(ctx context.Context, tenantID, workflowID string) (Workflow, error)
    
    // SavePlan persists plan state
    SavePlan(ctx context.Context, plan Plan) error
    
    // GetPlan retrieves plan by ID
    GetPlan(ctx context.Context, tenantID, planID string) (Plan, error)
    
    // SaveExecution persists execution state
    SaveExecution(ctx context.Context, execution Execution) error
    
    // GetExecution retrieves execution by ID
    GetExecution(ctx context.Context, tenantID, executionID string) (Execution, error)
}
```

### 7. Configuration Interfaces

#### Config Interface
```go
// internal/config/config.go
type Config interface {
    // Get retrieves a configuration value
    Get(key string) interface{}
    
    // GetString retrieves a string configuration value
    GetString(key string) string
    
    // GetInt retrieves an integer configuration value
    GetInt(key string) int
    
    // GetBool retrieves a boolean configuration value
    GetBool(key string) bool
    
    // Set updates a configuration value
    Set(key string, value interface{}) error
    
    // Validate checks if the configuration is valid
    Validate() error
}
```

### 8. Security Interfaces

#### Authentication Interface
```go
// internal/security/auth.go
type Authenticator interface {
    // Authenticate validates credentials and returns claims
    Authenticate(ctx context.Context, token string) (Claims, error)
    
    // Authorize checks if the user has required permissions
    Authorize(ctx context.Context, claims Claims, resource string, action string) error
}

type Claims struct {
    TenantID    string   `json:"tenant_id"`
    UserID      string   `json:"user_id"`
    Roles       []string `json:"roles"`
    Permissions []string `json:"permissions"`
    ExpiresAt   int64    `json:"exp"`
}
```

## Interface Stability Guarantees

### Compatibility Promise
- **Pre-1.0 (Q1-Q3)**: Interfaces may change with minor version increments
- **Post-1.0**: Semantic versioning compatibility guarantees apply
- **Deprecation Policy**: 2 minor version notice before interface removal

### Breaking Change Process
1. **Proposal**: RFC for interface changes
2. **Review**: Architecture team review and approval
3. **Implementation**: Backward-compatible transition period
4. **Migration**: Automated migration tools where possible
5. **Cleanup**: Remove deprecated interfaces after transition period

### Extension Points
- All interfaces designed for extension through composition
- Plugin architecture supports custom implementations
- Middleware patterns for cross-cutting concerns

## Implementation Status

### Q1 Foundation Interfaces
- [x] Core agent runtime interfaces defined
- [x] Basic planner interfaces established
- [x] Tool execution framework interfaces
- [x] Memory store interfaces
- [x] Message bus abstractions
- [x] State storage interfaces
- [x] Configuration management interfaces
- [x] Security and authentication interfaces

### Q2 Enterprise Interfaces (Planned)
- [ ] Advanced security interfaces (RBAC, audit)
- [ ] Multi-tenant isolation interfaces
- [ ] Cost management interfaces
- [ ] Compliance and governance interfaces

### Q3 Scale Interfaces (Planned)
- [ ] Horizontal scaling interfaces
- [ ] Performance monitoring interfaces
- [ ] Advanced orchestration interfaces
- [ ] Federation and multi-cluster interfaces

## Usage Examples

### Basic Agent Implementation
```go
type MyAgent struct {
    name string
    capabilities AgentCapabilities
}

func (a *MyAgent) Execute(ctx context.Context, input AgentInput) (AgentOutput, error) {
    // Implementation
    return AgentOutput{
        Result: "Hello, World!",
        Context: input.Context,
        TokenUsage: TokenUsage{InputTokens: 10, OutputTokens: 5},
    }, nil
}

func (a *MyAgent) GetCapabilities() AgentCapabilities {
    return a.capabilities
}

func (a *MyAgent) Validate() error {
    return nil
}
```

### Tool Registration
```go
registry := tools.NewRegistry()
myTool := &MyCustomTool{}
err := registry.Register(myTool)
if err != nil {
    log.Fatal(err)
}
```

## Migration Guide

### From Prototype to Production
1. **Interface Adoption**: Implement core interfaces in existing code
2. **Dependency Injection**: Use interface-based dependency injection
3. **Testing**: Mock interfaces for unit testing
4. **Configuration**: Use config interfaces for environment-specific settings

### Version Compatibility
- Check interface version compatibility before upgrades
- Use feature flags for gradual interface adoption
- Monitor deprecation warnings in logs

## References

- [AgentFlow Architecture Baseline ADR](/docs/adr/ADR-0001-architecture-baseline.md)
- [Technical Design Document](/Plan/agentflow_technical_design.md)
- [Go Interface Design Guidelines](https://golang.org/doc/effective_go.html#interfaces)
- [Semantic Versioning](https://semver.org/)

---

**Document Version**: 1.0  
**Last Updated**: 2025-01-16  
**Next Review**: 2025-02-16  
**Maintained By**: AgentFlow Core Team