# AgentFlow – 3-Quarter Development Plan (Specs, Tasks, Gates)

Purpose: Translate the PRD, TDD, and Phase Roadmap into an execution-focused, quarter-partitioned plan. Each Quarter groups related Phase(s) into higher-order Specs. Every Spec enumerates tasks with explicit engineering + quality subtasks: (a) Implementation, (b) Unit Tests, (c) Manual / Integration / Perf / Security Validation (as relevant), (d) Documentation & Developer Experience. Exit Steps define objective success criteria to advance. Specs are intentionally inter‑dependent; each Exit references upstream artifacts.

Legend:
- Tasks are hierarchical: Top-level functional task → required subtasks (Impl, Unit, Manual, Docs).
- “Relies On” lists predecessor Spec outputs (ensures traceable linkage across quarters).
- Gates (Gx) map to existing Phase gates where applicable; Quarter exit requires all constituent Spec exits satisfied.

Quarter Summary Alignment:
- Q1 (Months 1–3) – MVP Foundation: P0–P6 focus (Foundations, Messaging, Storage, Control API skeleton, Deterministic Planners, Worker Runtime, Tool Registry + Process Sandbox, Early Cost Heuristics Stub)
- Q2 (Months 4–6) – Enterprise Readiness: P6.5–P8.5 + Partial P9 (Template Engine, Cost Preflight Hard Gates, Model Gateway, MCP Adapter (early), WASM Runtime, Memory MVP, Security Core, Advanced Observability & Replay Foundations)
- Q3 (Months 7–9) – Scale & Advanced Capabilities: Remaining P9–P16 + advanced mid-phase expansions (Benchmarks/SLOs, Deployment Artifacts, Dashboard v1, Marketplace Connector, Security Hardening, CLI Deploy, Performance & Scaling, Production Readiness, Distributed Transactions, Comprehensive Testing Strategy, Advanced Communication Patterns, Agent Dev Tools)

---

## Quarter 1 – MVP Foundation (Months 1–3)

### Spec Q1.1 – Foundations & Project Governance (Phase P0)
Relies On: None (root). Enables: All subsequent Specs.

Tasks:
1. Repository & Module Layout (control plane, worker, cli, sdk stubs, dashboard stub)
   - Impl: Create Go modules, shared internal packages layout, root Makefile/Taskfile.
   - Unit: Lint config test (golangci-lint), placeholder unit test runs for each module.
   - Manual: Run build on Linux + Windows + WSL2; verify task runner parity.
   - Docs: Architecture README section describing repo conventions.
2. Dev Container & Toolchain Standardization
   - Impl: .devcontainer with pinned Go, NATS, Postgres clients; pre-commit hooks.
   - Unit: Script validating required binaries versions.
   - Manual: Open in VS Code devcontainer; run `af validate` stub.
   - Docs: Dev environment guide; Windows fallback notes.
3. CI Pipeline (Build, Lint, Test, SBOM, SAST, Dependencies, Secrets, License, Container Scan)
   - Impl: GitHub Actions workflows; cache strategy; provenance attestation.
   - Unit: Workflow dry-run using act / minimal branch test; config schema lint.
   - Manual: Force failing job for dependency vulnerability; confirm block.
   - Docs: CI policy & gating doc.
4. Security Tooling Integration (gosec, osv-scanner, gitleaks, syft/grype)
   - Impl: Scripts and severity thresholds (fail High/Critical).
   - Unit: Mock reports parsed; threshold logic tested.
   - Manual: Introduce benign vulnerable lib in branch → ensure failure.
   - Docs: Security baseline & exception process.
5. Migration Tooling Decision & Policy (goose + sqlc)
   - Impl: Pin versions; add initial empty migration; sqlc config.
   - Unit: Migration linter test; sqlc code compiles.
   - Manual: Run up/down locally; Windows path validation.
   - Docs: Migration policy (naming, reversibility stance).
6. CLI `af validate` Stub
   - Impl: Outputs JSON skeleton; environment probes placeholders.
   - Unit: JSON schema validation test.
   - Manual: Run on host vs devcontainer; warning displayed.
   - Docs: CLI usage quickstart.
7. Versioning & Release Engineering Baseline
   - Impl: Define semantic version scheme (pre-1.0 minor for breaking changes), tagging policy, CHANGELOG template.
   - Unit: Tag parsing & increment script tests.
   - Manual: Dry-run release workflow producing signed artifacts.
   - Docs: RELEASE.md (versioning & branching model).
8. Multi-Arch Container Build & Signing (Foundational)
   - Impl: Build amd64 + arm64 images (linux) for core services; cosign keyless signing + SBOM attestation integrated in CI.
   - Unit: Manifest list inspection test; signature presence test.
   - Manual: Pull signed image; verify cosign signature.
   - Docs: Supply chain security section (extends security baseline doc).
9. Initial Risk Register & ADR Template
   - Impl: /docs/risk-register.yaml with top ≥8 risks (id, desc, severity, mitigation link); /docs/adr/ template committed + first ADR (architecture baseline).
   - Unit: Risk YAML schema lint test; ADR filename pattern test.
   - Manual: Review sign-off recorded in PR comments.
   - Docs: CONTRIBUTING.md updated referencing ADR & risk processes.
10. Operational Runbook Seed
   - Impl: Create /docs/runbooks/index.md with placeholders (build failure, message backlog, cost spike) linking to future specs.
   - Unit: Link checker passes.
   - Manual: Validate discoverability from root README.
   - Docs: Runbook index (living document).

Exit Step (Gate G0 Criteria):
| Criterion | Measure |
|-----------|---------|
| CI green incl. security scans | All workflows pass; no High/Critical vulnerabilities |
| Cross-platform builds | Linux + Windows + WSL2 builds succeed |
| Devcontainer adoption | `af validate` warns outside container |
| SBOM & provenance | Artifacts published per build |
| Signed multi-arch images | amd64+arm64 images pushed; cosign verify passes |
| Risk register & ADR baseline | risk-register.yaml + first ADR merged |
| Release versioning policy | RELEASE.md published & referenced by CI |
| Interface freeze snapshot | /docs/interfaces (core Q1 interfaces) committed & referenced |
| Threat model kickoff scheduled | Threat modeling session date & owner logged in risk register |

---

### Spec Q1.2 – Messaging Backbone & Tracing Skeleton (Phase P1)
Relies On: Q1.1 toolchain & CI.
Enables: Orchestrator, Worker, Cost tracking, Replay.

Tasks:
1. Subject Taxonomy & Message Contract v1
   - Impl: Define constants, JSON schema, canonical serializer with deterministic field ordering + SHA256 envelope_hash.
   - Unit: Serializer determinism test (stable hash); JSON schema validation tests.
   - Manual: Inspect sample messages; backward-compatible extension scenario.
   - Docs: Message contract & evolution rules.
2. NATS JetStream Integration
   - Impl: Publish/Subscribe/Replay stubs; durable config flags.
   - Unit: In-memory or testcontainer pub/sub tests (ack ordering, replay sequence).
   - Manual: Local nats-server roundtrip latency measurement.
   - Docs: Environment vars (AF_BUS_URL) & retry guidelines.
3. OpenTelemetry Context Propagation
   - Impl: Inject/extract trace parent into message headers.
   - Unit: Trace continuity tests (root span id equality chain).
   - Manual: Jaeger UI verification end-to-end trace.
   - Docs: Trace attribute key conventions.
4. Structured Logging Baseline
   - Impl: Logger wrapper with trace & message IDs; JSON format.
   - Unit: Log enrichment test (contains correlation IDs).
   - Manual: Tail logs during ping-pong; verify fields present.
   - Docs: Logging standard & reserved keys.
5. Basic Performance Harness (Ping-Pong)
   - Impl: Benchmark script measuring p50/p95.
   - Unit: Assertion p95 < threshold in CI (allow env override).
   - Manual: Run locally; record baseline.
   - Docs: Perf harness usage & thresholds.

Exit Step (Gate G1): Deterministic message hashing; pub/sub/replay tests pass; OTEL spans visible; ping‑pong p95 < 15ms (CI Linux), doc published.
Additional Quantitative Assertions (G1 augmentation): Canonical serializer property-based test (N iterations no hash collision for permuted fields); envelope_hash recomputation variance = 0 across test matrix.

---

### Spec Q1.3 – Relational Storage & Migrations (Phase P2)
Relies On: Q1.1 (migrations tooling), Q1.2 (message envelope_hash for messages table).
Enables: Control API, Planners, Tool Audits, Budgets.

Tasks:
1. Core Schema Migrations (tenants, users, agents, workflows, plans, messages, tools, audits, budgets, rbac_roles, rbac_bindings)
   - Impl: SQL up migrations + minimal safe down (if applicable); indexes.
   - Unit: sqlc generated query compilation; forward/back clone test.
   - Manual: Fresh migrate, seed, rollback previous, re-migrate.
   - Docs: Schema ER diagram & changelog.
2. Audit Hash-Chain Columns
   - Impl: prev_hash + hash computation function (H(prev||serialized)).
   - Unit: Append-only integrity tests; tamper detection test.
   - Manual: Manual SQL tamper → CLI verify shows failure.
   - Docs: Hash-chain rationale & verification procedure.
3. Envelope Hash Persistence (messages table)
   - Impl: Insert trigger or app-layer guarantee.
   - Unit: Missing hash rejection test.
   - Manual: Inspect stored messages vs recomputed hash.
   - Docs: Replay integrity notes.
4. Redis & Vector Dev Bootstrap
   - Impl: docker-compose services + health checks; feature flags.
   - Unit: Connectivity tests; conditional skip on Windows if needed.
   - Manual: Start services; verify `af validate` surfaces status.
   - Docs: Local services guide.
5. Secrets Provider Stub
   - Impl: Interface + env/file provider methods.
   - Unit: Retrieval + masking tests.
   - Manual: Rotate sample secret file; reload verified.
   - Docs: Secrets usage & future providers.
6. Audit Verification CLI Subcommand
   - Impl: `af audit verify` computes chain & reports first tamper index.
   - Unit: Injected tamper fixture detection test (exit code >0).
   - Manual: Run against pristine vs modified DB.
   - Docs: Forensics verification procedure (extends hash-chain doc).
7. Backup & Restore Baseline
   - Impl: Script pg_dump (schema+data selective tables) + restore smoke test in CI.
   - Unit: Backup artifact integrity hash test.
   - Manual: Simulated accidental table drop → restore.
   - Docs: DR baseline & RPO/RTO placeholders.
8. MemoryStore Stub (In-Memory + Noop Summarizer)
   - Impl: Minimal in-memory implementation of Save/Query plus placeholder Summarize returning constant; wired into worker/planner DI (flag experimental until Q2.6 replaces).
   - Unit: Save/query determinism test; summarizer no-op assertion; basic race detector run.
   - Manual: Sample plan writes memory; verify retrieval via debug log or temporary inspection endpoint.
   - Docs: Note in storage doc referencing upgrade path to Q2.6 Memory Subsystem MVP.

Exit Step (Gate G2): All migrations apply & rollback cleanly; hash-chain integrity passes; Redis/Vector health validated; Windows migration run passes.
Additional Quantitative Assertions (G2 augmentation): Audit chain verify throughput ≥ 10k entries/sec on dev hardware; `af audit verify` detects injected tamper within first mismatched entry index; backup+restore roundtrip < 5 min for baseline dataset.

---

### Spec Q1.4 – Control Plane API Skeleton (Phase P3)
Relies On: Q1.3 schema, Q1.2 tracing.
Enables: Planner invocation, Budgets stub, SDK generation.

Tasks:
1. HTTP Server & Routing (/api/v1) + Middleware Stack (logging, tracing, recovery)
   - Impl / Unit / Manual / Docs (as pattern below).
2. AuthN (JWT Dev Secret) & Optional OIDC Flag
   - Impl: Token issuance, validation.
   - Unit: Auth success/failure tests.
   - Manual: OIDC flag off/on flows.
   - Docs: Auth flows & token claims.
3. Multi-Tenancy Enforcement
   - Impl: Tenant scoping queries & subject prefixes.
   - Unit: Cross-tenant access denial tests.
   - Manual: Two tenants seeded; ensure isolation.
   - Docs: Tenancy model.
4. RBAC Seed Roles (admin, developer, viewer)
   - Impl: Role binding & middleware enforcement.
   - Unit: Negative mutation tests for viewer.
   - Manual: Role switch smoke test.
   - Docs: RBAC matrix.
5. Rate Limiting (Redis)
   - Impl: Sliding window or token bucket.
   - Unit: Burst + sustained tests.
   - Manual: 429 header verification.
   - Docs: Quota semantics & headers.
6. OpenAPI Contract + SDK Codegen (Python/JS stubs)
   - Impl: Spec generation & semantic diff CI.
   - Unit: Schema lint tests.
   - Manual: Generated SDK import & basic call.
   - Docs: API versioning policy.
7. Budgets & Cost Estimate Endpoint (Heuristic)
   - Impl: PlanCostModel.Estimate + /plans/{id}/estimate.
   - Unit: Deterministic output fixtures.
   - Manual: Over-budget warning log.
   - Docs: Cost estimation limitations.
8. ExecProfileCompiler (deny-by-default tool perms) – Stub Enforcement
   - Impl: Profile synthesis; attach to tools.
   - Unit: Missing permission denial tests.
   - Manual: Attempt unauthorized host call.
   - Docs: Permission taxonomy.
9. Data Minimization Middleware (Feature Flag)
   - Impl: Redaction rules engine.
   - Unit: Golden redaction tests.
   - Manual: Log scan for PII.
   - Docs: Minimization strategy.
10. Residency Policy Gate (Strict Mode)
    - Impl: Egress filter & allow-list.
    - Unit: Block external host test.
    - Manual: Toggle flag & attempt egress.
    - Docs: Residency configuration.

Exit Step (Gate G3): Multi-tenant auth & RBAC enforced; OpenAPI stable; cost estimate warnings; residency strict mode blocks egress; SDK smoke tests pass; data minimization golden tests achieve 0 leaked sample PII tokens.
Gate G3 Clarification Addendum: OpenAPI semantic diff job records no unapproved breaking changes (exceptions linked to ADR); tenancy isolation matrix executed (cross-tenant requests denied & logged).

---

### Spec Q1.5 – Orchestrator & Deterministic Planners (Phase P4)
Relies On: Q1.4 API auth/tenancy, Q1.2 messaging.
Enables: Workflow execution, budgeting, composition.

Tasks:
1. Planner Interface & FSM Planner
   - Impl: YAML parsing, validation.
   - Unit: Determinism & error handling tests.
   - Manual: Sample FSM run.
   - Docs: FSM spec & constraints.
2. Behavior Tree Planner
   - Impl: Node types (sequence, selector, parallel, condition).
   - Unit: Branching determinism tests.
   - Manual: Complex scenario walkthrough.
   - Docs: BT node semantics.
3. Workflow Composition (Sub-Workflows / Fan-Out / Barriers)
   - Impl: Composition engine + barrier sync.
   - Unit: Fan-out/fan-in test, nested workflow test.
   - Manual: Barrier delay scenario.
   - Docs: Composition patterns.
4. Plan Cache & Admin APIs
   - Impl: TTL plan store; invalidate endpoint.
   - Unit: Cache hit/miss metrics tests.
   - Manual: Cache invalidation manual trigger.
   - Docs: Cache strategy.
5. Cost Preflight Integration (Heuristic Only)
   - Impl: Attach estimate/cost meta to plan.
   - Unit: Estimate attachment test.
   - Manual: Plan debug output includes cost.
   - Docs: Preflight flow (stub stage).
6. Router Module (NATS subjects & correlation IDs)
   - Impl: Publish plan steps messages.
   - Unit: Correlation ID continuity tests.
   - Manual: Trace inspection in Jaeger.
   - Docs: Routing subject map.
7. Replay Harness (Seed & Clock Controls – Partial)
   - Impl: Deterministic seed injection.
   - Unit: Repeatable output test.
   - Manual: Two run diff comparison.
   - Docs: Replay design (extended in Q2/Q3).
8. Idempotent Plan Step Handling
   - Impl: Step execution dedupe cache keyed by (plan_id, step_id, envelope_hash).
   - Unit: Duplicate delivery suppressed test.
   - Manual: Inject duplicate message; confirm single side-effect.
   - Docs: Idempotency contract section (planning doc addendum).
9. Deterministic Plan/Step ID Strategy
   - Impl: Monotonic ULID (or ULID + stable ordering hash) generation spec; optional plan structure hash stable across identical definitions.
   - Unit: Collision property test (N=50k); plan hash stability for identical YAML; permutation of unordered fields yields identical hash.
   - Manual: Generate identical plan twice → identical IDs & hash logged.
   - Docs: Planning guide subsection on ID determinism & collision handling.

Exit Step (Gate G4): FSM/BT deterministic; composition & barriers validated; plan cost meta attached; router spans linked; replay harness yields identical outputs for fixed seed; idempotent step handling tests pass; deterministic ID collision tests green.

---

### Spec Q1.6 – Data Plane Worker Runtime (Phase P5 core)
Relies On: Q1.5 planners, Q1.2 messaging.
Enables: Plan execution loop, cost observation.

Tasks:
1. Worker Lifecycle (OnStart/OnMessage/OnPlan/OnFinish)
   - Impl: Hooks + OTEL spans.
   - Unit: Lifecycle order tests.
   - Manual: Start/stop sequence logs.
   - Docs: Worker lifecycle diagram.
2. Durable Consumer Setup & Backpressure
   - Impl: Max in-flight, redelivery policy.
   - Unit: Burst test ensuring no loss.
   - Manual: Stress script observation.
   - Docs: Backpressure tuning guide.
3. CostObservation Emission
   - Impl: Emit metrics per tool/model.
   - Unit: Metric labels test.
   - Manual: Prom scrape verification.
   - Docs: Cost calibration overview.
4. Local Budget Guard (Soft)
   - Impl: Mirror central estimate (warn only).
   - Unit: Over-budget warning test.
   - Manual: Induce exceed scenario.
   - Docs: Guard rationale.
5. Replay-Safe Mode (Side-Effect Suppression Flag)
   - Impl: Toggle to no-op external calls.
   - Unit: Counter stays zero test.
   - Manual: Dual run comparison.
   - Docs: Replay instructions.
6. Benchmark Harness (Routing p95, Cold Start)
   - Impl: Script measuring metrics.
   - Unit: Assert thresholds in CI.
   - Manual: Record local baseline.
   - Docs: Benchmark invocation.
7. Idempotent Message Processing
   - Impl: Envelope hash + dedupe window to prevent reprocessing side-effects.
   - Unit: Duplicate redelivery acceptance (ack) without side-effect test.
   - Manual: Force NATS redelivery; observe single effect.
   - Docs: Idempotency operational notes (extends runbook).
8. Cost Actual vs Estimate Tagging
   - Impl: Attach estimate reference to emitted CostObservation for later variance metric.
   - Unit: Observation includes estimate id test.
   - Manual: Compare log vs metrics for sample plan.
   - Docs: Calibration linkage.
9. Redelivery Failure Injection Harness
   - Impl: Harness kills worker mid-step to trigger NATS redelivery; validates dedupe & exactly-once side-effects.
   - Unit: Crash scenario leaves no orphaned messages; side-effects exactly once.
   - Manual: Run harness observing redelivery logs & metrics.
   - Docs: Runbook snippet (redelivery troubleshooting) referencing harness.

Exit Step (Gate G5): Worker executes simple FSM plan end‑to‑end <100ms p95; cost observations emitted (with estimate linkage); replay-safe mode verified; no message loss under burst; idempotent message processing test passes; redelivery harness demonstrates exactly-once side-effects after induced crash.

---

### Spec Q1.7 – Tool Registry & Process Sandbox (Phase P6)
Relies On: Q1.4 ExecProfileCompiler, Q1.6 worker.
Enables: Secure tool execution, future anomaly detection.

Tasks:
1. Tool Interface & Registry CRUD
   - Impl: Tool schemas, REST endpoints.
   - Unit: CRUD & permission tests.
   - Manual: Register sample HTTP tool.
   - Docs: Tool schema reference.
2. Process Sandbox Executor (Default)
   - Impl: Restricted exec, resource caps.
   - Unit: Deny network by default; syscall/resource audit log presence; timeout enforcement.
   - Manual: Attempt unauthorized call & inspect audit log for denied syscall/host; induce resource cap breach → audited.
   - Docs: Sandbox guarantees & gaps + audit logging & escape detection subsection.
3. ExecProfile Enforcement
   - Impl: Apply timeouts, host allow-lists.
   - Unit: Missing permission denial.
   - Manual: Allowed host success, blocked host fail.
   - Docs: Profile fields explanation.
4. Audit Logging (tools.calls, tools.audit)
   - Impl: Append audit with hash-chain integration.
   - Unit: Hash continuity tests.
   - Manual: Inspect audit entries.
   - Docs: Audit event schema.
5. Basic Anomaly Detector (MVP Signals)
   - Impl: Subscribe & emit tools.anomaly.
   - Unit: Simulated anomaly triggers.
   - Manual: Latency spike scenario.
   - Docs: Anomaly signal catalog.
6. PHI Filter Hooks (Optional)
   - Impl: Redaction pre-persist.
   - Unit: Redaction golden tests.
   - Manual: Verify absence of PII.
   - Docs: Config & field mapping.

Exit Step (Gate G6): Deny-by-default enforced; audits hash-chained; anomaly events produced under synthetic triggers; process sandbox reliable on Windows + Linux.
Additional Quantitative Gate Additions (G6 augmentation): ExecProfile applied to 100% tool calls (sampled audit); PHI filter tests 0 leaked sample tokens.

---

### Spec Q1.8 – Observability & Metrics Baseline (Cross-Cutting Slice)
Relies On: Q1.2 tracing/logging, Q1.6 worker cost observation, Q1.7 tool events.
Enables: Later cost accuracy tracking (Q2.2), replay analytics (Q3.1), SLO dashboards (Q3.2).

Rationale: PRD/TDD emphasize production-grade visibility. Establishing a uniform metrics, logging, and trace correlation layer now avoids retrofitting instrumentation and ensures early performance regressions are caught.

Tasks:
1. Metrics Instrumentation Framework
   - Impl: Prometheus registry module; standardized label set (tenant, workflow_id, agent_id, tool_id, plan_id, outcome).
   - Unit: Metric registration & duplication guard tests.
   - Manual: /metrics endpoint scrape locally; verify core families present.
   - Docs: Metrics naming convention & reserved labels.
2. Core Counter & Histogram Set (v1)
   - Impl: agent_messages_total, tool_calls_total, planner_plan_duration_seconds (hist), worker_execution_latency_seconds (hist), cost_observation_tokens_total, sandbox_denials_total.
   - Unit: Emission tests with deterministic label values.
   - Manual: Run sample plan; confirm non-zero metrics.
   - Docs: Metric catalog (baseline section).
   - Extension: Add tool_execution_duration_seconds (hist) & memory_retrieval_duration_seconds (hist).
3. Log Field Governance & Linter
   - Impl: Reserved fields list (trace_id, span_id, message_id, plan_id, tenant_id, severity, event_type) + CI linter to detect collisions/missing mandatory keys in structured logs (sample-based regex/assertion).
   - Unit: Linter rule tests.
   - Manual: Introduce malformed log in branch – job fails.
   - Docs: Logging field governance doc update (Q1.2 extension).
4. Trace ↔ Log Correlation Helpers
   - Impl: Middleware injecting trace/span IDs into context-aware logger; correlation test harness.
   - Unit: Trace/log ID match tests.
   - Manual: Jaeger trace spans link to log lines via IDs.
   - Docs: Correlation usage guide.
5. Minimal Grafana Dashboard JSON (v0)
   - Impl: Panels for message latency, plan duration, tool calls rate, cost tokens.
   - Unit: JSON schema validation test.
   - Manual: Import into local Grafana; screenshot artifact.
   - Docs: Dashboard import instructions.
6. Performance Smoke CI Job
   - Impl: Lightweight scenario executing ping-pong + simple plan; asserts p95 thresholds using exported metrics (not ad-hoc scripts only).
   - Unit: Threshold evaluation logic tests.
   - Manual: Induce latency increase → job fails.
   - Docs: Performance gating policy (initial).
   - Extension: Define initial error budget (routing p99 <40ms, worker exec p99 <150ms) – monitor only (no fail) Q1.

Exit Step (Gate G6.1): Metrics families exposed & scraped; baseline dashboard committed; log linter enforced in CI; performance smoke job passes with p95 routing <15ms & worker exec <100ms.

---

### Spec Q1.9 – Minimal Model Gateway & LLM Agent Stub (Early Slice of Phase P7)
Relies On: Q1.4 residency gate, Q1.5 planners (for cost estimation integration), Q1.6 worker lifecycle, Q1.8 observability metrics.
Enables: Early testing of cost heuristics with real provider responses; derisks Q2 full provider routing.

Rationale: Prevent tight coupling between planners/tools and individual provider SDKs; provide token accounting & residency hooks now.

Tasks:
1. Provider Interface v0
   - Impl: interface { Complete(ctx, req) (resp, error); TokenUsage(resp) (prompt, completion int); } with OpenAI mock + noop provider.
   - Unit: Mock roundtrip & error translation tests.
   - Manual: Sample completion call logs metrics.
   - Docs: Provider interface quickref.
2. Residency Enforcement Hook
   - Impl: Wrapper enforcing allow-list (on‑prem vs external) using existing residency policy; deny if strict.
   - Unit: External denied test under strict mode.
   - Manual: Toggle strict → verify block event.
   - Docs: Residency integration note (extends Q1.4 doc).
3. Token & Cost Heuristic Integration
   - Impl: Translate provider token counts → cost_observation metrics; update PlanCostModel to optionally call gateway for dynamic estimate refinement (flagged experimental).
   - Unit: Cost variance test harness (fixture sizes).
   - Manual: Compare heuristic vs real tokens log.
   - Docs: Early calibration procedure (stub).
4. LLM Agent Stub
   - Impl: Agent runtime wrapper performing single Complete call with prompt template; emits planner_plan_duration_seconds and tool_calls_total entries.
   - Unit: Deterministic prompt assembly test.
   - Manual: End-to-end simple plan includes LLM agent step.
   - Docs: LLM agent configuration example.
5. Observability & Error Semantics
   - Impl: Standard error codes (timeout, rate_limited, provider_error); map to metrics & structured logs.
   - Unit: Error mapping tests.
   - Manual: Inject provider error; verify metrics increment.
   - Docs: Error taxonomy addendum.
6. Feature Flag (llm_gateway.enabled)
   - Impl: Integrated with Q1.10 config system.
   - Unit: Flag off -> calls blocked test.
   - Manual: Toggle & verify behavior.
   - Docs: Flag reference table entry.

Exit Step (Gate G6.2): OpenAI mock + noop providers operational; LLM agent executes in plan; residency strict mode blocks disallowed provider; cost tokens exported; error taxonomy documented.
Augmentation: cost_estimate_variance_ratio metric emitted (collection only), variance samples >= N=10 captured.

---

### Spec Q1.10 – Configuration, Feature Flags, Test Strategy & Threat Modeling (Cross-Cutting Governance)
Relies On: Q1.1 foundations, Q1.4 API scaffolding, Q1.8 metrics (for test gating), Q1.9 feature flag integration.
Enables: Controlled rollout of experimental features, repeatable test baselines, security risk tracking into Q2 (security core).

Tasks:
1. Configuration Layer
   - Impl: Hierarchical precedence (env > file > defaults); structured validation (e.g. envconfig + JSON schema for file variant).
   - Unit: Precedence & validation tests.
   - Manual: Override scenario demonstration.
   - Docs: Config loading order doc.
2. Feature Flag System
   - Impl: In-memory registry + typed accessors + dynamic reload (poll interval); initial flags: data_minimization, residency_strict, replay_safe_mode, llm_gateway.enabled, anomaly_detector, sandbox_strict.
   - Unit: Flag toggle propagation tests.
   - Manual: Runtime toggle demonstration.
   - Docs: Flag reference matrix.
3. Test Strategy & Coverage Gates
   - Impl: Define coverage thresholds (≥70% Q1 critical path, rising later); introduce coverage diff CI job; flaky test quarantine label process.
   - Unit: Coverage parser & threshold tests.
   - Manual: Introduce under-threshold branch → CI fails.
   - Docs: /docs/testing.md (strategy, pyramid, replay tests).
4. Deterministic Replay CI Job (Baseline)
   - Impl: Use Q1.5 replay harness to re-run golden plan scenario; diff serialized outputs (plan hash, key events).
   - Unit: False positive guard tests.
   - Manual: Induce nondeterminism; job fails.
   - Docs: Replay CI job section (extends planning doc).
5. Threat Modeling Workshop & Artifact
   - Impl: STRIDE-lite for core components; produce risk register with severity & mitigations mapping to future specs (tag tasks with risk IDs).
   - Unit: Lint risk YAML schema.
   - Manual: Review sign-off by security owner.
   - Docs: security.md updated with threat model summary.
   - Clarification: Include existing /docs/risk-register.yaml continuity and link to ADRs.
6. Migration Rollback Automation
   - Impl: Script to apply latest migration then rollback N steps; smoke verification of schema diff emptiness; Windows compatibility.
   - Unit: Rollback script tests.
   - Manual: Induce failing migration scenario & recover.
   - Docs: Migration rollback policy (extends storage doc).
7. Cost Accuracy Baseline Collection
   - Impl: Capture (heuristic_estimate, observed_tokens) pairs for sample plans; store in metrics (cost_estimate_variance_ratio) & JSON artifact in CI.
   - Unit: Variance computation tests.
   - Manual: Inspect artifact history.
   - Docs: Calibration roadmap.
8. Governance Artifacts Publication
   - Impl: Ensure ADR index & risk register surfaced in root README; add doc owner table.
   - Unit: Link presence tests.
   - Manual: README navigation review.
   - Docs: Governance section in README.
9. Error Taxonomy & Retry Policy Definition
   - Impl: Central classification (transient, validation, auth, quota, internal) with mapping for planner, worker, gateway; default exponential backoff + jitter.
   - Unit: Fixture errors map to expected class; retry budget exhaustion test.
   - Manual: Simulated transient provider timeout retried then succeeds; permanent error not retried.
   - Docs: /docs/testing.md & planning guide sections; new ADR (error-taxonomy) capturing rationale & mapping table.

Exit Step (Gate G6.3): Config & flags system operational; coverage gate enforced; replay CI deterministic; threat model artifact merged; migration rollback script green; initial cost variance metrics collected.
Augmentation: Governance artifacts (ADR index + risk register link) visible in README.

---

### Spec Q1.11 – MCP Protocol Adapter Early Slice (Handshake & Mapping)
Relies On: Q1.7 tool registry, Q1.8 observability baseline, Q1.9 model gateway (for any LLM-backed MCP tools), Q1.10 config/flags.
Enables: Early external tool ecosystem evaluation; feedback loop on permission & audit model before expansion in Q2.

Scope Boundary (Q1): Only minimal handshake, descriptor → internal ToolSchema mapping, basic permission derivation (network.* allow-list), audit event emission, feature flag gating. Advanced resiliency, redaction parity, timeouts/circuit breakers, and full audit schema equivalence deferred to Q2 expansion.

Tasks:
1. Feature Flag & Config Wiring
   - Impl: Flag mcp_adapter.enabled controls registration & endpoint exposure.
   - Unit: Flag off denies MCP init tests.
   - Manual: Toggle flag runtime demonstration.
   - Docs: Flag table update.
2. Handshake & Descriptor Fetch
   - Impl: Minimal client establishing MCP connection, retrieving tool descriptors.
   - Unit: Mock MCP server handshake tests (version, capability negotiation).
   - Manual: Connect to sample local MCP server.
   - Docs: Early MCP flow sequence diagram.
3. Descriptor → ToolSchema Mapping v1
   - Impl: Field mapping (name, description, input schema, output schema) with normalization rules.
   - Unit: Mapping fidelity tests against fixtures.
   - Manual: Inspect generated ToolSchema via API.
   - Docs: Mapping reference (initial subset table).
4. Permission Derivation (Basic)
   - Impl: Parse descriptor network calls (if provided) → ExecProfile host allow-list entries; default deny.
   - Unit: Unknown host denial test.
   - Manual: Attempt call to disallowed host.
   - Docs: Permission derivation doc (alpha).
5. Registration & Lifecycle Integration
   - Impl: Register mapped tools in registry with source=mcp; mark provenance for audit.
   - Unit: Registry CRUD roundtrip with source label.
   - Manual: List tools; verify MCP origin tagging.
   - Docs: Tool registry doc extension.
6. Audit & Telemetry Emission
   - Impl: Emit audit events (mcp.handshake, mcp.tool.register) including hash-chain integration; metrics mcp_handshakes_total, mcp_tools_registered_total.
   - Unit: Audit event schema tests.
   - Manual: View metrics & audit log entries.
   - Docs: Audit event definitions addendum.
7. Minimal Error Handling & Backoff
   - Impl: Single retry with fixed backoff for handshake; structured error codes.
   - Unit: Retry on transient failure test.
   - Manual: Induce failure (kill server) observe retry.
   - Docs: Limitations section (what's deferred).
8. Security & Threat Review Update
   - Impl: Update threat model (Q1.10) with MCP-specific risks (descriptor spoofing, permission escalation).
   - Unit: Risk entry lint test.
   - Manual: Security sign-off comment on PR.
   - Docs: security.md risk table update.

Exit Step (Gate G6.4): Flag-gated MCP adapter registers external tools (≥1 sample) with audited handshake; denied unknown hosts; mapping tests pass; telemetry metrics exposed; threat model updated with MCP risks.

---

Quarter 1 Exit Criteria: All Spec exits (G0–G6.4) satisfied; end-to-end demo (plan → worker → tool call incl. LLM & MCP-imported tool) reproducible in <5 minutes setup and <2 minutes first workflow execution; baseline metrics dashboard & replay + coverage + performance smoke CI jobs green; threat model (incl. MCP risks) & initial cost variance report published.

---

## Quarter 2 – Enterprise Readiness (Months 4–6)

### Spec Q2.1 – Template Engine & Developer Onboarding (Phase P6.5)
Relies On: Q1.7 tool registry for sample template tool references.
Enables: Template-first adoption & Marketplace integration later.

Tasks:
1. TemplateEngine Interface + Manifest Schema
   - Impl / Unit (schema validation) / Manual (invalid manifest) / Docs (schema ref).
2. Local Template Rendering (`af init`)
   - Impl: Deterministic render; dry-run.
   - Unit: Golden file render tests.
   - Manual: Windows path & newline validation.
   - Docs: Template author guide.
3. Starter Templates (Workflow + Agent + Tool)
   - Impl: Minimal reproducible example.
   - Unit: Lint & compile tests for generated code.
   - Manual: Run end-to-end from template.
   - Docs: Template catalog.
4. Semantic Version Binding
   - Impl: Version parser & constraint resolver.
   - Unit: Range resolution tests.
   - Manual: Upgrade path scenario.
   - Docs: Versioning strategy.

Exit Step (Gate G6.5): `af init` renders deterministic project cross-platform; semantic version constraints enforced; starter template passes smoke tests.

---

### Spec Q2.2 – Cost Preflight Hard Gates & Resource Management (Phase P6.8)
Relies On: Q1.5 plan structure, Q1.6 worker cost obs, Q2.1 templates supplying sample plans.
Enables: Budget enforcement & later advanced cost analytics.

Tasks:
1. Provider/Tool Cost Model Contracts
   - Impl: Interface returning est_tokens/$ + confidence.
   - Unit: Fixture-based estimates tests.
   - Manual: Add provider; observe registry update.
   - Docs: Cost model authoring.
2. Multi-Dimensional Resource Tracking (CPU/Mem/Net/Storage/Model)
   - Impl: Metrics & attribution tags.
   - Unit: Resource usage aggregation tests.
   - Manual: Synthetic workload saturating one dimension.
   - Docs: Resource taxonomy.
3. Scheduling Algorithms (Bin Packing / Load Balancing)
   - Impl: Allocation engine with pluggable strategies.
   - Unit: Packing efficiency test harness.
   - Manual: Strategy switch A/B comparison.
   - Docs: Scheduling design & tuning.
4. Dynamic Pricing Models (Time/Usage/Value-Based)
   - Impl: Pricing evaluator integrated into estimates.
   - Unit: Peak/off-peak savings simulation tests.
   - Manual: Off-peak scenario walkthrough.
   - Docs: Pricing policy definitions.
5. Chargeback/Showback Attribution
   - Impl: Aggregation & reporting endpoints.
   - Unit: Allocation correctness tests.
   - Manual: Multi-tenant report generation.
   - Docs: Financial reporting guide.
6. Preflight API & Gate Token
   - Impl: Signed token (plan scope, expiry, budget hash).
   - Unit: Tampered/expired token tests.
   - Manual: Blocked over-budget run scenario.
   - Docs: Gate token format.
7. Worker Enforcement (Hard Block)
   - Impl: Validation before execution.
   - Unit: Refusal test without valid token.
   - Manual: Over-budget plan blocked.
   - Docs: Enforcement flow diagram.
8. Metrics & Dashboards (cost_preflight_* & utilization_ratio)
   - Impl: Prom collectors + sample Grafana json.
   - Unit: Metric exposure tests.
   - Manual: Dashboard import & check.
   - Docs: Dashboard usage.

Exit Step (Gate G6.8): Over-budget plans blocked with audit; p50 estimate error <20%; resource utilization >80% on synthetic; dynamic pricing saves ≥15% in test scenario; worker rejects invalid tokens.

---

### Spec Q2.3 – Model Gateway & Provider Routing (Phase P7)
Relies On: Q2.2 cost gates, Q1.4 residency, Q1.6 worker.
Enables: Embeddings, LLM planner path, Memory subsystem.

Tasks:
1. Provider Abstraction (OpenAI, Anthropic, vLLM, Ollama, TGI)
   - Impl: Unified Complete/Embed.
   - Unit: Mock provider tests.
   - Manual: Residency strict -> on‑prem routing.
   - Docs: Provider config matrix.
2. Budget Integration & Rate Limiting
   - Impl: Central + worker guard parity checks.
   - Unit: Allow/block parity tests.
   - Manual: Quota exhaustion scenario.
   - Docs: Budget interplay.
3. Retry / Backoff / Circuit Breaker
   - Impl: Policy-driven resilience.
   - Unit: Failure injection tests.
   - Manual: Simulated outage failover.
   - Docs: Resilience policy.
4. Telemetry & Metrics (latency, tokens, failures)
   - Impl: OTEL spans + Prom histograms.
   - Unit: Label correctness tests.
   - Manual: Metrics dashboard review.
   - Docs: Metric dictionary.
5. Residency Enforcement (Zero Egress)
   - Impl: Transport guard.
   - Unit: External call blocked test.
   - Manual: Strict vs relaxed mode.
   - Docs: Residency scenarios.

Exit Step (Gate G7): Provider failover validated; residency strict blocks external providers; budget enforcement parity proven; telemetry complete.

---

### Spec Q2.4 – MCP Protocol Adapter Expansion (Phase P7.5)
Relies On: Q1.11 MCP early slice, Q2.3 provider telemetry for consistency baseline.
Enables: Production-grade external tool integration.

Tasks:
1. Advanced Resilience (Timeouts, Circuit Breakers, Exponential Backoff)
   - Impl: Policy-driven per-call timeouts & hedging (optional); configurable breaker thresholds.
   - Unit: Failure injection & breaker open/close state tests.
   - Manual: Simulated latency spike demonstrating breaker behavior.
   - Docs: Timeout & circuit breaker matrix.
2. Descriptor Delta Sync & Hot Reload
   - Impl: Periodic diff + update of MCP tool descriptors; safe in-place tool update with versioning.
   - Unit: Delta application tests.
   - Manual: Modify descriptor; observe version bump.
   - Docs: Descriptor lifecycle section.
3. Audit Parity & Redaction Enhancement
   - Impl: Full parity of audit fields vs native tools; sensitive field redaction rules.
   - Unit: Audit schema equivalence tests.
   - Manual: Redacted field verification.
   - Docs: Audit parity specification (finalized).
4. Streaming & Large Payload Handling
   - Impl: Chunked transfer or streaming API bridging for long-running MCP tool calls.
   - Unit: Stream ordering & completion tests.
   - Manual: Long-running tool demo.
   - Docs: Streaming usage guide.
5. Performance & Resource Metrics
   - Impl: mcp_call_duration_seconds (hist), mcp_active_sessions, mcp_errors_total (by code), integration with cost observations if applicable.
   - Unit: Metric emission tests.
   - Manual: Dashboard panel import.
   - Docs: Metrics dictionary update.
6. Security Hardening
   - Impl: Mutual auth (if supported), descriptor signature verification (optional), rate limiting per MCP server.
   - Unit: Signature mismatch & rate limit tests.
   - Manual: Invalid signature scenario.
   - Docs: Security enhancements section.
7. Load & Soak Test Suite
   - Impl: 30m soak harness with synthetic MCP tool calls; memory leak detection.
   - Unit: Resource usage threshold tests.
   - Manual: Soak run artifact review.
   - Docs: Soak test methodology.

Exit Step (Gate G7.5): MCP adapter maintains stable latency under soak; circuit breakers function; audit parity achieved; streaming & redaction validated; security enhancements operational.

---

### Spec Q2.5 – WASM Agent Runtime (Phase P7.8)
Relies On: Q1.6 worker lifecycle.
Enables: Multi-language agent development.

Tasks:
1. WASM Runtime Integration (wazero/wasmtime)
   - Impl: Load, instantiate, limits.
   - Unit: Memory/time limit tests.
   - Manual: Sample module run.
   - Docs: Runtime architecture.
2. AgentLifecycle Bridging (OnStart/OnMessage/OnPlan)
   - Impl: Host functions & JSON bridge.
   - Unit: Roundtrip serialization tests.
   - Manual: Cross-language example.
   - Docs: Host function reference.
3. Module Pooling & Hot Reload (Dev)
   - Impl: Cache & reload strategy.
   - Unit: Reload retains state tests.
   - Manual: Edit → rebuild → reload timing.
   - Docs: Dev workflow guide.
4. Language SDK Bindings (Rust, Python, TS)
   - Impl: Minimal wrappers.
   - Unit: Build/test each binding.
   - Manual: Run per-language example.
   - Docs: Binding usage examples.

Exit Step (Gate G7.8): WASM agents meet perf targets (cold <50ms, warm <5ms); isolation enforced; multi-language examples succeed.

---

### Spec Q2.6 – Memory Subsystem MVP (Phase P8)
Relies On: Q2.3 model gateway for embeddings; Q1.3 schema.
Enables: RAG features, context persistence.

Tasks:
1. MemoryStore Interface & SQL + Vector Schema
   - Impl: memory_records + adapters.
   - Unit: CRUD & embedding persistence tests.
   - Manual: Save/query flow.
   - Docs: Interface & schema reference.
2. Retrieval Pipeline (Cache → Vector → Rerank → Summarize)
   - Impl: Stage orchestrator.
   - Unit: Cache hit, summarization tests.
   - Manual: Large context summarization.
   - Docs: Retrieval flow diagram.
3. Cost Attribution & Budgets
   - Impl: Per-operation token tracking.
   - Unit: Budget exceed test.
   - Manual: Budget adjustment scenario.
   - Docs: Cost fields documentation.
4. GDPR Erasure Endpoint
   - Impl: Targeted deletion + audit.
   - Unit: Verify no residual queries.
   - Manual: Erasure request demo.
   - Docs: Privacy & compliance section.

Exit Step (Gate G8): Deterministic retrieval tests pass; budgets enforced; latency targets met; erase path audited.

---

### Spec Q2.7 – Security Core Early Enforcement (Phase P8.5)
Relies On: Q1.7 sandbox/audits, Q2.6 memory, Q2.3 residency.
Enables: Enterprise posture for early adopters.

Tasks:
1. External Secrets Provider (Vault/AWS/Azure Read-Only)
   - Impl / Unit (no logging) / Manual (secret rotation) / Docs (provider setup).
2. mTLS Between Services
   - Impl: Cert issuance & rotation.
   - Unit: Expired cert rejection test.
   - Manual: Rotate certificates live.
   - Docs: PKI & rotation policy.
3. Secret Rotation Mechanisms
   - Impl: Schedule & emergency path.
   - Unit: Rotation event tests.
   - Manual: Forced rotation drill.
   - Docs: Rotation runbook.
4. Network Policy (Micro-Segmentation)
   - Impl: Policy manager; enforcement checks.
   - Unit: Unauthorized path blocked test.
   - Manual: Attempt forbidden connection.
   - Docs: Network policy matrix.
5. PHI Redaction & Residency Reinforcement
   - Impl: Enhanced middleware sampling verify.
   - Unit: PHI field removal tests.
   - Manual: Log/audit verification.
   - Docs: PHI & residency guidelines.
6. Audit Hash-Chain Verification CLI
   - Impl: `af audit verify` command.
   - Unit: Tamper detection test.
   - Manual: Run against sample chain.
   - Docs: Forensics procedure.
7. Storage Encryption & Key Rotation
   - Impl: At-rest encryption layer.
   - Unit: Encrypted flag presence tests.
   - Manual: Key rotation simulation.
   - Docs: Key management doc.

Exit Step (Gate G8.5): Residency strict mode blocks non-approved egress; audit verify detects tamper; secret rotation & mTLS validated; PHI redaction tests pass.

Quarter 2 Exit Criteria: Spec exits G6.5 – G8.5 achieved; full workflow with memory + model gateway + cost preflight enforced end-to-end; multi-language agent (WASM) executes within budgets & residency constraints.

---

## Quarter 3 – Scale & Advanced Capabilities (Months 7–9)

### Spec Q3.1 – Advanced Observability & Deterministic Replay (Phase P9 Core)
Relies On: Q1.2 tracing, Q1.5 planners, Q1.6 worker, Q2.2 cost metrics.
Enables: Debugging, performance tuning, replay safety.

Tasks:
1. Unified Metrics Suite (agent_messages_total, tool_calls_total, planner_replans_total, latency histograms, cost ratios)
   - Impl / Unit: Metric registration tests / Manual: 30m soak / Docs: Metric catalog.
2. Distributed Replay Mode (Side-Effect Suppression Full)
   - Impl: Deterministic seeding & time-travel.
   - Unit: Repeat identical outcome tests.
   - Manual: Incident replay scenario.
   - Docs: Replay runbook.
3. Causal Tracing & Correlation Graph
   - Impl: Span links publish→consume.
   - Unit: Link presence tests.
   - Manual: Graph visualization check.
   - Docs: Trace semantics guide.
4. Interaction Graph Visualizer (API output for dashboard)
   - Impl: Graph generation + diff API.
   - Unit: Graph structure tests.
   - Manual: Large workflow visualization.
   - Docs: Graph schema.
5. Performance Profiling Hooks
   - Impl: Sampling profiler integration.
   - Unit: Profiler API tests.
   - Manual: CPU spike detection run.
   - Docs: Profiling guide.
6. Real-Time Cost Attribution (Per Distributed Transaction)
   - Impl: Correlate cost across spans.
   - Unit: Allocation correctness tests.
   - Manual: Multi-agent transaction trace.
   - Docs: Attribution methodology.

Exit Step (Gate G9): Golden trace replays consistent; metrics & graphs stable; cost attribution visible; replay forbids side-effects.

---

### Spec Q3.2 – Public APIs Finalization & SDKs (Phase P10) + Benchmarks & SLO Baseline (P10.5)
Relies On: Q1.4 initial API, Q3.1 observability.
Enables: External developer adoption & performance governance.

Tasks:
1. Final REST/WS Endpoints Completion
   - Impl: Remaining endpoints (memory erase, costs, tools audit, workflows exec).
   - Unit: Contract tests from OpenAPI.
   - Manual: WS stability test.
   - Docs: Endpoint catalog.
2. Pagination, Caching (ETags), Versioning Polish
   - Impl / Unit: Pagination correctness tests / Manual: 304 flow.
   - Docs: Pagination & caching guidance.
3. SDK Enhancements (Python/JS)
   - Impl: High-level helpers & streaming.
   - Unit: SDK unit tests & fixtures.
   - Manual: Example notebooks / scripts.
   - Docs: Quickstart, migration notes.
4. Benchmark Harness Scenarios (API, Routing, Tool, Memory, Model)
   - Impl: k6 + custom harness.
   - Unit: Config validation tests.
   - Manual: Baseline run artifact review.
   - Docs: Benchmark methodology.
5. SLO Definition & Alert Rules
   - Impl: Threshold config & Prom alerts.
   - Unit: Alert rule lint tests.
   - Manual: Inject latency regression; observe alert.
   - Docs: SLO policy.
6. Trend Dashboard & Artifact Publishing
   - Impl: CI upload pipeline.
   - Unit: Artifact presence tests.
   - Manual: Historical comparison check.
   - Docs: Reading benchmark reports.

Exit Step (G10 + G11): OpenAPI stable; SDK e2e examples succeed; benchmarks produce reproducible baselines; SLO alerts fire and recover on induced regression.

---

### Spec Q3.3 – Deployment Artifacts & Dashboard v1 (Phases P11 & P12)
Relies On: Q3.2 API stability; Q3.1 metrics.
Enables: Production operations & operational visibility.

Tasks:
1. Helm Charts (control, worker, nats, postgres, optional vector)
   - Impl / Unit: schema lint / Manual: kind install & health.
   - Docs: Chart values reference.
2. Terraform Module (AWS Reference)
   - Impl: VPC, EKS, RDS, Redis, GPU node group opt.
   - Unit: tfvalidate & terratest basics.
   - Manual: Plan/apply/destroy sandbox.
   - Docs: Infra deployment guide.
3. Air-Gapped Deployment Guide
   - Impl: Image bundle scripts.
   - Unit: Manifest integrity tests.
   - Manual: Offline install dry-run.
   - Docs: Air-gapped procedure.
4. Dashboard SPA (Health, Agents, Workflows, Traces, Costs, Audit)
   - Impl: React + WS integration.
   - Unit: Component tests.
   - Manual: Lighthouse & a11y checks.
   - Docs: Operator user guide.
5. Planner Visualizations (FSM/BT Graphs)
   - Impl: Graph rendering components.
   - Unit: Graph diff snapshot tests.
   - Manual: Complex plan view.
   - Docs: Debugging with graphs.
6. Cost & Replay Widgets
   - Impl: Real-time cost vs estimate; replay trigger.
   - Unit: Data binding tests.
   - Manual: Trigger replay from UI.
   - Docs: Replay user documentation.

Exit Step (G11 + G12): Helm + Terraform deploy healthy; dashboard pages functional (Lighthouse ≥80); replay & cost widgets accurate.

---

### Spec Q3.4 – Marketplace Connector & Security Hardening (Phases P12.5 & P13)
Relies On: Q2.1 template engine, Q3.3 deployment.
Enables: Template ecosystem & hardened enterprise posture.

Tasks:
1. Marketplace Connector (List/Search/Get + Cache + Integrity Hash)
   - Impl / Unit: Cache TTL tests / Manual: Offline cache mode.
   - Docs: Marketplace usage.
2. Signature Verification (Optional Sigstore/Cosign)
   - Impl: Signature parsing & policy.
   - Unit: Signature mismatch test.
   - Manual: Signed vs unsigned template scenario.
   - Docs: Supply chain security.
3. Secrets Provider Expansion & Rotation Logging
   - Impl: Multi-provider selection.
   - Unit: Rotation audit tests.
   - Manual: Provider failover drill.
   - Docs: Secrets operations.
4. PHI Filter Enforcement & Anomaly Alerting Enhancements
   - Impl: ML/regex enrichment.
   - Unit: Detection precision tests.
   - Manual: Simulated PHI ingress.
   - Docs: PHI policy updates.
5. Tamper-Evident Audit Export (Append-Only)
   - Impl: Export job & verify signature.
   - Unit: Integrity verification tests.
   - Manual: Forensic export run.
   - Docs: Compliance export guide.
6. mTLS + NetworkPolicy Hardening (Prod Defaults)
   - Impl: Policy templates.
   - Unit: Block unauthorized path test.
   - Manual: Pen test script check.
   - Docs: Network hardening runbook.

Exit Step (G12.5 + G13): Remote templates imported offline; signature checks enforced (when enabled); audit export integrity verified; enhanced anomaly alerts operational.

---

### Spec Q3.5 – CLI One-Command Deployment & Performance Scaling (Phases P14 & P15)
Relies On: Q3.3 deployment artifacts, Q3.2 benchmarks.
Enables: Rapid production rollout and scaling confidence.

Tasks:
1. CLI Deploy/Build/Rollback Commands
   - Impl: Terraform + Helm orchestration.
   - Unit: Dry-run plan parser tests.
   - Manual: Fresh acct deploy <1h.
   - Docs: Deploy walkthrough.
2. Rollback & DB Restore Workflow
   - Impl: Snapshot + revert automation.
   - Unit: Rollback smoke tests.
   - Manual: Induced failure rollback.
   - Docs: Rollback runbook.
3. KPI Instrumentation (dx_ttfd_seconds, deploy_ttpd_seconds)
   - Impl: Timing hooks.
   - Unit: Metric emission tests.
   - Manual: Deploy metrics present.
   - Docs: KPI definitions.
4. Scaling Policies (HPA, Queue Depth Triggers)
   - Impl: Autoscale configs.
   - Unit: Simulated load scaling tests.
   - Manual: Load test triggers scale.
   - Docs: Scaling tuning guide.
5. Performance Load & Chaos Scenarios
   - Impl: k6 + chaos scripts.
   - Unit: Scenario config validation.
   - Manual: Run: confirm SLO adherence.
   - Docs: Chaos playbook.

Exit Step (G14 + G15): `af deploy` end-to-end success; rollback proven; scaling maintains SLO (routing p95 <10ms, API p95 <50ms) under defined load; chaos tests show no data loss.

---

### Spec Q3.6 – Production Readiness & Advanced Capabilities (Phases P8.7, P8.8, P9.1, P14.5, P16)
Relies On: All prior Specs; especially Q3.5 deployment & scaling.
Enables: GA launch & advanced distributed reliability.

Tasks:
1. Distributed Transactions (Saga Orchestrator + Compensation)
   - Impl / Unit: Saga state machine tests / Manual: Failure compensation scenario.
   - Docs: Saga design & examples.
2. Event Sourcing / Replay Consistency
   - Impl: Append-only event_store + versioning.
   - Unit: Event ordering & idempotency tests.
   - Manual: Rehydrate state from events.
   - Docs: Event sourcing guide.
3. Comprehensive Chaos & Failure Injection Framework
   - Impl: Network partition, latency, crash modules.
   - Unit: Scenario enumeration coverage tests.
   - Manual: 95% planned failure scenario coverage run.
   - Docs: Chaos execution runbook.
4. Advanced Agent Communication Patterns (Gossip, Consensus)
   - Impl: Gossip dissemination + Raft/BFT module (scoped critical decisions).
   - Unit: Convergence & fault tolerance tests.
   - Manual: Partition & reconciliation demo.
   - Docs: Communication pattern selection matrix.
5. Agent Simulation & Dev Tools (Hot Reload, Visualization)
   - Impl: Simulation harness & reload pipeline.
   - Unit: Deterministic simulation tests.
   - Manual: 1000-agent simulation.
   - Docs: Simulation user guide.
6. Runbooks & Incident Management (RTO/RPO Drills)
   - Impl: Backup & restore automation.
   - Unit: Backup integrity tests.
   - Manual: DR drill within RTO/RPO targets.
   - Docs: Incident & on-call handbook.
7. Compliance Artifacts & Pen Test Remediation
   - Impl: Generate reports; track findings.
   - Unit: Artifact completeness tests.
   - Manual: Pen test fix verification.
   - Docs: Compliance checklist.
8. GA Launch Checklist & Version Tag (v1.0.0)
   - Impl: Final semantic version tag & release notes.
   - Unit: Release automation tests.
   - Manual: Final smoke post-deploy.
   - Docs: CHANGELOG & migration notes.

Exit Step (G16 + Advanced Goals): Saga + event sourcing stable; communication patterns scale to ≥1000 agents; simulation harness deterministic; DR drill passes; GA checklist signed & v1.0.0 released.

Quarter 3 Exit Criteria: All Spec exits (G9–G16) plus advanced distributed & simulation capabilities validated; production readiness sign-off completed.

---

## Dependency Chain Overview
Q1.1 → Q1.2 → (Q1.3 → Q1.4) → Q1.5 → Q1.6 → Q1.7 → Q1.8 → Q1.9 → Q1.10 → Q1.11 → Q2.1 → Q2.2 → Q2.3 → Q2.4 → Q2.5 → Q2.6 → Q2.7 → Q3.1 → Q3.2 → Q3.3 → Q3.4 → Q3.5 → Q3.6

## Traceability Mapping
- Deterministic Planning: Q1.5, validated again via replay Q3.1.
- Secure Tool Execution: Q1.7, hardened Q2.7 & Q3.4.
- Cost & Budgets: Q1.4 heuristic, Q1.9 token observations, Q1.10 variance baseline, Q2.2 hard gates, observed Q3.1 dashboards.
- Memory & Retrieval: Q2.6 baseline; performance & erasure monitored Q3.1.
- Observability & Replay: Q1.2 seed, Q1.8 baseline → Q3.1 maturity.
- Templates & Marketplace: Q2.1 local → Q3.4 remote.
- Deployment & Scaling: Q3.3 artifacts → Q3.5 automation.
- Production Hardening & GA: Q3.4 security, Q3.6 readiness.

## Quality & Testing Discipline (Applies to Every Task)
For each implementation item: (a) Unit tests (minimum 80% critical path), (b) Manual/Integration scenario validating behavior in realistic environment, (c) Documentation update (README, ADR if architectural), (d) Observability instrumentation (traces, metrics, logs) & cost tagging where applicable.

## Documentation & Manuals Matrix (Framework Deliverables)
Purpose: Ensure the framework ships with comprehensive, role-focused manuals. Each document gains an owning Spec (initial creation) and subsequent Specs that must update it. Updates are treated as required tasks (Docs subtasks already listed) but this matrix centralizes visibility.

| Manual / Doc | Primary Audience | Initial Spec (Create) | Update Triggers (Specs) | Key Contents / Acceptance | Publication Format |
|--------------|------------------|------------------------|-------------------------|---------------------------|--------------------|
| Developer Quickstart | Backend engineers | Q1.1 | Q1.2, Q1.4, Q1.5, Q1.7, Q2.1 | Clone → devcontainer → run demo plan <5 min | README + /docs/quickstart.md |
| Architecture Overview | Tech leads / architects | Q1.1 | Q1.5, Q2.6, Q3.1, Q3.6 | C4 Levels, component responsibilities, data flows | /docs/architecture.md + diagrams |
| Message & Event Contract Guide | Developers | Q1.2 | Q1.5, Q3.1 | Subjects, envelope schema, evolution rules, hash canonicalization | /docs/messaging.md |
| Database Schema Reference | Developers / DBAs | Q1.3 | Q2.6, Q2.7, Q3.6 | ERD, table purpose, migration policy, hash-chain integrity steps | /docs/storage.md |
| API Reference (OpenAPI) | External integrators | Q1.4 | Q3.2 | Versioning, pagination, ETags, error taxonomy | openapi.json + /docs/api.md |
| Security & Compliance Manual | Security, Compliance | Q1.4 (baseline) | Q2.7, Q3.4, Q3.6 | RBAC, residency, PHI redaction, secrets, audit verification, threat model | /docs/security.md |
| Planner & Workflow Authoring Guide | Workflow designers | Q1.5 | Q2.2 (cost gates), Q2.6 (memory), Q3.1 (replay) | FSM/BT syntax, composition patterns, cost preflight usage, replay tips | /docs/planning.md |
| Worker Operations Runbook | SRE / Operators | Q1.6 | Q2.2, Q3.5 | Scaling knobs, backpressure, metrics, recovery, replay-safe mode | /docs/runbooks/worker.md |
| Tool Registry & Sandbox Manual | Developers / SecOps | Q1.7 | Q2.7, Q3.4 | Tool schema, ExecProfile, permissions, anomaly signals, PHI filters | /docs/tools.md |
| Template Authoring Guide | Developers | Q2.1 | Q3.4 | Manifest schema, semantic versioning, remote marketplace usage | /docs/templates.md |
| Cost & Budget Management Guide | FinOps / Eng | Q2.2 | Q3.1, Q3.2 | Estimation, hard gates, utilization metrics, chargeback/showback | /docs/costs.md |
| Model Gateway & Residency Guide | ML / Platform | Q2.3 | Q2.7, Q3.6 | Provider routing, failover, zero egress, latency metrics | /docs/model-gateway.md |
| MCP Integration Guide | Developers | Q2.4 | Q3.4 | Handshake, permissions, auditing parity | /docs/mcp.md |
| WASM Agent Development Guide | Polyglot devs | Q2.5 | Q3.5 | Build pipelines, host functions, hot reload, perf targets | /docs/wasm-agents.md |
| Memory & Retrieval Guide | Developers / ML | Q2.6 | Q3.1 | Layers, retrieval pipeline, summarization, budgets, erasure | /docs/memory.md |
| Early Security Enforcement Addendum | Security | Q2.7 | Q3.4 | mTLS, rotation, network policy, PHI enhancements | /docs/security-addendum.md |
| Observability & Replay Runbook | SRE / Devs | Q3.1 | Q3.2, Q3.3 | Metrics catalog, tracing semantics, replay workflow | /docs/runbooks/observability.md |
| Benchmark & SLO Handbook | SRE / Mgmt | Q3.2 | Q3.5 | Scenarios, SLO thresholds, interpreting trend dashboards | /docs/benchmarks.md |
| Deployment & Air-Gapped Guide | Operators | Q3.3 | Q3.5 | Helm values, Terraform vars, offline bundles, residency validation | /docs/deployment.md |
| Dashboard User Guide | Product / Ops | Q3.3 | Q3.4 | Navigation, live workflow views, cost widgets, replay trigger | /docs/dashboard.md |
| Marketplace & Supply Chain Security | Dev / SecOps | Q3.4 | Q3.6 | Template fetching, cache integrity, signatures, offline mode | /docs/marketplace.md |
| CLI Deployment Guide | Developers / Ops | Q3.5 | Q3.6 | `af deploy` flows, rollback, KPIs, scaling flags | /docs/cli-deploy.md |
| Distributed Systems & Resilience Guide | Architects / SRE | Q3.6 | — | Saga orchestration, event sourcing, gossip/consensus, chaos practices | /docs/distributed.md |
| Production Readiness & Runbooks Index | All Ops Roles | Q3.6 | — | Runbook index, on-call rotations, DR procedures, GA checklist | /docs/runbooks/index.md |

Documentation Quality Gates:
- Every Spec exit requires relevant manuals updated (diff reviewed in PR).
- Lint: Markdown link checker + schema validation for embedded JSON/YAML examples.
- Versioned Docs: Major feature changes add CHANGELOG entries referencing doc sections.
- Discoverability: Root README links to all manuals; `af validate` surfaces doc URL hints on warnings.

---

End of Plan.
