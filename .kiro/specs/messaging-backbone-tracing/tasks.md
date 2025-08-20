# Implementation Plan - Messaging Backbone & Tracing Skeleton

This implementation plan converts the messaging backbone design into a series of prompts for code-generation that will implement each step in a test-driven manner. Each task builds incrementally on previous tasks and focuses on discrete, manageable coding steps that can be executed by a coding agent.

- [x] 1. Subject Taxonomy & Message Contract v1










  - Define constants for NATS subjects (workflows.*, agents.*, tools.*, system.*)
  - Create Go structs for Message with all required fields (ID, TraceID, SpanID, From, To, Type, Payload, Metadata, Cost, Timestamp, EnvelopeHash)
  - Implement JSON schema definitions for message validation with proper error handling
  - Build canonical serializer with deterministic field ordering that produces stable SHA256 envelope_hash
  - Write unit tests for serializer determinism (stable hash across multiple serializations of same content)
  - Write unit tests for JSON schema validation with various valid/invalid message scenarios
  - Create manual test to inspect sample messages and verify backward-compatible extension scenarios work
  - Document message contract schema and evolution rules in /docs/messaging.md
  - _Requirements: 1.1, 1.2, 1.3, 1.4, 1.5, 6.1, 6.2, 6.3, 6.4, 6.5, 8.1, 8.2, 8.3, 8.4, 8.5_

- [x] 2. NATS JetStream Integration  





  - Implement NATS JetStream client with connection management and configurable URL via AF_BUS_URL environment variable
  - Create stream configurations for AF_MESSAGES, AF_TOOLS, AF_SYSTEM with appropriate retention and replica settings
  - Build publish/subscribe functionality with durable consumer setup and acknowledgment handling
  - Implement message replay capabilities that deliver messages in chronological order
  - Add retry policies with exponential backoff and jitter for connection failures
  - Write unit tests using in-memory NATS or testcontainer for pub/sub functionality, ack ordering, and replay sequence validation
  - Create manual test to measure local nats-server roundtrip latency and verify it meets performance targets
  - Document environment variables (AF_BUS_URL) and retry guidelines in /docs/messaging.md
  - _Requirements: 2.1, 2.2, 2.3, 2.4, 2.5, 7.1, 7.2, 7.3, 7.4, 7.5_

- [x] 3. OpenTelemetry Context Propagation





  - Set up OpenTelemetry tracer with Jaeger exporter configuration
  - Implement trace context injection that adds trace parent information to outgoing message headers
  - Build trace context extraction that continues distributed traces from incoming message headers  
  - Add span creation and linking for all message bus operations with appropriate attributes
  - Write unit tests for trace continuity (root span ID equality chain across message hops)
  - Create manual test to verify end-to-end traces are visible in Jaeger UI
  - Document trace attribute key conventions and usage patterns in /docs/messaging.md
  - _Requirements: 3.1, 3.2, 3.3, 3.4, 3.5_

- [x] 4. Structured Logging Baseline










  - Create structured logger wrapper that outputs JSON format with consistent field ordering
  - Implement automatic enrichment of log entries with trace_id, span_id, and message_id correlation fields
  - Add context-aware logging that preserves correlation IDs across goroutines and message processing
  - Build log field governance with reserved field validation and linting rules
  - Write unit tests for log enrichment to verify all correlation IDs are present in log entries
  - Create manual test to tail logs during ping-pong scenario and verify all required fields are present
  - Document logging standards and reserved key conventions in /docs/messaging.md
  - _Requirements: 4.1, 4.2, 4.3, 4.4, 4.5_

- [x] 5. Basic Performance Harness (Ping-Pong)





  - Build benchmark script that measures message routing p50 and p95 latency using ping-pong pattern
  - Implement configurable test parameters (message count, concurrency, payload size) with statistical analysis
  - Create CI integration that asserts p95 < 15ms threshold with environment-specific overrides
  - Add performance regression detection that fails builds when thresholds are exceeded
  - Write unit tests for performance assertion logic and threshold evaluation
  - Create manual test to run benchmark locally and record baseline performance metrics
  - Document performance harness usage, thresholds, and tuning guidelines in /docs/messaging.md
  - _Requirements: 5.1, 5.2, 5.3, 5.4, 5.5_

## Exit Criteria (Gate G1)

This implementation plan ensures all Gate G1 exit criteria are satisfied:

- **Deterministic message hashing**: Task 1 implements canonical serializer with stable SHA256 envelope_hash generation
- **Pub/sub/replay tests pass**: Task 2 implements NATS JetStream with comprehensive pub/sub and replay testing  
- **OTEL spans visible**: Task 3 implements OpenTelemetry integration with Jaeger UI verification
- **Ping-pong p95 < 15ms (CI Linux)**: Task 5 implements performance harness with CI threshold enforcement
- **Documentation published**: All tasks include documentation updates to /docs/messaging.md

## Additional Quantitative Assertions (Gate G1 Augmentation)

- **Canonical serializer property-based test**: Task 1 includes N iterations with no hash collision for permuted fields
- **Envelope_hash recomputation variance = 0**: Task 1 validates hash determinism across test matrix scenarios

Each task follows the test-driven development approach with implementation, unit tests, manual validation, and documentation components as specified in the development plan.