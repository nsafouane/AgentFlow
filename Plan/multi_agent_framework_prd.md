# AgentFlow — *Production-Ready Multi-Agent Framework*

**A production-ready multi-agent framework with customizable agents, deterministic planning, and enterprise security**

> Production-grade multi-agent orchestration with full observability and deterministic behavior.

---

## Table of Contents

1. [Executive Summary](#executive-summary)
2. [Technical Goals & Scope](#technical-goals--scope)
3. [Product Requirements Document (PRD)](#product-requirements-document-prd)
4. [System Architecture: Control & Data Planes](#system-architecture-control--data-planes)
5. [Agent Model: Customizable & Extensible](#agent-model-customizable--extensible)
6. [Memory Subsystem: Smart & Cost-Aware](#memory-subsystem-smart--cost-aware)
7. [Planning System: Deterministic & Reliable](#planning-system-deterministic--reliable)
8. [Tool Registry: Secure by Default](#tool-registry-secure-by-default)
9. [Messaging & Observability](#messaging--observability)
10. [Developer Experience: Templates-First](#developer-experience-templates-first)
11. [Implementation Examples](#implementation-examples)
12. [Security & Multi-Tenancy](#security--multi-tenancy)
13. [Performance & Scaling Strategy](#performance--scaling-strategy)
14. [Refined Roadmap](#refined-roadmap)

---

## Executive Summary

AgentFlow is a Go-based framework that eliminates the complexity gap between AI prototypes and production systems. AgentFlow delivers **production-ready multi-agent orchestration** with enterprise security, deterministic planning, cost-aware memory management, and one-command deployment.

### Key Features
- **Deterministic Planning**: FSM and Behavior Tree planners for reliable workflows
- **Production Security**: Secure tool execution with clear permission models
- **Cost Tracking**: Token/cost monitoring and budgets
- **Simple Deployment**: Docker containers with Kubernetes support
- **Template-First Development**: Starter templates for common workflows

### MVP Implementation Approach
The following lightweight scaffolds are introduced **before full implementations**:
- **Hash-Chained Audit Log Scaffold** (P2): `audits` table gains `prev_hash` + `hash` columns; hashing library + forward-only append tests.
- **Message Envelope Canonical Hash** (P1–P2): Each bus message serialized deterministically; stored `envelope_hash` enables replay & tamper detection.
- **Sandbox Policy Contract** (P3): Deny-by-default tool permission compiler produces an `ExecProfile` (resources, timeouts) even before gVisor runtime.
- **Cost Estimation v0** (P3–P4): Planner invokes a `PlanCostModel` heuristic using static token/tool cost tables; calibrated later with real observations.
- **Budget Gates Stub** (P3): Budgets accepted & enforced as soft warnings until v1 (hard gate) at P6.8.
- **MCP Adapter Early Slice** (late Q1): Flag-gated minimal handshake + descriptor→ToolSchema mapping + basic permission derivation; full resiliency & parity deferred to Q2 expansion.

These early implementations enable evaluation and testing without waiting for the full security/runtime stack.

### Deliberate Scope Clarification
Previously considered "progressive complexity modes" (Simple / Structured / Recursive) were **removed** to avoid user-facing cognitive overhead. The framework now exposes *one coherent architecture* with optional planners & features enabled via configuration. Documentation will retain a historical appendix (deprecated concepts) for transparency.

Additional high-priority advantages based on 2024–2025 market shifts:
- **On‑Prem Model Serving (Residency First)**: Native support for vLLM, Ollama, and TGI with residency controls to prevent data egress.
- **Cost Preflight & Budgets**: Plan-level token/$ simulation before execution with hard/soft budget gates.
- **Data Minimization Mode**: “Sly‑Data”-style redaction/masking for PII/PHI before LLM/tool calls; optional air‑gapped mode.
- **Tamper‑Evident Audits**: Hash‑chained audit logs for compliance and forensics.
- **Reproducible Benchmarks**: Public harness and published p50/p95 latency and $/task vs competitors.

---

## Technical Goals & Scope

### Primary Goals
- **Production-First**: Build agent systems that handle real production workloads
- **Developer Velocity**: 10x faster time-to-production vs custom solutions  
- **Enterprise Ready**: Security, compliance, and reliability from day one
- **Template Ecosystem**: Rich library of proven workflows

### Non-Goals
- Model training/fine-tuning (focus on orchestration)
- Competing with cloud providers (integrate, don't replace)
- Academic research platform (focus on practical implementation)

---

## Product Requirements Document (PRD)

### Problem Statement
Multi-agent systems show tremendous potential but fail in production due to:
- **Reliability**: Non-deterministic behavior breaks production processes
- **Security**: Unsafe tool execution creates security risks
- **Cost**: Runaway token usage without controls
- **Observability**: Black-box debugging wastes developer time
- **Deployment**: Complex infrastructure requirements block adoption

### Technical Success Metrics

**Performance Goals**:
- Time to first working demo: **< 5 minutes**
- Time to production deployment: **< 1 hour** 
- Mean time to debug root cause: **< 5 minutes**
- System uptime: **99.9%** (hosted)
- Agent response time: **< 100ms** (internal routing)
- Cost estimation accuracy (p50): **≤ 5% variance** between estimated vs actual tokens/$ (baseline calibration begins early, enforced by P6.8)

### Target Users

**Primary: Backend Engineers (3-7 years experience)**
- **Pain**: Complex deployment, poor debugging, integration hell
- **Goal**: Ship AI features without becoming a DevOps expert
- **Success**: Deploy multi-agent system in < 2 hours

**Secondary: AI/ML Engineers**  
- **Pain**: Can't move from prototype to production
- **Goal**: Focus on agent logic, not infrastructure
- **Success**: Experiment with confidence it will scale

**Tertiary: Product Teams**
- **Pain**: Dependent on engineering for simple changes
- **Goal**: Modify workflows without code changes
- **Success**: Configure agent behavior through UI

---

## System Architecture: Control & Data Planes

### High-Level Architecture

```
┌─────────────────────── CONTROL PLANE ───────────────────────┐
│  ┌──────────────────┐  ┌──────────────────┐  ┌─────────────┐ │
│  │   Web Layer      │  │  Orchestrator    │  │  Registry   │ │  
│  │ • REST APIs      │  │ • Workflow Mgmt  │  │ • Tools     │ │
│  │ • WebSocket      │  │ • Plan Execution │  │ • Templates │ │
│  │ • Dashboard UI   │  │ • State Machine  │  │ • Policies  │ │
│  └──────────────────┘  └──────────────────┘  └─────────────┘ │
└──────────────────────────────────────────────────────────────┘
                               │
┌─────────────────────── DATA PLANE ──────────────────────────┐
│  ┌──────────────────┐  ┌──────────────────┐  ┌─────────────┐ │
│  │ Agent Runtimes   │  │  Message Bus     │  │ Tool Exec   │ │
│  │ • Lightweight    │  │ • NATS Stream    │  │ • Sandbox   │ │
│  │ • Auto-scaling   │  │ • Guaranteed     │  │ • gVisor    │ │
│  │ • State Mgmt     │  │ • Tracing        │  │ • Audit Log │ │
│  └──────────────────┘  └──────────────────┘  └─────────────┘ │
└──────────────────────────────────────────────────────────────┘

     ┌─────────────── STORAGE LAYER ────────────────┐
     │ • State Store (Badger/Postgres)              │
     │ • Memory Layer (Vector + SQL + Cache)        │  
     │ • Metrics (Prometheus) • Traces (Jaeger)     │
     └───────────────────────────────────────────────┘
```

### Architectural Principles

**Separation of Concerns**:
- **Control Plane**: Manages workflows, exposes APIs, handles UI
- **Data Plane**: Executes agents, processes messages, runs tools
- **Storage Layer**: Persists state, indexes memory, stores metrics

**Independent Scaling**:
- Control plane scales for API throughput
- Data plane scales for agent workload  
- Storage scales for data volume

**Pluggable Components**:
- Message Bus: NATS (default), Redis Streams, RabbitMQ
- State Store: Badger (dev), Postgres (prod), MongoDB
- Memory: Pinecone, Weaviate, local embeddings + vector DB
- Model Gateway: OpenAI/Anthropic (managed) and on‑prem backends (vLLM, Ollama, TGI) with provider routing and residency policies
 - Cost & Budgeting: `PlanCostModel` & calibration loop (heuristic → observed refinement)
 - Audit Integrity: `AuditHasher` (hash chaining) + optional external anchor
 - Execution Policies: `ExecProfileCompiler` generating sandbox/runtime constraints
- External Tool Protocols: MCP Adapter (early handshake/mapping in Q1, resilience & streaming expansion in Q2)

---

## Agent Model: Customizable & Extensible

### Core Design Principles
- **Agents are units of capability** (role + behavior + resources)
- **Configurable via YAML** (non-technical users) or **Go code** (developers)
- **Reasoning loop**: Sense → Plan → Act → Remember → Reflect

### Go Type System

```go
// Core Agent Configuration
type AgentConfig struct {
    ID            AgentID           `yaml:"id"`
    Type          string           `yaml:"type"`          // "llm", "http", "fsm", "custom"
    Role          string           `yaml:"role"`          // Semantic description
    ModelConfig   ModelConfig      `yaml:"model"`         // Provider + params
    MemoryConfig  MemoryConfig     `yaml:"memory"`        // Storage settings
    PlannerConfig PlannerConfig    `yaml:"planner"`       // Planning strategy
    Tools         []string         `yaml:"tools"`         // Allowed tool IDs
    Policies      PolicyConfig     `yaml:"policies"`      // Rate limits, costs, permissions
    Concurrency   int             `yaml:"concurrency"`   // Max parallel tasks
    RetryPolicy   RetryPolicy     `yaml:"retry"`         // Failure handling
}

// Runtime Agent Instance
type Agent struct {
    Config      AgentConfig
    State       *AgentState
    MsgChan     chan Message
    Memory      MemoryStore
    Planner     Planner
    Tools       map[string]Tool
    Health      HealthChecker
    Runtime     AgentRuntime // Lifecycle hooks
}

// Extensibility Interfaces
type AgentRuntime interface {
    OnStart(ctx context.Context) error
    OnMessage(ctx context.Context, msg Message) error
    OnPlan(ctx context.Context, plan *Plan) error
    OnFinish(ctx context.Context) error
}
```

### Agent Types

**Built-in Types**:
- **LLM Agent**: Configurable model calls with structured I/O
- **HTTP Agent**: REST API interactions with retry/circuit breaker
- **Database Agent**: SQL/NoSQL query execution
- **FSM Agent**: Finite state machine for deterministic behavior
- **HumanTask Agent**: Creates, assigns, and monitors tasks requiring human intervention
- **Custom Agent**: User-defined Go implementations

**Configuration Example**:
```yaml
agents:
  classifier:
    type: llm
    role: "Message classifier for customer support"
    model:
      provider: openai
      model: gpt-4
      temperature: 0.1
    memory:
      short_term:
        enabled: true
        ttl_minutes: 30
    tools: []
    
  responder:
    type: llm  
    role: "Generate customer responses"
    model:
      provider: anthropic
      model: claude-3-sonnet
    planner:
      type: fsm
      transitions: "response_fsm.yaml"
    tools: [send_email, create_ticket]

  approval_manager:
    type: human_task
    role: "Handle workflows requiring human approval"
    config:
      assignment_rules:
        - role: "approval_specialist"
        - backup_role: "supervisor"
      escalation:
        timeout_minutes: 240  # 4 hours
        escalation_chain: ["supervisor", "manager"]
      notification:
        channels: ["email", "slack"]
        reminders: [60, 120]  # minutes
    tools: [send_notification, update_task_status]
```

**HumanTask Agent Details**:
- **Purpose**: Seamlessly integrate human decisions into automated workflows
- **Task Lifecycle**: Created → Assigned → In Progress → Completed/Escalated
- **Assignment Rules**: Role-based routing with backup and escalation policies
- **Notifications**: Multi-channel alerts with configurable reminders
- **Integration**: Works with existing identity providers and team management systems
- **Audit Trail**: Complete history of human interactions for compliance

---

## Memory Subsystem: Smart & Cost-Aware

### Memory Architecture

**Three-Layer Design**:
1. **Short-term (Contextual)**: Per-session scratch space, TTL-based
2. **Long-term (Knowledge)**: Persistent facts, vector-indexed, searchable  
3. **Episodic (Logs)**: Chronological events for replay and auditing

**Cost-Aware Features**:
- **Token budgets** per agent/workflow
- **Retrieval caching** to minimize vector DB calls
- **Intelligent summarization** to compress context
- **Cost tracking** per memory operation
- **Residency & Tenancy Guards** to ensure data stays in approved regions and tenants never cross-contaminate embeddings

### Memory Configuration

```yaml
memory:
  short_term:
    enabled: true
    ttl_minutes: 60
    max_items: 50
    cost_budget_tokens: 1000
    
  long_term:
    enabled: true
    backend: pinecone  
    embedding_model: text-embedding-3-small  # Cost-optimized
    k: 10
    cache_ttl_minutes: 15  # Reduce vector DB calls
    cost_budget_dollars: 10
    
  episodic:
    enabled: true
    retention_days: 30
    compression_ratio: 0.7  # Summarize old episodes
    
  summarizer:
    enabled: true
    cadence: daily
    model: gpt-3.5-turbo  # Cost-effective for summaries
```

### Memory Interfaces

```go
type MemoryStore interface {
    Save(ctx context.Context, agentID AgentID, record MemoryRecord) error
    Query(ctx context.Context, agentID AgentID, opts QueryOptions) ([]MemoryRecord, error)
    Summarize(ctx context.Context, agentID AgentID, opts SummarizeOptions) (string, error)
    
    // Cost management
    GetCosts(ctx context.Context, agentID AgentID, period TimePeriod) (CostReport, error)
    SetBudget(ctx context.Context, agentID AgentID, budget Budget) error
}

type MemoryRecord struct {
    ID        string
    AgentID   AgentID
    Type      MemoryType    // "fact", "conversation", "event", "summary"
    Content   string
    Embedding []float32
    Metadata  map[string]interface{}
    Cost      TokenCost
    CreatedAt time.Time
}
```

### Intelligent Retrieval Pipeline

1. **Cache Check**: Look for recent identical queries
2. **Vector Search**: Query embedding store with budget limits
3. **Reranking**: Score results by relevance and recency
4. **Summarization**: Compress if context window exceeded
5. **Cost Tracking**: Log tokens used for billing/budgets

---

## Planning System: Deterministic & Reliable

### Planner Architecture

**Two-Track Approach**:
- **Deterministic Track**: FSM and Behavior Tree planners for reliability
- **Dynamic Track**: LLM-based planners for creativity and adaptability

**Design Philosophy**: Production systems need **predictable behavior**. LLM planners are powerful but non-deterministic. AgentFlow provides both options with clear trade-offs.

### Planner Types

#### 1. Finite State Machine (FSM) Planner - **Production Recommended**

```yaml
planner:
  type: fsm
  definition: |
    states:
      classify:
        on_success: knowledge_search
        on_error: escalate
      knowledge_search:
        on_success: generate_response
        on_error: fallback_response
        # Conditional Transitions: evaluate tool outputs
        conditional_transitions:
          - condition: "response.confidence < 0.7"
            target: human_review
          - condition: "response.data_sources.length == 0"
            target: escalate
      generate_response:
        on_success: send_response
        on_error: human_review
    initial_state: classify
    final_states: [send_response, escalate]
```

**Benefits**:
- **Deterministic**: Same input always produces same path
- **Debuggable**: Clear state transitions and failure points
- **Reliable**: No LLM dependencies for control flow
- **Fast**: Microsecond transition times
- **Conditional Transitions**: Evaluate tool outputs to determine next state while maintaining plan structure predictability

#### 2. Behavior Tree Planner - **Complex Logic**

```yaml
planner:
  type: behavior_tree
  definition: |
    root:
      type: sequence
      children:
        - type: condition
          check: "message.urgency == 'high'"
        - type: parallel
          children:
            - type: action
              agent: classifier
            - type: action
              agent: escalation_checker
        # Conditional Transitions: dynamic branching based on tool outputs
        - type: conditional_selector
          conditions:
            - expression: "classifier.confidence > 0.8"
              child:
                type: action
                agent: responder
            - expression: "escalation_checker.escalate == true"
              child:
                type: action
                agent: human_task_agent
            - default: true
              child:
                type: action
                agent: fallback_handler
```

**Benefits**:
- **Conditional Transitions**: Nodes can branch based on previous step outputs using expression evaluation
- **Expression Language**: Supports JSONPath and CEL expressions for complex condition evaluation
- **Maintainable**: Clear hierarchical structure while enabling dynamic behavior

#### 3. LLM Planner - **Dynamic Scenarios**

```yaml
planner:
  type: llm
  model: gpt-4
  deterministic_scaffolding: true  # Add structure constraints
  max_steps: 10
  template: |
    Given the goal: {{.goal}}
    Available agents: {{.agents}}
    Current state: {{.state}}
    
    Generate a plan with exactly these fields:
    1. steps: List of sequential actions
    2. assignments: Which agent handles each step
    3. failure_handling: What to do if each step fails
    
    Use this exact JSON format: {{.schema}}

#### Cost Preflight and Fallbacks
- Before executing any plan, run a **preflight cost simulation** (tokens and dollars) per step; reject or warn based on budget policy.
- If LLM planner output fails schema/constraints, **fallback to FSM subplan** for safe execution or request human‑in‑loop.
```

### Planner Interfaces

```go
type Planner interface {
    Plan(ctx context.Context, goal Goal, state *WorkflowState) (*Plan, error)
    Replan(ctx context.Context, failure PlanFailure, state *WorkflowState) (*Plan, error)
    ValidatePlan(ctx context.Context, plan *Plan) error
}

type Plan struct {
    ID          string
    Steps       []PlanStep
    Assignments map[string]AgentID  // step -> agent mapping
    Metadata    PlanMetadata
    CreatedAt   time.Time
    Cost        PlanCost
}

type FSMPlanner struct {
    StateMachine StateMachine
    Transitions  TransitionTable
}

type LLMPlanner struct {
    Model       ModelConfig
    Template    PromptTemplate
    Constraints PlanConstraints  // Deterministic scaffolding
}
```

### Replanning Triggers & Policies

**Automatic Replanning**:
- Agent failure (timeout, error, resource limit)
- Cost budget exceeded
- Quality threshold not met (if evaluators configured)
- External event (webhook, schedule, user interruption)

**Replanning Strategies**:
- **Conservative**: Retry same plan with backoff
- **Adaptive**: Modify plan based on failure context
- **Escalation**: Switch to simpler/more reliable planner
- **Human-in-loop**: Request human intervention

---

## Tool Registry: Secure by Default

### Security-First Architecture

**Zero-Trust Execution**:
- **Default Deny**: No tool can access host or network by default
- **Explicit Permissions**: Each capability must be granted explicitly
- **Sandboxed Execution**: gVisor microVMs for code execution
- **Audit Everything**: Every tool call logged with full context

### Tool Execution Environment

```yaml
tool_security:
  default_sandbox: gvisor        # gvisor, docker, process
  network_policy: deny_all       # allow_all, deny_all, custom
  filesystem_access: read_only   # none, read_only, custom
  resource_limits:
    memory: 512MB
    cpu: 0.5
    timeout: 30s
  audit_level: full              # none, basic, full
```

### Tool Definition Schema

```go
type Tool interface {
    ID() string
    Schema() ToolSchema
    Call(ctx context.Context, input ToolInput) (ToolOutput, error)
    RequiredPermissions() []Permission
}

type ToolSchema struct {
    ID          string                 `json:"id"`
    Name        string                 `json:"name"`
    Description string                 `json:"description"`
    InputSchema map[string]interface{} `json:"input_schema"`
    OutputSchema map[string]interface{} `json:"output_schema"`
    Permissions []Permission           `json:"permissions"`
    Cost        CostModel             `json:"cost"`
    Timeout     time.Duration         `json:"timeout"`
}

// Built-in secure tools
type HTTPTool struct {
    AllowedHosts []string
    MaxBodySize  int64
    Timeout      time.Duration
}

type DatabaseTool struct {
    Connection   string
    ReadOnly     bool
    AllowedOps   []string  // SELECT, INSERT, etc.
}

type CodeExecutorTool struct {
    Language     string    // python, javascript, bash
    Sandbox      SandboxConfig
    Dependencies []string
}
```

### Tool Registry & RBAC

```yaml
tool_registry:
  tools:
    send_email:
      type: smtp
      permissions: [network.smtp]
      cost_per_call: 0.001
      rate_limit: 100/hour
      
    deploy_k8s:
      type: kubernetes
      permissions: [k8s.deploy, k8s.read]
      cost_per_call: 0.1
      rate_limit: 10/hour
      security_level: high
      
  rbac:
    roles:
      support_agent:
        allowed_tools: [send_email, create_ticket]
        cost_budget: 10.0
      devops_agent:  
        allowed_tools: [deploy_k8s, run_tests]
        cost_budget: 100.0
```

### Enterprise Security Features

- **Secrets Management**: Integration with HashiCorp Vault, AWS Secrets Manager
- **Network Policies**: Fine-grained network access controls
- **Resource Quotas**: Per-agent, per-workflow, per-tenant limits
- **Compliance Logging**: SOC2, HIPAA, GDPR-compliant audit trails
- **Threat Detection**: Anomaly detection for unusual tool usage patterns

### Data Minimization & Audit Immutability
- **Minimization Mode**: Redact or mask sensitive fields (PII/PHI, secrets) before persistence or egress to LLMs/tools.
- **Tamper‑Evident Audits**: Hash‑chain audit entries and optionally export to an append‑only store; verify integrity in investigations.
- **Air‑Gapped Operations**: Documented mode disabling external egress and substituting local models and stubs.

---

## Messaging & Observability

### Message Architecture

**NATS JetStream** (Default):
- **Guaranteed Delivery**: Messages persist until acknowledged
- **Distributed Tracing**: Every message carries OpenTelemetry context
- **Replay Capability**: Stream replay for debugging and recovery

```go
type Message struct {
    ID          string                 `json:"id"`
    TraceID     string                 `json:"trace_id"`
    SpanID      string                 `json:"span_id"`
    From        AgentID                `json:"from"`
    To          AgentID                `json:"to"`
    Type        MessageType            `json:"type"`
    Payload     interface{}            `json:"payload"`
    Timestamp   time.Time              `json:"timestamp"`
    Metadata    map[string]interface{} `json:"metadata"`
    Cost        MessageCost            `json:"cost"`
}

type MessageBus interface {
    Publish(ctx context.Context, msg Message) error
    Subscribe(ctx context.Context, agentID AgentID) (<-chan Message, error)
    Replay(ctx context.Context, workflowID string, from time.Time) ([]Message, error)
}
```

### Observability Stack

**Real-Time Dashboard Components**:
1. **Workflow Timeline**: Visual representation of message flow
2. **Agent Status Panel**: Health, load, recent activity per agent
3. **Plan Execution Tree**: Current plan state with progress indicators
4. **Memory Inspector**: Query agent memory stores in real-time
5. **Tool Call Auditor**: Security events and resource usage
6. **Cost Dashboard**: Token usage, API costs, budget alerts

**Time-Travel Debugging**:
- **State Snapshots**: Consistent point-in-time workflow state
- **Message Replay**: Re-execute from any point in history
- **What-If Analysis**: Test different decisions without affecting production
- **Root Cause Analysis**: Automated failure pattern detection
 - **Cost Simulation View**: Compare preflight estimates to actuals over time to tighten guardrails

### OpenTelemetry Integration

```go
// Distributed tracing through entire agent workflow
func (a *Agent) ProcessMessage(ctx context.Context, msg Message) error {
    ctx, span := otel.Tracer("agentflow").Start(ctx, "agent.process_message")
    defer span.End()
    
    // Add agent and message metadata to trace
    span.SetAttributes(
        attribute.String("agent.id", string(a.Config.ID)),
        attribute.String("agent.role", a.Config.Role),
        attribute.String("message.type", string(msg.Type)),
        attribute.Int64("message.cost.tokens", msg.Cost.Tokens),
    )
    
    // Memory access traced
    memories, err := a.Memory.Query(ctx, a.Config.ID, QueryOptions{})
    if err != nil {
        span.RecordError(err)
        return err
    }
    
    // Tool calls traced
    for _, tool := range a.Tools {
        result, err := tool.Call(ctx, msg.Payload)
        // ... handle result
    }
    
    return nil
}
```

---

## Developer Experience: Templates-First

### Onboarding Strategy

**5-Minute Success Path**:
```bash
# 1. Install (single binary)
curl -sSL https://get.agentflow.dev | sh

# 2. Run working demo
af demo support
# → Opens browser showing working customer support system
# → Real chat interface, live agent activity, debug panels

# 3. Customize and deploy
af init my-support --template=support
cd my-support
af config set openai.api_key=sk-xxx
af deploy --provider=aws
# → Production URL in ~3 minutes
```

### Template Library (Battle-Tested)

#### Customer Support Template
```yaml
name: customer-support
description: "Enterprise customer support with escalation"
agents:
  classifier:
    type: llm
    role: "Classify customer messages by urgency and category"
    model: {provider: openai, model: gpt-4, temperature: 0.1}
    planner: {type: fsm, definition: "classify_fsm.yaml"}
    
  knowledge_agent:
    type: rag
    role: "Search knowledge base for relevant solutions"
    memory: {long_term: {backend: pinecone, k: 5}}
    tools: [search_docs, query_faq]
    
  response_agent:
    type: llm
    role: "Generate helpful customer responses"
    planner: {type: behavior_tree, definition: "response_tree.yaml"}
    tools: [send_email, create_ticket, schedule_callback]

triggers:
  - http_endpoint: {path: "/chat", method: POST}
  - email_webhook: {provider: sendgrid}
  - slack_integration: {channel: "#support"}

policies:
  cost_budget: 50.0  # $50/day
  sla_response_time: 60s
  escalation_threshold: 3_failures
```

#### Content Generation Pipeline
```yaml
name: content-pipeline  
description: "Automated blog post creation with SEO optimization"
agents:
  researcher:
    type: web_search
    role: "Research trending topics and gather information"
    tools: [google_search, reddit_scraper, news_api]
    
  writer:
    type: llm
    role: "Create engaging blog content"
    model: {provider: anthropic, model: claude-3-sonnet}
    memory: {short_term: {enabled: true}}
    
  editor:
    type: llm  
    role: "Edit and improve content quality"
    model: {provider: openai, model: gpt-4}
    planner: {type: fsm, definition: "editing_workflow.yaml"}
    
  seo_optimizer:
    type: custom
    role: "Optimize content for search engines"
    tools: [keyword_analysis, meta_generator, readability_check]

workflow:
  type: sequential_with_feedback
  quality_gates: [plagiarism_check, brand_voice_check]
  human_review: optional
```

### CLI Developer Experience

```bash
# Project lifecycle
af init my-project --template=support
af config list                    # Show all configuration options
af config set model.provider=anthropic
af agent add responder --type=llm # Add new agent to workflow

# Development workflow
af dev                            # Hot-reload server + debug UI
af test --scenario=angry_customer # Run test scenarios
af validate                       # Check config and dependencies
af costs --period=today          # Show token usage and costs

# Deployment
af build                         # Create deployment artifacts
af deploy --dry-run              # Preview deployment changes
af deploy --provider=aws         # Deploy to production
af logs --agent=responder        # Stream agent logs
af rollback                      # Quick rollback to previous version
```

### Integration Patterns

#### Existing Go Application
```go
// Add AgentFlow to existing Gin application
func main() {
    // Your existing app
    r := gin.Default()
    r.GET("/users", getUsersHandler)
    
    // Initialize AgentFlow
    agents := agentflow.New(agentflow.Config{
        TemplateDir: "./workflows",
    })
    
    // Load workflow from template
    workflow, err := agents.LoadTemplate("customer-support")
    if err != nil {
        log.Fatal(err)
    }
    
    // Mount under /ai prefix
    r.Group("/ai").Use(agents.GinMiddleware())
    
    // Use in existing handlers
    r.POST("/contact", func(c *gin.Context) {
        // Your existing logic
        contact := saveContact(c)
        
        // Trigger agent workflow asynchronously  
        agents.TriggerAsync("customer-support", map[string]interface{}{
            "message": c.PostForm("message"),
            "email": c.PostForm("email"),
            "priority": inferPriority(contact),
        })
        
        c.JSON(200, gin.H{"status": "received"})
    })
    
    r.Run(":8080")
}
```

#### Microservices Integration
```go
// Dedicated agents service
func main() {
    app := agentflow.New(agentflow.Config{
        Database: "postgresql://agents-db:5432/agents",
        MessageBus: "nats://nats-cluster:4222",
        ServiceRegistry: "consul://consul:8500",
    })
    
    // Register workflows for other services to call
    app.RegisterWorkflow("fraud-detection", loadFraudWorkflow())
    app.RegisterWorkflow("recommendation-engine", loadRecommendationWorkflow())
    app.RegisterWorkflow("content-moderation", loadModerationWorkflow())
    
    // Health checks for Kubernetes
    app.EnableHealthChecks()
    app.EnableMetrics()  // Prometheus endpoints
    
    // Expose both gRPC and HTTP APIs
    go app.RunGRPC(":9090")
    app.RunHTTP(":8080")
}

// Other services call via client
client := agentflow.NewClient("http://agents-service:8080")
result, err := client.ExecuteAsync("fraud-detection", map[string]interface{}{
    "transaction_id": txnID,
    "user_id": userID,
    "amount": amount,
})
```

---

## Implementation Examples

### Example 1: E-commerce Fraud Detection (FSM Planner)

**System Logic**: Analyze transactions through multiple checks with deterministic flow.

```yaml
# fraud-detection.yaml
name: fraud-detection
agents:
  risk_scorer:
    type: ml_model
    role: "Calculate transaction risk score"
    model: {provider: local, endpoint: "http://ml-service:8080/score"}
    planner:
      type: fsm
      definition: |
        states:
          score_transaction:
            on_success: check_velocity
            on_error: flag_for_manual_review
          check_velocity:
            condition: "risk_score < 0.7"
            on_true: approve_transaction
            on_false: additional_checks
          additional_checks:
            on_success: final_decision
          final_decision:
            condition: "combined_score < 0.8"
            on_true: approve_transaction
            on_false: flag_for_manual_review
        initial_state: score_transaction
    
  velocity_checker:
    type: database
    role: "Check user transaction patterns"
    tools: [query_transactions, calculate_velocity]
    
  decision_engine:
    type: rules_engine
    role: "Make final approval decision"
    tools: [approve_transaction, flag_transaction, notify_user]

workflow:
  timeout: 2s  # Real-time requirement
  sla: 99.9%
  cost_budget: 0.01  # $0.01 per transaction
```

### Example 2: Content Moderation (Behavior Tree)

**Processing Logic**: Multi-stage content analysis with parallel processing.

```yaml
# content-moderation.yaml  
name: content-moderation
agents:
  text_analyzer:
    type: llm
    role: "Analyze text content for policy violations"
    model: {provider: openai, model: gpt-4}
    planner:
      type: behavior_tree
      definition: |
        root:
          type: parallel  # Run checks simultaneously
          success_policy: all  # All checks must pass
          children:
            - type: sequence
              name: toxicity_check
              children:
                - type: action
                  agent: toxicity_detector
                - type: condition
                  check: "toxicity_score < 0.3"
            - type: sequence  
              name: spam_check
              children:
                - type: action
                  agent: spam_detector
                - type: condition
                  check: "spam_confidence < 0.5"
            - type: sequence
              name: personal_info_check
              children:
                - type: action
                  agent: pii_detector
                - type: condition
                  check: "pii_found == false"
                  
  image_analyzer:
    type: vision_model
    role: "Analyze images for inappropriate content"
    tools: [nsfw_detector, violence_detector]
    
  decision_maker:
    type: fsm
    role: "Make moderation decision based on all analyses"
    planner:
      type: fsm
      definition: |
        states:
          collect_results:
            on_success: make_decision
          make_decision:
            condition: "all_checks_passed"
            on_true: approve_content
            on_false: determine_action
          determine_action:
            condition: "violation_severity > 0.8"
            on_true: ban_content
            on_false: flag_for_review
        initial_state: collect_results

policies:
  sla_response_time: 500ms
  human_review_threshold: 0.6
  appeals_process: enabled
```

### Example 3: DevOps Deployment Pipeline (LLM + FSM Hybrid)

**Deployment Logic**: Intelligent deployment with rollback capabilities.

```go
// Custom deployment agent combining LLM planning with FSM execution
package main

import (
    "context"
    "github.com/yourorg/agentflow"
    "github.com/yourorg/agentflow/agents"
    "github.com/yourorg/agentflow/planners"
)

func createDeploymentWorkflow() *agentflow.Workflow {
    // LLM Planner for dynamic deployment strategy
    deploymentPlanner := planners.NewLLMPlanner(planners.Config{
        Model: "gpt-4",
        Template: `
        Given the deployment request:
        - Service: {{.service_name}}
        - Environment: {{.environment}}  
        - Changes: {{.git_diff}}
        - Current Traffic: {{.current_load}}
        
        Choose deployment strategy:
        1. blue_green (zero downtime, higher cost)
        2. rolling (gradual, some risk)
        3. canary (safest, slower)
        
        Consider: traffic patterns, change risk, cost constraints
        Output JSON: {"strategy": "...", "rollout_percentage": ..., "monitoring_duration": ...}
        `,
        DeterministicConstraints: planners.Constraints{
            OutputSchema: deploymentSchema,
            ValidStrategies: []string{"blue_green", "rolling", "canary"},
        },
    })
    
    // FSM for execution (deterministic)
    executionFSM := planners.NewFSMPlanner(planners.FSMConfig{
        States: map[string]planners.State{
            "validate_build": {
                OnSuccess: "deploy_to_staging",
                OnError: "notify_failure",
            },
            "deploy_to_staging": {
                OnSuccess: "run_integration_tests",
                OnError: "rollback_staging",
            },
            "run_integration_tests": {
                OnSuccess: "deploy_to_production",
                OnError: "notify_test_failure",
            },
            "deploy_to_production": {
                OnSuccess: "monitor_health",
                OnError: "rollback_production",
            },
            "monitor_health": {
                Timeout: "10m",
                OnSuccess: "deployment_complete",
                OnError: "rollback_production",
            },
        },
        InitialState: "validate_build",
    })

    workflow := agentflow.NewWorkflow("devops-deployment")
    
    // Planning agent (LLM-based)
    workflow.AddAgent(agents.NewLLMAgent(agents.Config{
        ID: "deployment_planner",
        Role: "Analyze deployment requirements and choose strategy",
        Planner: deploymentPlanner,
        Tools: []string{"analyze_git_diff", "check_service_health", "estimate_traffic"},
    }))
    
    // Execution agents (FSM-based for reliability)
    workflow.AddAgent(agents.NewFSMAgent(agents.Config{
        ID: "build_validator",
        Role: "Validate build artifacts and run security scans",
        Planner: executionFSM,
        Tools: []string{"docker_build", "security_scan", "dependency_check"},
    }))
    
    workflow.AddAgent(agents.NewFSMAgent(agents.Config{
        ID: "deployer",
        Role: "Execute deployment strategy",
        Tools: []string{"kubectl_apply", "terraform_apply", "update_load_balancer"},
    }))
    
    workflow.AddAgent(agents.NewFSMAgent(agents.Config{
        ID: "monitor",
        Role: "Monitor deployment health and trigger rollbacks",
        Tools: []string{"check_metrics", "run_health_checks", "rollback_deployment"},
        Memory: memory.NewTimeSeriesStore(memory.Config{
            RetentionDays: 30,
            MetricsBackend: "prometheus",
        }),
    }))
    
    return workflow
}
```

---

## Security & Multi-Tenancy

### Security Architecture

**Defense in Depth**:
1. **Network Layer**: VPC isolation, security groups, firewalls
2. **Application Layer**: JWT auth, RBAC, rate limiting
3. **Agent Layer**: Tool permissions, resource quotas, sandboxing
4. **Data Layer**: Encryption at rest/transit, PII redaction

### Multi-Tenant Isolation

```yaml
# Tenant configuration
tenants:
  tenant_a:
    namespace: "agentflow-tenant-a"
    database_schema: "tenant_a"
    resource_quotas:
      max_agents: 100
      max_workflows: 50
      cost_budget: 1000.0
      storage_gb: 100
    security_policy:
      tool_allowlist: ["http_client", "database", "email"]
      network_policy: "restricted"
      audit_level: "full"
      
  tenant_b:
    namespace: "agentflow-tenant-b" 
    database_schema: "tenant_b"
    resource_quotas:
      max_agents: 500
      max_workflows: 200
      cost_budget: 5000.0
      storage_gb: 500
    security_policy:
      tool_allowlist: ["*"]  # Enterprise tier
      network_policy: "open"
      audit_level: "basic"
```

### Compliance Features

**SOC 2 Type II**:
- Audit logging for all agent actions
- Data encryption and key management
- Access controls and segregation of duties
- Security monitoring and incident response

**GDPR Compliance**:
- Data minimization in memory stores
- Right to erasure (delete agent memories)
- Data portability (export agent data)
- Privacy by design in tool execution

**HIPAA Ready**:
- BAA-compliant hosting options
- PHI detection and redaction
- Encrypted storage and transmission
- Access logging and monitoring

---

## Performance & Scaling Strategy

### Performance Targets

**Control Plane**:
- API Response Time: **< 50ms** (95th percentile)
- Dashboard Load Time: **< 2s** (initial page load)
- Workflow Creation: **< 100ms** (simple workflows)

**Data Plane**:
- Message Routing: **< 10ms** (internal messages)
- Agent Startup: **< 500ms** (cold start)
- Tool Execution: **< 30s** (configurable timeout)
- Memory Query: **< 100ms** (cached), **< 1s** (vector search)

### Scaling Architecture

**Horizontal Scaling**:
```yaml
# Kubernetes deployment
apiVersion: apps/v1
kind: Deployment
metadata:
  name: agentflow-control-plane
spec:
  replicas: 3  # Control plane
  selector:
    matchLabels:
      app: agentflow-control
  template:
    spec:
      containers:
      - name: agentflow
        image: agentflow/server:latest
        env:
        - name: MODE
          value: "control-plane"
        resources:
          requests:
            cpu: 500m
            memory: 1Gi
          limits:
            cpu: 2
            memory: 4Gi

---
apiVersion: apps/v1  
kind: Deployment
metadata:
  name: agentflow-data-plane
spec:
  replicas: 10  # Agent workers - scales based on workload
  selector:
    matchLabels:
      app: agentflow-worker
  template:
    spec:
      containers:
      - name: agentflow-worker
        image: agentflow/worker:latest
        env:
        - name: MODE
          value: "data-plane"
        resources:
          requests:
            cpu: 1
            memory: 2Gi
          limits:
            cpu: 4
            memory: 8Gi
```

**Auto-Scaling Policies**:
- **CPU-based**: Scale when CPU > 70%
- **Queue-based**: Scale when message queue depth > 100
- **Cost-based**: Scale down during low-usage periods
- **Custom metrics**: Scale based on active agent count

### Performance Optimization

**Caching Strategy**:
- **Memory Cache**: Recent agent memories (Redis)
- **Response Cache**: Common LLM responses (TTL-based)
- **Tool Cache**: Expensive tool results (configurable TTL)
- **Plan Cache**: Successful plans for similar scenarios

**Resource Management**:
- **Connection Pooling**: Database and external API connections
- **Circuit Breakers**: Prevent cascade failures from slow services
- **Bulkhead Pattern**: Isolate critical workflows from experimental ones
- **Graceful Degradation**: Fall back to simpler agents when resources constrained

### Benchmarking (Public & Reproducible)
- Provide a versioned benchmark harness covering QA, coding assistant, and RAG flows.
- Publish p50/p95 latencies, throughput, and $/task for common providers and on‑prem backends.
- Track determinism rate for FSM/BT vs guarded LLM planner.

---

## 3-Quarter Development Plan

### Q1: MVP Foundation (Months 1-3) - **"Working Framework"**

**Goal**: Ship a working multi-agent framework that developers can use immediately

**Core Features**:
- ✅ Agent runtime with Go interfaces (AgentRuntime, Planner, Tool, MemoryStore)
- ✅ NATS message bus integration for reliable messaging
- ✅ PostgreSQL state backend for production readiness
- ✅ FSM planner for deterministic workflows
- ✅ Basic LLM and HTTP agents
- ✅ Simple tool registry with Docker sandboxing
- ✅ CLI with `af init`, `af dev`, `af demo`
- ✅ Web dashboard with real-time agent monitoring
- ✅ Cost tracking and basic budget alerts

**Key Deliverable**: **Customer Support Template**
- End-to-end workflow: Classify → Knowledge Search → Response → Action
- 5-minute demo experience (`af demo support`)
- One-command deployment (`af deploy`)

**Success Criteria**:
- Demo runs in < 5 minutes from install
- Production deployment in < 1 hour
- Sub-100ms message routing performance

### Q2: Enterprise Readiness (Months 4-6) - **"Production Ready"**

**Goal**: Make AgentFlow enterprise-ready and expand market reach

**Production Infrastructure**:
- ✅ gVisor sandboxing for secure tool execution
- ✅ OpenTelemetry tracing and Prometheus metrics
- ✅ Vector database integration (Pinecone/Qdrant)
- ✅ JWT authentication and RBAC
- ✅ Audit logging with hash-chain integrity
- ✅ Health checks and automatic recovery
- ✅ Multi-environment deployment support

**Enterprise Features** (Carefully Selected):
- ✅ On-premise model backends (vLLM, Ollama, TGI)
- ✅ Basic multi-tenancy with data isolation
- ✅ Compliance-ready audit trails
- ✅ Cost estimation with 10% accuracy target
- ✅ Template marketplace with versioning

**Human Workflow Integration** (Enhanced):
- ✅ Human task management with escalation
- ✅ Approval workflows with role-based assignment
- ✅ Multi-channel notifications (email, Slack)

**Key Deliverable**: **Three Production Templates**
1. Customer Support (enhanced with knowledge base)
2. Content Generation (research → write → edit → publish)  
3. DevOps Automation (build → test → deploy → monitor)

**Success Criteria**:
- 99.9% uptime for hosted instances
- SOC 2 audit readiness
- First enterprise customer deployment

### Q3: Scale & Advanced Capabilities (Months 7-9) - **"Market Leadership"**

**Goal**: Scale to handle enterprise workloads and advanced use cases

**Advanced Workflow Features**:
- ✅ Behavior tree planner implementation
- ✅ Time-travel debugging and replay capabilities
- ✅ Advanced memory management (summarization, compression)
- ✅ A/B testing for agent configurations

**Developer Experience Enhancements**:
- ✅ VS Code extension for workflow development
- ✅ MCP protocol support for tool ecosystem
- ✅ Advanced CLI features (testing, profiling, optimization)
- ✅ Public benchmark suite with published results

**WASM Runtime Integration** (Q3 Enhancement):
- ✅ Multi-language agent support (Python, Rust, C#, TypeScript)
- ✅ Secure WASM execution environment
- ✅ Performance optimization (<50ms cold start)

**Enterprise Scale Features**:
- ✅ Advanced cost analytics and chargeback
- ✅ Auto-scaling across cloud providers
- ✅ Advanced monitoring and alerting
- ✅ Workflow composition with sub-workflows

**Key Deliverable**: **Self-Service Platform**
- Template library with 20+ proven workflows
- Community-driven template ecosystem
- Visual workflow builder (basic)

**Success Criteria**:
- 1000+ active developer community
- Template usage in 80% of new projects
- $1M ARR pipeline from enterprise prospects

**Key Deliverable**: **Enterprise Cloud Platform**
- Managed AgentFlow service with 99.99% SLA
- Enterprise sales and support organization
- Professional services for custom implementations

**Success Criteria**:
- $1M ARR from enterprise customers
- 99.99% uptime for enterprise tier
- 50+ enterprise customers

---

*Production-ready multi-agent orchestration for the open source community.*