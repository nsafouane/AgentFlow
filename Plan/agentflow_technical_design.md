# AgentFlow — Technical Design Document (TDD)

This technical design translates the PRD into an actionable, production-ready architecture and implementation plan for AgentFlow. It specifies component responsibilities, interfaces, data models, contracts, security controls, deployment topology, and a 3-quarter implementation strategy.

## Implementation Strategy: 3-Quarter Plan

### Q1: MVP Foundation (Months 1-3)
**Focus**: Working framework with core capabilities
- Core interfaces and agent runtime
- NATS messaging with PostgreSQL state
- FSM planner and basic agents (LLM, HTTP)
- Docker tool sandboxing (gVisor in Q2)
- Basic cost tracking and dashboard
- Support template
- MCP protocol adapter early slice (handshake + descriptor mapping + basic permission derivation, flag-gated)

### Q2: Production Readiness (Months 4-6)  
**Focus**: Production security and advanced features
- gVisor sandboxing and zero-trust security
- Advanced observability (OpenTelemetry, Prometheus)
- Human workflow integration and approval flows
- On-premise model backends (vLLM, Ollama)
- Template hub and versioning
- Multi-tenancy foundation
- MCP protocol adapter expansion (resilience, streaming, audit parity, redaction, circuit breakers)

### Q3: Scale & Advanced Capabilities (Months 7-9)
**Focus**: Advanced features and scaling
- WASM runtime for multi-language support
- Behavior trees and workflow composition
- Advanced cost analytics and chargeback
- MCP protocol integration
- Visual workflow builder (basic)
- Event sourcing and distributed transactions

## 1) Scope, Assumptions, Non‑Goals

- Scope
  - Control plane: REST/gRPC APIs, orchestrator, registry, dashboard.
  - Data plane: agent runtimes, message bus integration, sandboxed tool execution.
  - Storage: relational state, vector memory, caches, telemetry backends.
  - Templates, CLI, and core planners (FSM, BT, LLM w/ constraints).
- Assumptions
  - Go 1.22+, Linux containers for prod, dev on macOS/Windows WSL2 ok.
  - NATS JetStream default; pluggable bus via interface.
  - Postgres primary state in prod; Badger/SQLite for dev.
  - Vector backends (Pinecone default) abstracted behind MemoryStore.
  - Zero-trust tool execution via gVisor (preferred) or Docker fallback.
  - On‑prem model serving available via vLLM/Ollama/TGI providers when residency policies require zero egress.
- Non‑Goals
  - Model training/fine-tuning.
  - Proprietary cloud replacement; integrate with existing infra.

## 2) High-Level Architecture and Responsibilities

- Control Plane
  - Web Layer (REST/gRPC, WebSocket) — authentication, tenancy, rate-limits.
  - Orchestrator — workflow lifecycle, planning, assignment, retries, budgets.
  - Registry — tools, templates, policies, planners, models.
  - Dashboard — real-time status, traces, costs, audits.
- Data Plane
  - Agent Runtime — executes plans, message handling, memory ops, tool calls.
  - Message Bus — NATS JetStream with durable streams and replay.
  - Tool Executor — sandboxed execution with explicit permissions and audit.
- Storage Layer
  - Postgres — agents, workflows, plans, policies, audit, tenancy metadata.
  - Vector DB — long‑term memory embeddings and metadata.
  - Cache — Redis for short-term and response/tool caches.
  - Telemetry — Prometheus, OpenTelemetry/Jaeger.
  - Artifact Store — optional append‑only store for tamper‑evident audit exports.

## 3) Core Interfaces (Go)

## 3) Core Interfaces (Go)

Contract notes: all public components must implement these small, testable interfaces. Quarter annotations indicate implementation priority.

### Q1: Core MVP Interfaces
- Messaging
  - type MessageBus interface { Publish(ctx, msg) error; Subscribe(ctx, subject) (<-chan Message, error); Replay(ctx, stream, from) ([]Message, error) }
- Agent Runtime
  - type AgentRuntime interface { OnStart(ctx) error; OnMessage(ctx, Message) error; OnPlan(ctx, *Plan) error; OnFinish(ctx) error }
- Planner
  - type Planner interface { Plan(ctx, Goal, *WorkflowState) (*Plan, error); Replan(ctx, PlanFailure, *WorkflowState) (*Plan, error); ValidatePlan(ctx, *Plan) error }
- Memory
  - type MemoryStore interface { Save(ctx, AgentID, MemoryRecord) error; Query(ctx, AgentID, QueryOptions) ([]MemoryRecord, error); Summarize(ctx, AgentID, SummarizeOptions) (string, error); GetCosts(ctx, AgentID, TimePeriod) (CostReport, error); SetBudget(ctx, AgentID, Budget) error }
- Tool
  - type Tool interface { ID() string; Schema() ToolSchema; Call(ctx context.Context, input ToolInput) (ToolOutput, error); RequiredPermissions() []Permission }
- Cost / Budgets
  - type PlanCostModel interface { Estimate(ctx context.Context, plan *Plan) (CostEstimate, error); Calibrate(ctx context.Context, obs CostObservation) error }

### Q2: Production & Security Interfaces
- Audit Integrity
  - type AuditHasher interface { Chain(previousHash, serializedEnvelope []byte) (newHash []byte, err error) }
- Execution Policy Compilation
  - type ExecProfileCompiler interface { Compile(agent AgentConfig, tool ToolSchema, policies Policies) (ExecProfile, error) }
- Human Intervention
  - type HumanIntervention interface { CreateTask(ctx context.Context, request HumanTaskRequest) (TaskID, error); AwaitResponse(ctx context.Context, taskID TaskID, timeout time.Duration) (HumanTaskResponse, error); Escalate(ctx context.Context, taskID TaskID, escalationLevel int) error }
- Configuration Provider
  - type ConfigurationProvider interface { GetConfig(ctx context.Context, key string) (interface{}, error); SetConfig(ctx context.Context, key string, value interface{}) error; WatchConfig(ctx context.Context, key string) (<-chan ConfigChange, error) }
- Agent State Manager
  - type AgentStateManager interface { Checkpoint(ctx context.Context, agentID string, state AgentState) error; Restore(ctx context.Context, agentID string) (AgentState, error); Hibernate(ctx context.Context, agentID string) error; Migrate(ctx context.Context, agentID string, targetNode string) error }
- Security Management
  - type SecretRotator interface { RotateSecret(ctx context.Context, secretID string) error; GetRotationStatus(ctx context.Context, secretID string) (RotationStatus, error); ScheduleRotation(ctx context.Context, secretID string, schedule string) error }
  - type NetworkPolicyManager interface { CreatePolicy(ctx context.Context, policy NetworkPolicy) error; EnforcePolicy(ctx context.Context, policyID string) error; ValidateConnection(ctx context.Context, source, destination string) (bool, error) }

### Q3: Advanced Scaling Interfaces
- WASM Runtime
  - type WasmAgent interface { AgentRuntime; LoadWasm(ctx, wasmBytes []byte) error; CallWasmFunction(ctx, function string, input []byte) ([]byte, error) }
- Distributed Transactions
  - type SagaOrchestrator interface { BeginSaga(ctx context.Context, sagaID string, definition SagaDefinition) error; AddStep(ctx context.Context, sagaID string, step SagaStep) error; Compensate(ctx context.Context, sagaID string, fromStep int) error; GetSagaStatus(ctx context.Context, sagaID string) (SagaStatus, error) }
- Multi-Dimensional Resource Management
  - type ResourceScheduler interface { AllocateResources(ctx context.Context, agentID string, requirements ResourceRequirements) (*ResourceAllocation, error); ReleaseResources(ctx context.Context, allocationID string) error; GetResourceUsage(ctx context.Context, agentID string) (*ResourceUsage, error); OptimizeAllocations(ctx context.Context, tenantID string) ([]OptimizationRecommendation, error) }
  - type ChargebackCalculator interface { CalculateUsageCosts(ctx context.Context, tenantID string, period TimePeriod) (*ChargebackReport, error); AllocateCosts(ctx context.Context, organizationID string) (*CostAllocation, error); GenerateInvoice(ctx context.Context, tenantID string, period TimePeriod) (*Invoice, error) }
- Agent Development Tools
  - type AgentSimulator interface { CreateSimulation(ctx context.Context, config SimulationConfig) (*Simulation, error); RunSimulation(ctx context.Context, simulationID string) (*SimulationResult, error); GetSimulationMetrics(ctx context.Context, simulationID string) (*SimulationMetrics, error); StopSimulation(ctx context.Context, simulationID string) error }
  - type DevelopmentHarness interface { CreateTestEnvironment(ctx context.Context, config TestEnvironmentConfig) (*TestEnvironment, error); HotReloadAgent(ctx context.Context, agentID string, newCode []byte) error; InjectTestData(ctx context.Context, environmentID string, data TestData) error; GetDevelopmentMetrics(ctx context.Context, environmentID string) (*DevelopmentMetrics, error) }
- Workflow Composition
  - type WorkflowComposer interface { ComposeWorkflows(ctx context.Context, subWorkflows []WorkflowDefinition, composition CompositionPattern) (*ComposedWorkflow, error); CreateExecutionBarrier(ctx context.Context, barrierConfig BarrierConfig) (*ExecutionBarrier, error); ModifyWorkflow(ctx context.Context, workflowID string, modifications []WorkflowModification) error; GetCompositionMetrics(ctx context.Context, workflowID string) (*CompositionMetrics, error) }
- Advanced Observability
  - type DistributedTracer interface { StartTrace(ctx context.Context, operationName string) (context.Context, Span); InjectHeaders(ctx context.Context, carrier map[string]string) error; ExtractHeaders(ctx context.Context, carrier map[string]string) (context.Context, error); RecordAgentInteraction(ctx context.Context, sourceAgent, targetAgent string, messageType string) error; GetTraceContext(ctx context.Context) (*TraceContext, error) }
  - type InteractionVisualizer interface { RecordInteraction(ctx context.Context, interaction *AgentInteraction) error; GenerateInteractionGraph(ctx context.Context, workflowID string, timeRange TimeRange) (*InteractionGraph, error); GetAgentMetrics(ctx context.Context, agentID string, timeRange TimeRange) (*AgentMetrics, error); CreatePerformanceProfile(ctx context.Context, workflowID string) (*PerformanceProfile, error); ExportVisualization(ctx context.Context, graphID string, format string) ([]byte, error) }

## 4) Data Contracts and Storage Design

### 4.1 Relational Schema (Postgres)

- tenants(id pk, name, tier, created_at)
- users(id pk, tenant_id fk, email, role, hashed_secret, created_at)
- agents(id pk, tenant_id fk, type, role, config_json, policies_json, created_at)
- workflows(id pk, tenant_id fk, name, version, config_yaml, planner_type, template_version_constraint varchar, created_at)
- plans(id pk, workflow_id fk, state jsonb, steps jsonb, assignments jsonb, cost jsonb, created_at)
- messages(id pk, tenant_id fk, trace_id, span_id, from_agent, to_agent, type, payload jsonb, metadata jsonb, cost jsonb, ts timestamptz)
- tools(id pk, tenant_id fk, schema jsonb, permissions jsonb, cost_model jsonb, created_at)
- audits(id pk, tenant_id fk, actor_type, actor_id, action, resource_type, resource_id, details jsonb, ts)
- (Early integrity slice P2) audits.prev_hash bytea NULL, audits.hash bytea NOT NULL
- budgets(id pk, tenant_id fk, scope, limit_tokens numeric, limit_dollars numeric, period, created_at)
- human_tasks(id pk, tenant_id fk, workflow_id fk, agent_id fk, title, description, task_data jsonb, assignee_role, assignee_user_id fk, status, priority, due_at timestamptz, created_at, updated_at, completed_at)
- agent_states(id pk, tenant_id fk, agent_id fk, state_data jsonb, checkpoint_type varchar, node_id varchar, created_at timestamptz, metadata jsonb)
- compliance_events(id pk, tenant_id fk, event_type varchar, resource_type varchar, resource_id varchar, actor_id varchar, event_data jsonb, compliance_framework varchar, created_at timestamptz)
- secret_rotations(id pk, tenant_id fk, secret_id varchar, rotation_status varchar, scheduled_at timestamptz, completed_at timestamptz, rotation_metadata jsonb)
- network_policies(id pk, tenant_id fk, policy_name varchar, source_pattern varchar, destination_pattern varchar, action varchar, created_at timestamptz, policy_data jsonb)
- sagas(id pk, tenant_id fk, saga_id varchar, definition jsonb, status varchar, started_at timestamptz, completed_at timestamptz, error_message varchar)
- saga_steps(id pk, saga_id fk, step_order int, step_type varchar, step_data jsonb, status varchar, started_at timestamptz, completed_at timestamptz, compensation_data jsonb)
- event_store(id pk, tenant_id fk, stream_id varchar, event_type varchar, event_data jsonb, event_version int, created_at timestamptz, correlation_id varchar)
- **resource_quotas(id pk, tenant_id fk, resource_type varchar, quota_limit bigint, quota_used bigint, reset_period varchar, created_at timestamptz)**
- **usage_metrics(id pk, tenant_id fk, agent_id fk, resource_type varchar, usage_amount decimal, measurement_timestamp timestamptz, cost_per_unit decimal)**
- **cost_allocations(id pk, tenant_id fk, organization_id fk, period_start date, period_end date, total_cost decimal, breakdown_json jsonb, allocation_rules_json jsonb)**
- **workflow_compositions(id pk, tenant_id fk, parent_workflow_id fk, composition_pattern varchar, sub_workflows jsonb, barrier_config jsonb, created_at timestamptz)**
- **execution_barriers(id pk, workflow_id fk, barrier_type varchar, condition_expression varchar, timeout_seconds int, status varchar, created_at timestamptz)**
- **workflow_dependencies(id pk, source_workflow_id fk, target_workflow_id fk, dependency_type varchar, condition_expression varchar, created_at timestamptz)**
- rbac_roles(id pk, tenant_id fk, name, allowed_tools text[], policies jsonb)
- rbac_bindings(id pk, tenant_id fk, user_id fk, role_id fk)

Indexes: btree on (tenant_id), ts for messages, GIN on jsonb fields (payload, metadata), partial indexes for hot queries.

### 4.2 Message Contract (JSON on bus)

- Message
  - id: string (ULID)
  - trace_id: string
  - span_id: string
  - from: string (agent_id)
  - to: string (agent_id or workflow topic)
  - type: enum [request, response, event, control]
  - payload: object (schema per message type)
  - metadata: object { workflow_id, attempt, planner, … }
  - cost: { tokens:int, dollars:float }
  - ts: RFC3339
  - envelope_hash: string (deterministic canonical serialization; enables replay & tamper-evidence)

Subjects (NATS):
- workflows.<workflow_id>.in
- workflows.<workflow_id>.out
- agents.<agent_id>.in
- agents.<agent_id>.out
- tools.calls
- tools.audit

### 4.3 Memory Records (Vector + SQL)

- memory_records(id, tenant_id, agent_id, type, content, metadata jsonb, embedding vector, cost jsonb, created_at)
- Retrieval pipeline: cache → vector kNN → rerank → summarize(if overflow) → track cost.

## 5) Planners

- FSM Planner (production default)
  - Input: YAML definition (states, transitions, conditions, initial, finals).
  - Engine: deterministic transitions, microsecond execution, pure functions.
  - Validation: DAG checks (no orphan states), timeout guards, terminal guarantees.
  - Failure modes: on_error transitions, exponential backoff with jitter, capped retries.
  - **Conditional Transitions**: Transitions accept optional `condition` field with expression (CEL/JSONPath) evaluated against previous step's output to determine next state while maintaining structural predictability.

- Behavior Tree Planner
  - Nodes: selector | sequence | parallel | condition | action.
  - Success/failure policies; deterministic evaluation order; time-sliced ticks.
  - **Conditional Transitions**: Nodes support conditional branching using `conditional_selector` with expression-based child selection based on tool outputs.

### Conditional Transitions Implementation
- **Expression Language**: Support for CEL (Common Expression Language) and JSONPath for condition evaluation
- **Evaluation Context**: Previous step outputs, agent state, workflow context, and environment variables
- **Type Safety**: Static analysis of expressions with schema validation
- **Performance**: Compiled expressions cached for repeated evaluation
- **Debugging**: Expression evaluation trace included in observability data
- **Examples**:
  - `response.confidence >= 0.8` - numeric comparison
  - `len(results.items) > 0` - array length check
  - `status in ['approved', 'completed']` - membership test
  - `user.role == 'admin' && urgent == true` - complex boolean logic
- LLM Planner
  - Prompt templating with strict JSON schema; temperature ≤ 0.2.
  - Deterministic scaffolding: regex/JSON schema validation, allowed decisions whitelist.
  - Max steps/latency guards, budget awareness, plan caching.
  - Preflight cost simulation per candidate plan; reject or warn per budget policy; fallback to FSM subplan on invalid output.
  - Uses PlanCostModel.Estimate() for budgeting; improved over time by Calibrate() observations emitted by data plane.

## LLM Planner Deterministic Scaffolding

### JSON Schema Validation
- Strict output format enforcement
- Required field validation
- Enum constraints for decision types

### Decision Trees
- Pre-defined decision branches for common scenarios
- Fallback rules when LLM output is invalid
- Human-in-loop escalation triggers

### Prompt Engineering
- Temperature ≤ 0.2 for consistency
- Few-shot examples in prompt templates
- Chain-of-thought reasoning with validation steps

### **Workflow Composition Patterns**

#### **Sub-Workflow Management**
- **Nested Workflows**: Hierarchical workflow composition with parent-child relationships
- **Workflow Libraries**: Reusable workflow components with parameterization
- **Dynamic Composition**: Runtime composition of workflows based on conditions
- **Version Management**: Semantic versioning for workflow components with compatibility checks
- **Dependency Resolution**: Automatic resolution of workflow dependencies and conflicts

#### **Parallel Execution Patterns**
- **Fan-Out/Fan-In**: Split execution into parallel branches with synchronization points
- **Execution Barriers**: Coordinated synchronization across multiple workflow instances
- **Resource Pools**: Shared resource management across parallel workflow executions
- **Load Balancing**: Intelligent distribution of work across available execution resources
- **Failure Isolation**: Isolation of failures in parallel branches with partial success handling

#### **Dynamic Workflow Modification**
- **Runtime Adaptation**: Modify workflow structure during execution based on conditions
- **Hot Swapping**: Replace workflow components without interrupting execution
- **Conditional Branching**: Dynamic path selection based on runtime conditions
- **Loop Constructs**: Dynamic loop generation with break conditions
- **Exception Handling**: Structured exception handling with recovery workflows

#### **Composition Patterns**
- **Pipeline Pattern**: Sequential processing with data transformation between stages
- **Scatter-Gather Pattern**: Parallel processing with result aggregation
- **Saga Pattern**: Long-running workflows with compensation logic
- **State Machine Composition**: Combining multiple state machines into complex workflows
- **Event-Driven Composition**: Workflow composition based on event triggers and reactions

#### **Synchronization Mechanisms**
- **Barrier Synchronization**: Wait for multiple parallel branches to complete
- **Condition Variables**: Block execution until specific conditions are met
- **Semaphore Controls**: Limit concurrent access to shared resources
- **Message Passing**: Coordinate through structured message exchange
- **Consensus Protocols**: Distributed consensus for critical decision points

#### **Quality of Service**
- **Priority Scheduling**: Priority-based execution ordering for workflows
- **Resource Guarantees**: Reserved resource allocation for critical workflows
- **SLA Enforcement**: Service level agreement monitoring and enforcement
- **Graceful Degradation**: Adaptive behavior under resource constraints

## 6) Agent Runtime

- Lifecycle hooks: OnStart → OnMessage → OnPlan → OnFinish (graceful shutdown).
- Concurrency: per-agent worker pool (size=concurrency); mailbox backpressure via NATS durable consumer with max in-flight.
- Policies: retry policy, rate limits, token and dollar budgets; enforced centrally and locally.
- Residency policy tags (tenant/region) propagated to model/memory/tool layers; egress blocked by default when residency is strict.
- Health: liveness/readiness endpoints; heartbeats on bus; circuit breakers for tools.
- Cost calibration: worker emits CostObservation(actual_tokens, model, tool, latency) → PlanCostModel.Calibrate to reduce estimation variance (target p50 ≤5% by Gate G6.8).

### WASM Runtime Integration
- **Purpose**: Enable developers to write agent logic in languages beyond Go (Python, Rust, C#, TypeScript) by compiling to WebAssembly.
- **Runtime Interface**: WASI-compatible environment providing sandboxed execution and secure communication with the Go host runtime.
- **Security Model**: 
  - WASM modules execute in isolated memory with no direct access to host resources
  - All external interactions (network, memory, tools) must go through defined host functions
  - Resource limits (CPU, memory, execution time) enforced at the WASM runtime level
- **Integration Pattern**:
  - `WasmAgent` implements the standard `AgentRuntime` interface
  - WASM modules export standardized entry points: `on_start()`, `on_message()`, `on_plan()`, `on_finish()`
  - JSON-based message passing between host and guest environments
- **Development Workflow**:
  - Developers write agents in their preferred language with AF SDK bindings
  - Compile to WASM using standard toolchains (wasm-pack, TinyGo, etc.)
  - Deploy WASM modules alongside workflow definitions
- **Performance Considerations**:
  - WASM instantiation costs amortized through module pooling
  - Hot-reload support for development environments
  - Benchmark targets: <50ms cold start, <5ms warm function calls

### State Management
- **Purpose**: Ensure agent fault tolerance through persistent state and recovery mechanisms
- **Checkpointing Strategies**:
  - **Periodic Checkpoints**: Automatic state snapshots at configurable intervals (default: 30 seconds)
  - **Event-driven Checkpoints**: State saved after significant events (plan completion, tool execution, message processing)
  - **Manual Checkpoints**: Explicit checkpointing via AgentStateManager interface
- **State Persistence**:
  - Agent working memory, conversation context, and execution state serialized to `agent_states` table
  - Compression and deduplication to minimize storage overhead
  - Encryption at rest for sensitive state data
- **Recovery Procedures**:
  - **Cold Start Recovery**: Agent resumes from last checkpoint with state restoration
  - **Warm-up Procedures**: Gradual re-establishment of connections and context
  - **Cross-restart Recovery**: Agents survive node failures through state migration
- **Hibernation Management**:
  - **Resource Optimization**: Idle agents moved to hibernation state to free resources
  - **Wake-up Triggers**: Message arrival, scheduled tasks, or explicit activation
  - **State Preservation**: Full context maintained during hibernation periods
- **Migration Capabilities**:
  - **Load Balancing**: Agents migrated between nodes for resource distribution
  - **Maintenance Windows**: Graceful agent relocation during node updates
  - **Failure Recovery**: Automatic failover to healthy nodes with state restoration
- **Performance Targets**:
  - Checkpoint operation: <100ms for typical agent state
  - Recovery time: <5 seconds from checkpoint to ready state
  - Migration time: <10 seconds for cross-node transfer

## 7) Tool Registry and Secure Execution

- Default deny; explicit allow-lists per role/agent.
- Sandbox profiles
  - gVisor: no host FS, no raw network unless allow-listed, CPU/mem/timeout limits.
  - Docker fallback: rootless, seccomp/apparmor, read-only FS.
  - Early slice (P3): ExecProfileCompiler generates static execution constraints (timeouts, memory cap, allowed domains) enforced even in process mode.
- Tool schema
  - id, name, description, input_schema, output_schema, permissions[], cost, timeout.
- Permissions model
  - network.* (http, smtp, dns with allow-listed domains)
  - fs.read, fs.tmp, none
  - k8s.*, db.read, db.write (scoped)
- Auditing
  - Log inputs/outputs hashes, duration, resource usage, exit codes; PII-safe redaction rules.
  - Hash‑chain audit records (prev_hash, hash) for tamper evidence; periodic anchor to external store optional.

### Tool Anomaly Detection (Threat Detection)
- Service: ToolAnomalyDetector subscribes to tools.audit and tools.calls streams.
- Signals/features per call: latency, exit code, CPU/mem, bytes in/out, permission set, allowed host, error class, frequency by agent/tenant, hour-of-day.
- Detection rules (initial):
  - **Static Rules** (active from tenant/tool creation):
    - Flag use of sensitive tool combinations (e.g., `db.write` followed by `network.http_post` within 5 minutes)
    - Alert on execution outside of defined operating hours (configurable per tenant)
    - Block calls to non-allow-listed IP ranges or domains (zero-day protection)
    - Detect privilege escalation attempts (tool requesting permissions beyond its defined scope)
    - Flag rapid-fire tool usage patterns that exceed baseline thresholds
    - Monitor for data exfiltration patterns (large outbound transfers after sensitive data access)
  - **Behavioral Baselines** (learned over time):
    - Baseline deviation (z-score over sliding window) on latency/resource usage
    - Rare/first-time host access, unexpected permission elevation
    - High-failure streaks or sudden call-rate spikes per agent/tenant
    - Sensitive tool access outside maintenance window
- Actions:
  - Flag and emit security event (subject: tools.anomaly)
  - Auto-trip circuit breaker for the tool/agent (configurable) with timed reset
  - Escalate to human-in-loop via webhook/alert channel
- Metrics/alerts:
  - tool_anomalies_total{tenant,agent,tool}, tools_auto_blocked_total{tenant,tool}
  - Alerting rules in Prometheus for rate-of-change and absolute thresholds

### MCP Protocol Adapter
- Purpose: Integrate Model Context Protocol tools/servers as first-class Tools.
- Mapping:
  - MCP tool descriptor → ToolSchema (id, input/output schema, timeout)
  - Permissions derivation: network.* scoped to MCP server host(s)
- Sandboxing:
  - Treat MCP calls as network tools; enforce allow-listed domains and timeouts
  - All I/O audited as with native tools
- Conformance:
  - Add MCP adapter tests (handshake, error propagation, schema mapping)
  - Backoff and circuit breaker policies applied to MCP endpoints
  - Auto-scope permissions to MCP server hostnames (deny‑by‑default) and enforce timeouts.

### Template Versioning
- **Strategy**: Templates use Semantic Versioning 2.0.0 (MAJOR.MINOR.PATCH)
- **Binding**: The orchestrator binds workflows to a specific major version using `template_version_constraint` column
- **Compatibility**: Breaking changes require major version increment; backward-compatible features use minor version
- **Migration Process**:
  - **New Major Version Creation**: Templates can publish new major versions with breaking changes
  - **Workflow Migration**: Explicit migration required to move workflows from one major version to another
  - **Parallel Execution**: Multiple major versions can run simultaneously for gradual migration
  - **Deprecation Policy**: Old major versions supported for minimum 6 months after new version release
- **Version Resolution**:
  - `~1.2` allows 1.2.x patch updates but not 1.3.0
  - `1.x` allows any 1.x.x version but not 2.0.0
  - `>=1.5 <2.0` explicit range specification
- **Cache Invalidation**: Plan cache keys include template version to prevent stale plan execution
- **Validation**: Template registry validates semantic version format and prevents version conflicts

## 8) Messaging & Observability

- OpenTelemetry: traces across message publish/consume, memory ops, tool calls.
- Metrics (Prometheus)
  - agent_messages_total, tool_calls_total, planner_replans_total
  - latencies: message_route_seconds, tool_call_seconds, planner_plan_seconds
  - costs: tokens_total, dollars_total by tenant/agent/workflow
  - security: tool_anomalies_total, tools_auto_blocked_total
  - estimates: cost_preflight_tokens_total, cost_preflight_error_ratio
- Logs: structured JSON; correlation via trace/span ids.
- Replay: JetStream consumers with by-time sequence; sandboxed “replay mode” disables external side-effects.
  - Early scaffold (P1–P2): store canonical envelope + envelope_hash; full deterministic re-exec introduced Gate G9.

## 9) APIs (HTTP + gRPC)

Base path: /api/v1

- Auth
  - POST /auth/token — exchange credentials for JWT (tenant-scoped)
- Workflows
  - GET /workflows — list
  - POST /workflows — create/update from YAML
  - POST /workflows/{id}/execute — sync/async trigger
  - GET /workflows/{id}/plans/{plan_id} — plan detail
  - POST /workflows/{id}/replay — time-travel execution
- Budgets & Cost (early stub P3)
  - POST /budgets — create budget (soft enforcement warnings until Gate G6.8)
  - GET /plans/{id}/estimate — return CostEstimate via PlanCostModel
- Agents
  - GET /agents — list, health, load
  - GET /agents/{id} — details
- Tools
  - GET /tools — registry list
  - POST /tools/{id}/test — dry-run in sandbox (no network by default)
- Memory
  - POST /agents/{id}/memory/query — RAG query
  - POST /agents/{id}/memory/erase — GDPR delete
- Observability
  - GET /traces/{trace_id} — distributed trace
  - GET /costs?scope=tenant|workflow|agent&period=today — cost dashboard data
  - GET /benchmarks — latest benchmark runs and SLO deltas (optional, if enabled)

Responses are JSON; long-running operations return job ids with WebSocket updates.

### API Contracts & SDKs
- Contracts:
  - OpenAPI spec published at /api/v1/openapi.json
  - Protobuf/gRPC definitions for core services (ExecuteWorkflow, Tools, Memory)
  - Versioning via Accept header (vnd.agentflow.v1+json) and URL path /v1
- SDKs (generated + thin hand-written wrappers):
  - Python and JavaScript SDKs covering: execute (sync/async), WS stream, memory save/query, costs
  - Auth helpers (JWT), retries, pagination utilities
- Pagination & filtering:
  - Standard: page[size,number] or cursor, filter[field]=value, sort=field
  - Responses include next_cursor and total_count when applicable
- Caching & ETags:
  - Strong ETags on GET list/detail; If-None-Match supported
- Rate limiting & quotas:
  - 429 Too Many Requests with Retry-After and X-RateLimit-* headers
  - Quotas enforced per tenant and per user; burst + sustained windows

## 10) Configuration

### Configuration Provider Architecture
- **Pattern**: Pluggable configuration sources abstracted behind `ConfigurationProvider` interface
- **Hierarchy**: defaults.yml → env vars → tenant overrides → workflow/agent config
- **Initial Implementation**: File and environment variable provider for immediate needs
- **Future Extensions**: Git-based config, dedicated config services, UI-driven configuration
- **Benefits**: 
  - Avoids complex YAML files at scale
  - Enables hot-reload and dynamic configuration
  - Supports multiple config sources simultaneously
  - Maintains backward compatibility

### Configuration Sources
- **FileConfigProvider**: Reads from YAML/JSON files with hierarchical merging
- **EnvironmentConfigProvider**: Environment variable-based configuration
- **TenantConfigProvider**: Per-tenant overrides stored in database
- **GitConfigProvider** (future): Git repository-based configuration management
- **UIConfigProvider** (future): Dashboard-driven configuration interface

### Configuration Management
- Secrets: HashiCorp Vault/AWS Secrets Manager providers.
- Validation: Schema-based validation with type checking and constraint enforcement
- Versioning: Configuration change tracking with rollback capabilities
- Hot-reload: Watch-based configuration updates without service restart

- Example env
  - AF_MODE=control|data
  - AF_DB_DSN=postgres://…
  - AF_NATS_URL=nats://…
  - AF_VECTOR_BACKEND=pinecone|qdrant
  - AF_QDRANT_URL=http://qdrant:6333 (if AF_VECTOR_BACKEND=qdrant)
  - AF_MODEL_BACKENDS=openai,anthropic,vllm,ollama,tgi
  - AF_RESIDENCY_POLICY=strict|lenient|off
  - AF_SANDBOX=gvisor|docker|process
  - AF_API_RATE_LIMIT_BURST=100, AF_API_RATE_LIMIT_SUSTAINED=50/s
  - AF_SECURITY_ANOMALY_ACTION=flag|auto_block|escalate (comma-separated)

- Dev profiles
  - Windows: default to AF_SANDBOX=process with reduced guarantees; WSL2 recommended for gVisor/Docker parity. CLI `af validate` surfaces active sandbox and limitations.

## 11) Deployment Topology

- Containers: agentflow/server (control), agentflow/worker (data).
- Kubernetes
  - Separate deployments; HPA on CPU and queue depth; PodDisruptionBudgets.
  - JetStream cluster with persistence; Postgres HA (Patroni/Cloud managed).
  - NetworkPolicies to isolate workers; admission controls for sandbox pods.
- Scaling
  - Control plane scaled by API throughput; workers by queue depth and active agents.

## Deployment Automation & Infrastructure

### Cloud Provider Integrations
- AWS: CloudFormation/CDK templates for EKS, RDS, ElastiCache
- GCP: Terraform modules for GKE, Cloud SQL, Memorystore  
- Azure: ARM templates for AKS, PostgreSQL, Redis

### Deployment Artifacts
- Helm charts: control plane, workers, NATS JetStream, Postgres, optional Qdrant; values for residency, sandbox, budgets.
- Terraform modules (AWS first): VPC, EKS, RDS Postgres, ElastiCache Redis, GPU node groups for vLLM/TGI, ingress + cert-manager.
- Air‑gapped guide: offline OCI images, template bundles, disable external egress, local model provider configuration (Ollama/TGI).

### CI/CD Pipeline
- GitHub Actions workflows for build/test/deploy
- Multi-environment promotion (dev → staging → prod)
- Automated rollback on health check failures

### 1-Hour Production Target
- Pre-built infrastructure templates
- Automated DNS and TLS certificate provisioning
- Health check and smoke test automation

### One-Command Deployment UX
- UX contract: `af deploy` performs environment discovery, builds artifacts, applies infra, migrates DB, warms caches, and prints endpoints
- Progressive disclosure
  - Defaults: local → managed single-tenant cluster; provider chosen from `af config`
  - Advanced flags: `--provider`, `--region`, `--scale`, `--dry-run`, `--no-confirm`
- State & rollbacks
  - Deployment state recorded per environment; `af rollback` re-applies previous release bundle and database restore point
  - Pre/post hooks for smoke tests and health gates; auto-rollback on failure
- Secrets integration: resolves from Vault/AWS SM/Azure Key Vault via providers; dry-run uses placeholders
- Output: production/control plane URLs, admin credentials provisioning flow, and next-step hints
  - Benchmark URL and last run summary if benchmarking is enabled.

## 12) Collaborative Protocols

### Overview
Standard communication patterns built on top of the basic message bus to enable sophisticated multi-agent collaboration. These protocols provide reusable patterns for common coordination scenarios while maintaining the flexibility of the underlying messaging system.

### Protocol Definitions

#### Contract Net Protocol
**Purpose**: Task bidding and allocation among multiple agents
**Sequence**:
1. **Call for Proposals**: Initiator broadcasts task requirements
2. **Proposal Submission**: Capable agents submit bids with capability/cost information
3. **Winner Selection**: Initiator evaluates proposals and selects winner
4. **Contract Award**: Winner receives task assignment and confirmation
5. **Task Execution**: Winner executes task and reports completion

**Implementation**: Agent mixin providing `initiateBidding()`, `submitProposal()`, `awardContract()` methods
**Message Flow**: 
- `task.bid_request` → `task.proposal` → `task.award` → `task.completion`

#### Request-Confirm Protocol
**Purpose**: Two-phase commit for distributed transactions
**Sequence**:
1. **Prepare Phase**: Coordinator sends prepare request to all participants
2. **Voting Phase**: Participants respond with vote (commit/abort)
3. **Decision Phase**: Coordinator decides based on all votes
4. **Commit Phase**: Coordinator sends final decision to all participants
5. **Acknowledgment**: Participants confirm completion

**Implementation**: `TwoPhaseCoordinator` and `TwoPhaseParticipant` interfaces
**Message Flow**: 
- `txn.prepare` → `txn.vote` → `txn.decision` → `txn.ack`

#### Multi-turn Conversation Protocol
**Purpose**: Structured dialogue between agents with context preservation
**Sequence**:
1. **Conversation Initiation**: Agent starts conversation with topic/goal
2. **Turn Exchange**: Agents exchange messages with conversation context
3. **Context Management**: Each turn includes conversation history summary
4. **Conversation Termination**: Explicit end or timeout-based conclusion

**Implementation**: `ConversationManager` with thread tracking and context compression
**Message Flow**: 
- `conv.start` → `conv.turn` (repeated) → `conv.end`

### Protocol Implementation

#### Agent Mixins
- **ProtocolMixin**: Base interface for protocol participation
- **BiddingMixin**: Implements Contract Net Protocol client/server roles
- **TransactionMixin**: Implements Request-Confirm Protocol coordinator/participant
- **ConversationMixin**: Implements Multi-turn Conversation management

#### Helper Libraries
- **ProtocolRouter**: Routes protocol messages to appropriate handlers
- **StateTracker**: Maintains protocol state across message exchanges
- **TimeoutManager**: Handles protocol timeouts and retry logic
- **ProtocolValidator**: Validates message conformance to protocol specifications

#### Configuration
```yaml
agent:
  protocols:
    - contract_net:
        role: bidder
        evaluation_timeout: 30s
    - two_phase_commit:
        role: participant
        prepare_timeout: 60s
    - conversation:
        max_turns: 50
        context_window: 10
```

### **Advanced Communication Patterns**

#### **Gossip Protocols**
**Purpose**: Decentralized information dissemination and state synchronization
**Implementation**: 
- **Epidemic Spreading**: Random peer selection for message propagation
- **Anti-Entropy**: Periodic full state synchronization between agents
- **Rumor Mongering**: Efficient update propagation with stop conditions
- **Failure Detection**: Heartbeat-based failure detection through gossip

**Message Flow**: 
- `gossip.rumor` → `gossip.ack` → `gossip.anti_entropy`

#### **Consensus Mechanisms**
**Purpose**: Distributed agreement among multiple agents
**Implementations**:
- **Raft Consensus**: Leader election and log replication for critical decisions
- **Byzantine Fault Tolerance**: Consensus despite arbitrary agent failures
- **Proof of Authority**: Authority-based consensus for trusted networks
- **Quorum Voting**: Majority-based decision making

**Message Flow**: 
- `consensus.propose` → `consensus.vote` → `consensus.commit`

#### **Hierarchical Communication**
**Purpose**: Efficient communication in large agent networks
**Patterns**:
- **Tree-Based Messaging**: Hierarchical message propagation
- **Leader-Follower**: Centralized coordination with distributed execution
- **Multi-Level Hierarchies**: Nested agent groups with escalation

**Message Flow**: 
- `hierarchy.broadcast` → `hierarchy.propagate` → `hierarchy.aggregate`

#### **Peer-to-Peer Communication**
**Purpose**: Decentralized agent discovery and direct communication
**Implementations**:
- **Distributed Hash Tables**: Agent discovery and routing
- **Chord Protocol**: Consistent hashing for network organization
- **Content-Based Routing**: Message routing based on content similarity

**Message Flow**: 
- `p2p.discover` → `p2p.route` → `p2p.deliver`

### Network Resilience Features
- **Partition Detection**: Automatic detection of network splits
- **Split-Brain Prevention**: Mechanisms to prevent inconsistent state
- **Graceful Degradation**: Continued operation during partial connectivity
- **Healing Mechanisms**: Automatic recovery after partition resolution
- **State Reconciliation**: Conflict resolution after network healing

### Benefits
- **Reusability**: Standard patterns reduce development time
- **Interoperability**: Agents can collaborate using well-defined protocols
- **Reliability**: Proven patterns with timeout and error handling
- **Observability**: Protocol messages are traced and observable
- **Testing**: Protocol conformance can be unit tested independently

## 13) Compliance and Audit Framework

### Overview
Production-grade compliance and audit capabilities supporting SOC2, GDPR, HIPAA requirements with comprehensive audit trails and data handling controls.

### Compliance Frameworks

#### SOC2 Type II Compliance
- **Security**: Zero-trust architecture with mTLS, secret rotation, network segmentation
- **Availability**: High availability with automated failover and disaster recovery
- **Processing Integrity**: Data validation, integrity checks, and tamper-evident audit logs
- **Confidentiality**: Encryption at rest and in transit, access controls, data minimization
- **Privacy**: GDPR-compliant data handling, consent management, right to erasure

#### GDPR Compliance
- **Data Minimization**: Collect and process only necessary data for legitimate purposes
- **Consent Management**: Explicit consent tracking and withdrawal mechanisms
- **Right to Erasure**: Automated data deletion workflows with verification
- **Data Portability**: Standardized export formats for data subject requests
- **Breach Notification**: Automated detection and reporting within 72-hour requirement

#### HIPAA Compliance
- **PHI Detection**: Automated identification of protected health information
- **Access Controls**: Role-based access with minimum necessary principle
- **Audit Logging**: Comprehensive logging of all PHI access and modifications
- **Encryption**: FIPS 140-2 compliant encryption for PHI at rest and in transit
- **Associate Agreements**: Vendor compliance verification and management

### Audit Trail System

#### Comprehensive Logging
- **Actor Tracking**: User, agent, and system actions with full attribution
- **Resource Access**: All data access patterns with purpose and justification
- **Configuration Changes**: System and security configuration modifications
- **Data Lineage**: Complete data flow tracking across system boundaries
- **Performance Metrics**: SLA compliance and security posture measurements

#### Tamper-Evident Records
- **Hash Chaining**: Cryptographic linking of audit records for integrity verification
- **Immutable Storage**: Write-once, read-many storage with retention policies
- **Digital Signatures**: Cryptographically signed audit entries with timestamps
- **External Notarization**: Optional third-party audit log anchoring
- **Forensic Analysis**: Tools for compliance auditing and incident investigation

### Data Handling Controls

#### Classification and Labeling
- **Sensitivity Levels**: Public, Internal, Confidential, Restricted classifications
- **Automatic Classification**: ML-based data classification with manual override
- **Retention Policies**: Automated data lifecycle management and disposal
- **Geographic Restrictions**: Data residency controls with regional enforcement
- **Cross-Border Transfer**: GDPR Article 46 safeguards for international transfers

#### Privacy Engineering
- **Privacy by Design**: Default privacy-preserving system configurations
- **Data Anonymization**: Statistical disclosure control and k-anonymity
- **Differential Privacy**: Mathematical privacy guarantees for analytics
- **Pseudonymization**: Reversible data de-identification with key management
- **Purpose Limitation**: Strict data usage controls tied to operational justification

### Compliance Monitoring

#### Real-time Monitoring
- **Policy Violations**: Automated detection of compliance policy breaches
- **Anomaly Detection**: Behavioral analysis for unusual access patterns
- **Threshold Alerting**: Configurable alerts for compliance metric deviations
- **Dashboard Views**: Executive and operational compliance status dashboards
- **Regulatory Reporting**: Automated report generation for compliance audits

#### Remediation Workflows
- **Incident Response**: Automated workflows for compliance violations
- **Corrective Actions**: Systematic approaches to compliance gap remediation
- **Preventive Controls**: Proactive measures to prevent future violations
- **Continuous Improvement**: Regular compliance posture assessments and updates
- **Vendor Management**: Third-party compliance verification and monitoring

## 14) Distributed Transactions and Event Sourcing

### Overview
Robust distributed transaction management using saga patterns, event sourcing for audit capabilities, and conflict resolution for concurrent multi-agent operations.

### Saga Pattern Implementation

#### Saga Orchestration
- **Centralized Coordination**: SagaOrchestrator manages complex distributed workflows
- **Step-by-Step Execution**: Sequential execution with rollback capabilities
- **Compensation Logic**: Automatic compensation for failed transactions
- **State Management**: Persistent saga state with recovery mechanisms
- **Timeout Handling**: Configurable timeouts with automatic retry and escalation

#### Saga Types
- **Choreography Saga**: Decentralized coordination through event publication
- **Orchestration Saga**: Centralized coordination with explicit step management
- **Hybrid Saga**: Combination approach for complex multi-tenant scenarios
- **Nested Saga**: Hierarchical sagas for complex processes
- **Parallel Saga**: Concurrent execution with synchronization points

#### Compensation Strategies
- **Semantic Compensation**: Logic-based rollback operations
- **Syntactic Compensation**: Reverse operations (create/delete, increment/decrement)
- **Best-Effort Compensation**: Partial rollback with manual intervention options
- **Timeout-Based Compensation**: Automatic compensation after timeout periods
- **External System Compensation**: Integration with external system rollback APIs

### Event Sourcing System

#### Event Store Design
- **Immutable Events**: Write-once event storage with complete audit trail
- **Stream Organization**: Events organized by aggregate ID and version
- **Snapshotting**: Periodic snapshots for performance optimization
- **Event Replay**: Complete system state reconstruction from events
- **Concurrent Access**: Optimistic concurrency control with version checking

#### Event Types
- **Domain Events**: Significant events (agent actions, workflow state changes)
- **Integration Events**: Cross-boundary events for system coordination
- **Infrastructure Events**: System-level events (startup, shutdown, errors)
- **Compensation Events**: Events triggered during saga compensation
- **Snapshot Events**: Periodic state snapshots for optimization

#### Event Processing
- **Event Handlers**: Idempotent event processing with duplicate detection
- **Event Projections**: Read-model generation from event streams
- **Event Correlation**: Causal relationship tracking across events
- **Event Archival**: Long-term event storage with compliance requirements
- **Event Analytics**: Intelligence and audit analytics from event data

### Conflict Resolution

#### Concurrency Control
- **Optimistic Locking**: Version-based conflict detection with retry mechanisms
- **Pessimistic Locking**: Resource locking for critical sections
- **MVCC (Multi-Version Concurrency Control)**: Multiple versions with timestamp ordering
- **CRDTs (Conflict-free Replicated Data Types)**: Mathematically sound conflict resolution
- **Last-Writer-Wins**: Simple conflict resolution with timestamp-based ordering

#### Consistency Models
- **Strong Consistency**: ACID properties for critical operations
- **Eventual Consistency**: AP (Availability-Partition tolerance) for scalability
- **Causal Consistency**: Ordering preservation for causally related operations
- **Session Consistency**: Consistency within user sessions
- **Monotonic Consistency**: Monotonic read and write consistency guarantees

#### Conflict Resolution Strategies
- **Application Rules**: Domain-specific conflict resolution logic
- **User Intervention**: Manual resolution for complex conflicts
- **Automated Merging**: Algorithmic conflict resolution with heuristics
- **Priority-Based**: Role or priority-based conflict resolution
- **Temporal Ordering**: Time-based conflict resolution with NTP synchronization

### Integration with Messaging

#### Event-Driven Architecture
- **Event Publication**: Automatic event publication to message bus
- **Event Subscription**: Durable subscriptions for event processing
- **Event Ordering**: Guaranteed ordering within event streams
- **Event Deduplication**: Duplicate event detection and handling
- **Event Transformation**: Event format transformation for different consumers

#### Message Replay Integration
- **Deterministic Replay**: Consistent event ordering during replay
- **Side-Effect Isolation**: Event sourcing without external side effects
- **Snapshot Integration**: Efficient replay from snapshots
- **State Verification**: Event replay validation against known states
- **Performance Optimization**: Optimized replay for large event volumes

## 15) Performance Targets and Optimizations

- SLOs
  - Control API p95 < 50ms; message routing p95 < 10ms; agent startup < 500ms; memory query cached < 100ms / vector < 1s.
- Optimizations
  - Connection pooling, gRPC keepalive, zero-copy message marshaling where possible.
  - Redis caches: memory, response, tool, plan caches with per-tenant TTLs.
  - Circuit breakers and bulkheads per workflow; graceful degradation paths.

  ### KPI Measurement & Reporting (PRD Metrics)
  - Instrumentation
    - TTFD: timer from `af install/init` to first successful `af demo` run; exported as `dx_ttfd_seconds`
    - TTPD: timer from `af deploy` start to healthy endpoints ready; `deploy_ttpd_seconds`
    - MTTR (root cause): dashboard action + replay duration measured via trace annotations; `debug_mttr_seconds`
    - Uptime: SLOs evaluated via blackbox probes + service health metrics; `service_slo_violations_total`
    - Routing latency: `message_route_seconds` histogram with tenant/workflow labels
  - Cost Preflight: `cost_preflight_error_ratio` comparing estimate vs actual by model/provider
  - Surfacing
    - Dashboard tiles for PRD KPIs with targets; per-tenant and global views
    - Webhook/export to external observability (Datadog/Grafana Cloud)
    - Error budgets tracked and displayed with burn rates

## 16) Advanced Debugging and Visualization

### Overview
Sophisticated debugging capabilities for complex multi-agent interactions including agent interaction graphs, performance profiling, and real-time cost attribution.

### Agent Interaction Visualization

#### Interaction Graph Generation
- **Real-time Graph Construction**: Dynamic graph building from agent communication patterns
- **Multi-dimensional Visualization**: Node sizing by throughput, edge coloring by latency
- **Temporal Analysis**: Time-series visualization of interaction patterns
- **Cluster Detection**: Automatic identification of agent collaboration clusters
- **Bottleneck Identification**: Visual highlighting of performance bottlenecks

#### Performance Profiling
- **Agent-Level Profiling**: CPU, memory, and network usage per agent
- **Workflow Profiling**: End-to-end performance analysis of complete workflows
- **Tool Execution Profiling**: Detailed analysis of tool call performance
- **Resource Contention Analysis**: Identification of resource conflicts between agents
- **Scalability Analysis**: Performance characteristics under different load conditions

#### Cost Attribution
- **Real-time Cost Tracking**: Live cost accumulation across distributed transactions
- **Cost Breakdown Analysis**: Detailed cost attribution by agent, tool, and resource type
- **Cost Prediction Modeling**: Predictive cost analysis based on historical patterns
- **Budget Variance Analysis**: Real-time comparison of actual vs. budgeted costs
- **Cost Optimization Recommendations**: AI-driven suggestions for cost reduction

### Distributed Debugging Tools

#### Causal Event Analysis
- **Event Timeline Reconstruction**: Chronological ordering of causally related events
- **Causality Path Visualization**: Visual representation of event dependency chains
- **Inconsistency Detection**: Identification of causal ordering violations
- **Parallel Event Analysis**: Understanding of concurrent event streams
- **Event Correlation Debugging**: Troubleshooting of event correlation issues

#### Transaction Flow Analysis
- **Saga Step Visualization**: Visual representation of saga execution progress
- **Compensation Flow Tracking**: Visualization of rollback and compensation operations
- **State Transition Analysis**: Detailed analysis of agent state changes
- **Lock Contention Analysis**: Identification and visualization of resource locks
- **Deadlock Detection**: Automatic detection and visualization of deadlock scenarios

#### Multi-Agent Debugging
- **Distributed Breakpoints**: Coordinated debugging across multiple agents
- **State Synchronization Debugging**: Analysis of distributed state consistency
- **Message Flow Debugging**: Detailed analysis of inter-agent message flows
- **Protocol Compliance Checking**: Validation of communication protocol adherence
- **Agent Behavior Analysis**: Pattern recognition in agent decision-making

### Debugging Infrastructure

#### Debug Data Collection
- **Comprehensive Logging**: Structured logging with correlation IDs and causality information
- **Metric Collection**: High-resolution metrics for performance and behavior analysis
- **Event Capture**: Complete event capture with context preservation
- **State Snapshots**: Periodic and on-demand state snapshots for analysis
- **Communication Interception**: Non-intrusive monitoring of agent communications

#### Analysis Tools
- **Interactive Query Interface**: SQL-like interface for debugging data exploration
- **Pattern Recognition**: Machine learning-based pattern detection in debugging data
- **Anomaly Detection**: Automatic identification of unusual system behavior
- **Root Cause Analysis**: Automated suggestions for issue root causes
- **Impact Analysis**: Assessment of issue impact across the system

#### Visualization Dashboard
- **Real-time Dashboards**: Live system state and performance visualization
- **Historical Analysis**: Time-series analysis of system behavior
- **Comparative Analysis**: Side-by-side comparison of different time periods
- **Alert Visualization**: Visual representation of system alerts and issues
- **Custom Dashboards**: User-configurable dashboards for specific debugging needs

## 17) Multi-Dimensional Resource Management

### Overview
Sophisticated resource accounting and optimization beyond simple token counting, including CPU, memory, network, and storage with dynamic pricing models and cost optimization recommendations.

### Resource Types and Metrics

#### Computational Resources
- **CPU Utilization**: Agent processor usage measured in CPU seconds and percentage
- **Memory Consumption**: Agent memory usage including heap, stack, and WASM runtime memory
- **GPU Acceleration**: GPU usage for ML-intensive agents with compute unit tracking
- **Parallel Execution Slots**: Concurrent agent execution capacity and scheduling
- **WASM Runtime Overhead**: Isolation and sandboxing resource costs

#### Storage and Data Resources
- **Database Operations**: Read/write operations with query complexity metrics
- **Vector Database Usage**: Embedding storage and similarity search operations
- **Event Store Capacity**: Event sourcing storage with retention policies
- **Cache Utilization**: Memory cache usage and hit/miss ratios
- **Backup and Archive**: Long-term storage costs for compliance and recovery

#### Network and Communication Resources
- **Message Bus Throughput**: Message volume and bandwidth consumption
- **External API Calls**: Third-party service usage and rate limiting
- **Inter-Agent Communication**: Network overhead for distributed coordination
- **Content Delivery**: Static asset and template distribution costs
- **Bandwidth Allocation**: Network capacity reservation and usage

#### Specialized Resources
- **AI Model Inference**: Token consumption across different model providers
- **Tool Execution Time**: External tool usage duration and complexity
- **Human Task Duration**: Human-in-the-loop task completion time
- **Compliance Processing**: Audit trail generation and verification costs
- **Security Scanning**: Real-time security analysis and threat detection

### Dynamic Pricing Models

#### Time-Based Pricing
- **Peak/Off-Peak Rates**: Dynamic pricing based on system load and demand
- **Seasonal Adjustments**: Long-term pricing variations based on usage patterns
- **Real-Time Optimization**: Automatic workload shifting to reduce costs
- **Spot Pricing**: Opportunistic resource usage at reduced rates
- **Reserved Capacity**: Pre-purchased resource allocations with discounts

#### Usage-Based Pricing
- **Tiered Pricing**: Volume discounts for high-usage scenarios
- **Burst Pricing**: Premium rates for temporary resource spikes
- **Quality of Service Tiers**: Different pricing for performance guarantees
- **Geographic Pricing**: Location-based pricing for data residency requirements
- **Multi-Tenant Discounts**: Shared resource efficiency pricing

#### Value-Based Pricing
- **Outcome-Based Cost**: Cost tied to value delivered
- **Performance-Based Adjustments**: Pricing modifiers based on SLA achievement
- **Success Fee Models**: Additional charges for high-value outcomes
- **Risk-Adjusted Pricing**: Cost modifications based on failure probability
- **Innovation Credits**: Discounts for experimental or research workloads

### Resource Scheduling and Optimization

#### Intelligent Scheduling
- **Load Balancing**: Optimal resource distribution across available capacity
- **Priority Queuing**: Resource allocation based on application priorities
- **Deadline-Aware Scheduling**: Time-sensitive workload prioritization
- **Resource Affinity**: Optimal placement based on data locality and dependencies
- **Preemptive Scheduling**: Higher priority workload resource reallocation

#### Optimization Algorithms
- **Bin Packing**: Efficient resource utilization through optimal placement
- **Genetic Algorithms**: Long-term optimization of resource allocation patterns
- **Machine Learning Optimization**: Predictive resource allocation based on historical patterns
- **Multi-Objective Optimization**: Balancing cost, performance, and reliability
- **Real-Time Adjustment**: Dynamic reallocation based on changing conditions

#### Cost Optimization Strategies
- **Resource Right-Sizing**: Automatic adjustment of allocated resources to actual usage
- **Idle Resource Detection**: Identification and deallocation of unused resources
- **Workload Consolidation**: Combining compatible workloads for efficiency
- **Temporal Load Shifting**: Moving non-urgent workloads to off-peak periods
- **Provider Optimization**: Selecting optimal cloud providers and regions

### Chargeback and Showback

#### Cost Attribution
- **Granular Tracking**: Per-agent, per-workflow, and per-tenant cost allocation
- **Shared Resource Allocation**: Fair distribution of shared infrastructure costs
- **Overhead Distribution**: System overhead allocation based on usage patterns
- **Cross-Subsidization**: Cost sharing mechanisms for organizational efficiency
- **Activity-Based Costing**: Cost allocation based on actual resource consumption

#### Financial Reporting
- **Real-Time Dashboards**: Live cost monitoring and budget tracking
- **Periodic Reports**: Monthly, quarterly, and annual cost summaries
- **Variance Analysis**: Comparison of actual vs. budgeted costs
- **Trend Analysis**: Historical cost patterns and forecasting
- **ROI Calculations**: Return on investment analysis for agent deployments

#### Budget Management
- **Multi-Level Budgets**: Organization, department, and project-level budgets
- **Soft and Hard Limits**: Warnings and enforcement for budget overruns
- **Approval Workflows**: Budget approval processes for cost overruns
- **Forecasting**: Predictive budget planning based on usage trends
- **Cost Alerts**: Automated notifications for budget thresholds

### Integration with Existing Systems

#### Production Integration
- **ERP System Integration**: Connection with existing financial systems
- **LDAP/Active Directory**: User and organization hierarchy integration
- **Cloud Provider APIs**: Direct integration with AWS, Azure, GCP billing
- **Third-Party Tools**: Integration with existing monitoring and billing tools
- **Custom Reporting**: Flexible export formats for external analysis

#### API and Data Export
- **RESTful APIs**: Programmatic access to cost and usage data
- **Real-Time Streaming**: Live cost data streams for external systems
- **Bulk Export**: Historical data export for analysis and archival
- **Standard Formats**: Support for common financial data formats
- **Audit Trails**: Complete tracking of all cost-related modifications

## 18) Agent Development Tools and Experience

### Overview
Sophisticated development tools for creating, testing, and debugging agents including simulation environments, behavior visualization, hot reloading, and IDE integration.

### Agent Simulation Environment

#### Simulation Framework
- **Virtual Agent Networks**: Create isolated simulation environments with configurable network topologies
- **Synthetic Workload Generation**: Generate realistic agent workloads for testing and optimization
- **Time Acceleration**: Fast-forward simulation for long-term behavior analysis
- **Deterministic Replay**: Reproducible simulations with controlled randomness
- **Multi-Scale Simulation**: Support for single agent to large-scale multi-agent simulations

#### Environment Modeling
- **Resource Constraints**: Simulate realistic CPU, memory, and network limitations
- **Failure Injection**: Inject controlled failures for resilience testing
- **External Service Mocking**: Mock external APIs and services with configurable responses
- **Network Conditions**: Simulate various network conditions including latency and partitions
- **Load Patterns**: Generate realistic load patterns based on historical data

#### Simulation Analytics
- **Performance Metrics**: Comprehensive performance analysis during simulation
- **Behavior Analysis**: Pattern recognition in agent decision-making processes
- **Resource Utilization**: Detailed resource usage tracking and optimization suggestions
- **Cost Projection**: Extrapolate simulation results to real-world cost estimates
- **Scalability Testing**: Automated scaling analysis with bottleneck identification

### Behavior Visualization and Analysis

#### Real-Time Visualization
- **Agent State Visualization**: Live visualization of agent internal states and transitions
- **Communication Flow Diagrams**: Real-time visualization of inter-agent communication
- **Decision Tree Exploration**: Interactive exploration of agent decision processes
- **Performance Heatmaps**: Visual representation of performance bottlenecks
- **Resource Usage Graphs**: Dynamic visualization of resource consumption patterns

#### Behavior Analysis Tools
- **Pattern Recognition**: Automatic identification of common behavior patterns
- **Anomaly Detection**: Identification of unusual or unexpected agent behaviors
- **Performance Profiling**: Detailed analysis of agent execution performance
- **Comparative Analysis**: Side-by-side comparison of different agent implementations
- **Regression Detection**: Automatic detection of behavior regressions

#### Interactive Debugging
- **Step-by-Step Execution**: Manual stepping through agent execution for detailed analysis
- **Breakpoint Management**: Set conditional breakpoints based on agent state or behavior
- **Variable Inspection**: Real-time inspection of agent variables and internal state
- **Call Stack Analysis**: Detailed analysis of agent function call hierarchies
- **Timeline Navigation**: Navigate through agent execution timeline with replay capability

### Hot Reloading and Live Development

#### Code Hot Reloading
- **WASM Module Replacement**: Live replacement of WASM modules without restart
- **Configuration Updates**: Real-time configuration changes without service interruption
- **Template Modification**: Live updates to workflow templates and agent definitions
- **Tool Integration**: Hot reloading of tool implementations and configurations
- **Dependency Management**: Automatic dependency resolution during hot reloading

#### Development Workflow
- **Incremental Compilation**: Fast incremental compilation for rapid development cycles
- **Automated Testing**: Automatic test execution on code changes
- **Live Preview**: Real-time preview of agent behavior changes
- **Error Highlighting**: Immediate feedback on compilation and runtime errors
- **Performance Impact**: Real-time performance impact analysis of code changes

#### Safety Mechanisms
- **Rollback Capability**: Automatic rollback on failed hot reload attempts
- **State Preservation**: Maintain agent state during code updates
- **Graceful Degradation**: Fallback to previous version on critical errors
- **Transaction Safety**: Ensure transactional integrity during hot reloading
- **Validation Gates**: Pre-deployment validation to prevent problematic updates

### IDE Integration and Tooling

#### Code Editor Integration
- **Syntax Highlighting**: Advanced syntax highlighting for agent configuration files
- **IntelliSense**: Auto-completion for agent APIs and configuration options
- **Error Detection**: Real-time error detection and correction suggestions
- **Refactoring Tools**: Automated refactoring for agent code and configurations
- **Code Navigation**: Advanced navigation features for complex agent hierarchies

#### Debugging Integration
- **Integrated Debugger**: Full-featured debugger integrated with popular IDEs
- **Remote Debugging**: Debug agents running on remote clusters
- **Multi-Agent Debugging**: Coordinated debugging across multiple agents
- **Performance Profiler**: Integrated performance profiling tools
- **Log Integration**: Seamless integration with centralized logging systems

#### Project Management
- **Template Scaffolding**: Automated project setup with best practices
- **Dependency Management**: Visual dependency management and conflict resolution
- **Version Control Integration**: Advanced Git integration with agent-specific features
- **Deployment Pipeline**: Integrated CI/CD pipeline management
- **Documentation Generation**: Automatic documentation generation from code and configurations

### Testing and Quality Assurance

#### Automated Testing Framework
- **Unit Testing**: Comprehensive unit testing framework for individual agents
- **Integration Testing**: Multi-agent integration testing with realistic scenarios
- **Property-Based Testing**: Generative testing for complex agent behaviors
- **Regression Testing**: Automated regression testing for behavior changes
- **Performance Testing**: Automated performance testing with benchmark comparisons

#### Test Data Management
- **Synthetic Data Generation**: Generate realistic test data for various scenarios
- **Test Environment Provisioning**: Automated provisioning of isolated test environments
- **Data Anonymization**: Automatic anonymization of production data for testing
- **Test Case Generation**: AI-powered generation of comprehensive test cases
- **Coverage Analysis**: Detailed code coverage analysis with visualization

#### Quality Metrics
- **Code Quality Analysis**: Static analysis for code quality and best practices
- **Behavior Consistency**: Analysis of agent behavior consistency across scenarios
- **Performance Benchmarking**: Automated performance benchmarking and comparison
- **Security Scanning**: Automated security vulnerability scanning
- **Compliance Validation**: Validation against production compliance requirements

### Developer Experience Enhancements

#### Documentation and Learning
- **Interactive Tutorials**: Step-by-step interactive tutorials for agent development
- **API Documentation**: Comprehensive, searchable API documentation with examples
- **Best Practices Guide**: Curated best practices and design patterns
- **Troubleshooting Assistant**: AI-powered troubleshooting and problem resolution
- **Community Integration**: Integration with developer community and knowledge sharing

#### Productivity Tools
- **Code Generation**: AI-powered code generation for common patterns
- **Template Library**: Extensive library of agent templates and patterns
- **Snippet Manager**: Reusable code snippets with intelligent insertion
- **Workflow Automation**: Automated common development workflows
- **Performance Optimization**: Automated suggestions for performance improvements

#### Collaboration Features
- **Shared Development Environments**: Collaborative development with real-time sharing
- **Code Review Integration**: Streamlined code review process with agent-specific checks
- **Knowledge Sharing**: Integrated knowledge base and expert system
- **Mentoring Tools**: Built-in mentoring and learning assistance
- **Team Analytics**: Team productivity analytics and insights

## 19) Developer Experience (CLI + Templates)

- CLI commands (non-exhaustive): init, dev (hot-reload), test (scenarios), validate, costs, build, deploy, logs, rollback.
- Templates: support, content-pipeline, devops-automation.
- VS Code extension: schema validation, snippets, trace viewer (future).

### Template Engine & Hub
- Template manifest (template.yaml)
  - name, description, version (semver), tags[], authors, license
  - compatibility: af_version range, required planners/tools, minimal infra
  - assets: workflow YAMLs, agent configs, policies, seed data, dashboard presets
  - parameters: typed inputs with defaults and validation (string|number|enum|secret)
- Packaging
  - OCI artifact or tarball with signed provenance (Sigstore/Cosign)
  - Integrity hash recorded in registry; optional vendor signature trust policy
- Registry (af hub)
  - API: list/search/get/signature, rate limits, per-tenant curated catalogs
  - Import/export: `af template push/pull`, offline bundles for air-gapped installs
  - Versioning: semver with deprecations; upgrade hooks and plan-cache bust rules
- Installation flow
  - `af init my-app --template=<name>@<version> -p key=value` → parameterized render
  - Post-install checks: `af validate` runs health, dependency, and policy checks
  - Uninstall: reversible ops; leaves audit trail

### Developer Onboarding Implementation
- Hot-reload (`af dev`)
  - File watchers on workflow/agent/template files; reload planners and agents without process restart when safe; fall back to process restart when needed
  - Ephemeral data plane: in-memory/SQLite state, NATS JetStream file mode, stubbed secrets provider
  - Port-forwarded dashboard with live traces; auto-open browser
- Demo provisioning (`af demo <template>`)
  - Seeds sample data and mock integrations; disables external side-effects by default
  - Deterministic clock/ULID seed to make golden traces reproducible
- Workflow testing
  - Scenario DSL: .af/scenarios/*.yaml defining inputs, expected states, tool stubs
  - Golden trace assertions and snapshot tests; CI target `af test --ci`
  - k6 performance scenarios generated from traces for quick smoke/load tests
- IDE integration (VS Code ext)
  - Schema validation and quick-fixes; FSM/BT graph preview; inline trace viewer
  - “Run Scenario” code lens for scenarios and workflows

### Dashboard & Visual Builder
- Decision: React + tRPC for dashboard and visual workflow builder.
- Initial features: FSM graph and BT tree visualization, step timings, cost overlays, live plan progress, tool audit panel.
- Accessibility: keyboard navigation and high-contrast theme; multi-tenant theming.
- Lightweight ops UI for air-gapped installs may use HTMX fallback.

### SDKs
- Ship Python, JavaScript, and .NET SDKs (matching API Contracts) with examples and end-to-end tests.
- Provide codegen targets (OpenAPI + protobuf) and publish to package registries.

### Windows Dev Ergonomics
- Document sandbox modes: process (default on Windows), Docker rootless (WSL2), gVisor (WSL2).
- `af validate` warns when capabilities differ from production (e.g., no network isolation in process mode) and suggests WSL2 setup.

## 20) Testing & Validation Strategy

### Unit Testing Framework
- **Coverage Requirements**: 80%+ code coverage with focus on critical paths
- **Test Organization**: Domain-driven test structure mirroring production code
- **Mocking Strategy**: Interface-based mocking for external dependencies
- **Property-Based Testing**: Generative testing for complex business logic
- **Mutation Testing**: Code quality validation through mutation testing

### Integration Testing
- **Service Integration**: End-to-end testing of service boundaries
- **Database Integration**: Transaction testing with real database instances
- **Message Bus Integration**: Event ordering and delivery guarantees
- **External Service Integration**: Mock external APIs with contract testing
- **Multi-Tenant Testing**: Isolation and data segregation validation

### **Distributed Systems Testing**
#### **Chaos Engineering**
- **Network Partitions**: Simulate network splits and healing
- **Service Failures**: Random service restarts and cascading failures
- **Resource Exhaustion**: CPU, memory, and disk space limitations
- **Clock Skew**: Time synchronization issues across distributed nodes
- **Byzantine Failures**: Arbitrary node behavior simulation

#### **Consistency Validation**
- **Eventually Consistent Systems**: Convergence time measurement
- **Strong Consistency**: ACID transaction validation across services
- **Conflict Resolution**: Concurrent update scenarios with resolution verification
- **Data Integrity**: Cross-service data consistency checks
- **Saga Testing**: Multi-step transaction rollback and compensation

#### **Performance and Load Testing**
- **Throughput Testing**: Message processing rates under load
- **Latency Testing**: End-to-end response time measurement
- **Scalability Testing**: Horizontal scaling behavior validation
- **Resource Utilization**: Memory and CPU usage under various loads
- **Stress Testing**: System behavior beyond normal operational limits

#### **Fault Tolerance Testing**
- **Graceful Degradation**: Service behavior during partial failures
- **Circuit Breaker Testing**: Failure detection and recovery mechanisms
- **Retry Logic**: Exponential backoff and jitter validation
- **Timeout Handling**: Request timeout and cancellation scenarios
- **Bulkhead Pattern**: Isolation effectiveness during failures

### **Behavior Verification**
#### **Agent Behavior Testing**
- **Deterministic Execution**: Same inputs produce same outputs
- **State Machine Validation**: FSM transitions and invariants
- **Behavior Tree Testing**: Complex decision tree validation
- **Goal Achievement**: Success criteria and failure handling
- **Resource Utilization**: Agent resource consumption patterns

#### **Workflow Testing**
- **End-to-End Scenarios**: Complete business process validation
- **Error Propagation**: Error handling across workflow steps
- **Compensation Testing**: Workflow rollback and cleanup
- **Timeout Scenarios**: Long-running workflow behavior
- **Human-in-the-Loop**: Manual intervention and approval flows

#### **Security Testing**
- **Authentication Testing**: Token validation and expiration
- **Authorization Testing**: Role-based access control validation
- **Input Validation**: Injection attack prevention
- **Data Encryption**: At-rest and in-transit encryption verification
- **Audit Trail Testing**: Complete action traceability

### Test Data Management
- **Synthetic Data Generation**: Realistic test data creation
- **Data Anonymization**: Production data sanitization for testing
- **Test Environment Isolation**: Environment-specific test data
- **Data Lifecycle Management**: Test data creation, usage, and cleanup
- **Compliance Testing**: GDPR, HIPAA data handling validation

### Continuous Testing
- **CI/CD Integration**: Automated testing in deployment pipeline
- **Test Environment Management**: Ephemeral test environments
- **Test Automation**: Automated test execution and reporting
- **Performance Regression**: Continuous performance monitoring
- **Security Scanning**: Automated vulnerability assessment

### Test Environments
- **Development Environment**: Local development and unit testing
- **Integration Environment**: Service integration and API testing
- **Staging Environment**: Production-like environment for end-to-end testing
- **Performance Environment**: Dedicated environment for load testing
- **Chaos Environment**: Isolated environment for failure testing

### Legacy Testing Components
- Unit tests: planners (FSM/BT/LLM scaffolding), memory adapters, tool sandbox wrappers, registry.
- Integration: NATS JetStream streams, Postgres migrations, vector backend adapter, OpenTelemetry export.
- E2E: Template workflows; golden trace assertions; cost budget enforcement.
- Replay tests: record+replay streams; verify deterministic outcomes for FSM/BT.
- Security: sandbox escape tests, RBAC enforcement, secrets access boundaries.
- Performance: load tests (k6), chaos tests (message delays, dropped consumers).
- Benchmarking: versioned harness producing QA/coding/RAG metrics across providers (OpenAI/Anthropic/vLLM/Ollama/TGI); CI publishes trend charts.

## 21) Migration & Versioning

## 22) Progressive Complexity Modes (Deprecation Note)

Earlier drafts referenced user-facing modes (Simple / Structured / Recursive). Post research alignment these are deprecated to minimize cognitive load. A single composable architecture remains; planners, memory tiers, adaptive behaviors are opt-in configuration flags. Historical references retained only for transparency.

## 23) Additional SLIs / SLO Hooks

- CostEstimationAccuracy = |estimate-actual|/actual (p50 ≤5%, p95 ≤12% by Gate G6.8)
- PlanDeterminismRate = stable hash(plan_graph) across identical inputs (≥99.5%)
- ReplayFidelity = matched step output hashes original vs replay (≥98% baseline → 99.5%)
- SandboxPolicyCoverage = % tool calls with compiled ExecProfile (100% by Gate G6)
- BudgetPreemptionRatio = preemptions / violations (target >5)
- ToolAnomalyMTTD < 60s

- DB: goose/sqlc migrations; semantic versioned; backward-compatible jsonb fields.
- Workflows/agents: versioned configs; plan cache invalidation on breaking changes.
- Plan cache key includes: planner type/version, policy hash, template version, and relevant env toggles; cache bust on any change.
- Tools: schema version per tool; compatibility checks at registry load.
 - Benchmarks: semver for scenarios; results tagged with commit SHA and environment; reproducible seeds.

## 24) Security & Multi‑Tenancy

- AuthN: JWT (tenant claim), optional OIDC; mTLS between services (in-cluster).
- AuthZ: RBAC roles; per-tenant quotas (agents, workflows, cost, storage).
- Isolation: namespace per tenant; data partition via tenant_id and schema.
- Compliance: audit logging everywhere; PII redaction; right-to-erasure endpoints.
 - Tamper-evident audits: prev_hash field, periodic notarization option; verification CLI.

### HIPAA/PHI Handling
- PHI detection: configurable detectors (regex + dictionary + ML) for identifiers (MRN, SSN), ICD-10 codes, medications, providers.
- Enforcement points: tool outputs/inputs, memory ingestion, message payloads.
- Actions: redact or mask before persistence/egress; tag records with `phi=true` metadata.
- Storage controls: encryption at rest with key rotation; restricted access roles; detailed access logging.
- Incident workflow: anomaly → alert → review → remediation; documented runbooks.

## 25) Failure Modes & Recovery

- Planner failure → fallback strategy (conservative/adaptive/escalation/human-in-loop).
- Tool failure → retry with backoff; circuit break; alternate tool if defined.
- Bus outage → local queue buffer; backpressure; idempotent message handling.
- Cost budget exceeded → soft warn → hard stop with escalation hooks.

## 26) Open Questions

- LLM planner determinism threshold per tenant — default values and override limits?
- Resolved: Default self-hosted vector backend → Qdrant (adapter provided; Pinecone remains default managed option).
- Resolved: UI technology → React + tRPC; HTMX fallback for lightweight ops.

## 27) Acceptance Criteria (mapped to PRD KPIs)

- Residency: With AF_RESIDENCY_POLICY=strict, no external egress occurs; model/memory calls use only on‑prem backends; tests verify via network policy logs.
- Cost preflight: median absolute percentage error ≤ 10% across benchmark scenarios; violations alert and are graphed.
- Audit immutability: verification tool confirms hash‑chain integrity for a sample period; tamper attempt triggers detection alert.



# Appendix A — API Schemas (selected)

- Execute Workflow Request
  - async=true → { job_id, trace_id }
- Memory Query

states:
  knowledge_search:
    on_success: generate_response
    on_error: fallback_response
initial_state: classify
final_states: [send_response, escalate]

# Appendix C — NATS Streams (JetStream) Suggested Config

- Streams
  - AF_MESSAGES: subjects=workflows.*.*, max_age=7d, storage=file, replicas=3
  - AF_AGENTS: subjects=agents.*.*, max_age=3d, storage=file, replicas=3
  - AF_AUDIT: subjects=tools.*, max_age=30d, storage=file, replicas=3
- Consumers
  - durable per agent (agents.<id>.in) with ack wait=30s, max in-flight=concurrency

---

Document owner: Platform Engineering
Reviewers: Security, SRE, DX, Applied AI
Status: Draft v0.1

## Time-Travel Debugging Implementation

### State Snapshots
- Workflow state checkpointing at plan boundaries
- Agent memory snapshots with vector index versioning
- Message stream bookmarking for replay start points

### Replay Infrastructure  
- Sandboxed replay environment (no external side effects)
- Mock external service responses during replay
- Deterministic timestamp injection for consistency

### Guardrails
- Disable all external side-effects during replay (network denied, tools stubbed)
- Secrets resolution disabled or replaced with placeholders in replay mode
- Deterministic random seed and clock injection for reproducible runs

### Data Retention Policies
- Defaults (can be overridden per tenant tier):
  - Messages: 7 days (matches AF_MESSAGES stream)
  - Agent streams: 3 days
  - Audit logs: 30 days
  - Memory records: 180 days (configurable per type)
- Purge jobs run daily; export-before-purge hooks per tenant.

### Failure Analysis
- Pattern detection ML models for common failure modes
- Automated root cause suggestion based on trace analysis
- Integration with alerting systems for proactive detection
