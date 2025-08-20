# AgentFlow Messaging System

This document describes the AgentFlow messaging system, including the message contract, subject taxonomy, serialization format, and schema evolution rules.

## Message Contract

### Message Structure

All AgentFlow messages follow a standardized structure with the following fields:

```go
type Message struct {
    ID           string                 `json:"id"`            // ULID identifier
    TraceID      string                 `json:"trace_id"`      // OpenTelemetry trace ID
    SpanID       string                 `json:"span_id"`       // OpenTelemetry span ID
    From         string                 `json:"from"`          // Source agent ID
    To           string                 `json:"to"`            // Target agent ID or topic
    Type         MessageType            `json:"type"`          // Message type
    Payload      interface{}            `json:"payload"`       // Message-specific data
    Metadata     map[string]interface{} `json:"metadata"`      // Workflow context
    Cost         CostInfo               `json:"cost"`          // Token/dollar tracking
    Timestamp    time.Time              `json:"ts"`            // RFC3339 timestamp
    EnvelopeHash string                 `json:"envelope_hash"` // SHA256 of canonical content
}
```

### Required Fields

The following fields are required for all messages:
- `id`: ULID identifier following the format `[0-9A-HJKMNP-TV-Z]{26}`
- `from`: Source agent identifier (minimum 1 character)
- `to`: Target agent identifier or topic (minimum 1 character)
- `type`: Message type (one of: `request`, `response`, `event`, `control`)
- `ts`: RFC3339 timestamp

### Optional Fields

- `trace_id`: OpenTelemetry trace ID (32-character hex string)
- `span_id`: OpenTelemetry span ID (16-character hex string)
- `payload`: Message-specific data (any valid JSON)
- `metadata`: Workflow context (JSON object)
- `cost`: Token and dollar cost tracking
- `envelope_hash`: SHA256 hash of canonical message content (64-character hex string)

### Message Types

- **request**: Request for action or information
- **response**: Response to a previous request
- **event**: Notification of an event or state change
- **control**: System control or administrative message

### Cost Tracking

The `cost` field tracks resource consumption:

```go
type CostInfo struct {
    Tokens  int     `json:"tokens"`  // Token count (≥ 0)
    Dollars float64 `json:"dollars"` // Dollar amount (≥ 0.0)
}
```

## Subject Taxonomy

AgentFlow uses a hierarchical subject taxonomy for NATS message routing:

### Workflow Subjects
- `workflows.<workflow_id>.in` - Inbound workflow messages
- `workflows.<workflow_id>.out` - Outbound workflow messages

### Agent Subjects
- `agents.<agent_id>.in` - Agent-specific inbound messages
- `agents.<agent_id>.out` - Agent-specific outbound messages

### Tool Subjects
- `tools.calls` - Tool execution requests
- `tools.audit` - Tool audit events

### System Subjects
- `system.control` - System control messages
- `system.health` - Health check messages

### Subject Builder

Use the `SubjectBuilder` utility for constructing subjects:

```go
builder := messaging.NewSubjectBuilder()
subject := builder.WorkflowIn("workflow-123") // "workflows.workflow-123.in"
```

## Canonical Serialization

### Deterministic Ordering

AgentFlow uses canonical serialization to ensure deterministic message hashing:

1. **Field Ordering**: All object fields are sorted alphabetically by key
2. **Recursive Sorting**: Nested objects and maps are recursively sorted
3. **Stable Arrays**: Array elements maintain their original order
4. **Consistent Formatting**: JSON output uses consistent formatting without extra whitespace

### Envelope Hash

The `envelope_hash` field contains a SHA256 hash of the canonical message content:

1. Create a copy of the message with `envelope_hash` set to empty string
2. Serialize the message using canonical ordering
3. Compute SHA256 hash of the serialized bytes
4. Store the hex-encoded hash in the `envelope_hash` field

### Hash Verification

To verify message integrity:

```go
serializer, _ := messaging.NewCanonicalSerializer()
err := serializer.ValidateHash(message)
if err != nil {
    // Message has been tampered with
}
```

## Schema Evolution Rules

### Backward Compatibility

1. **Required Fields**: Never remove or rename required fields
2. **Optional Fields**: New optional fields can be added to `metadata`
3. **Field Types**: Never change the type of existing fields
4. **Enum Values**: Never remove existing enum values from `type` field

### Forward Compatibility

1. **Unknown Fields**: Parsers must reject messages with unknown top-level fields
2. **Metadata Extension**: Use the `metadata` field for extensibility
3. **Version Indication**: Include schema version in metadata when needed

### Schema Versioning

When significant changes are needed:

1. Add `schema_version` to message metadata
2. Maintain backward compatibility for at least one major version
3. Document migration paths in this file
4. Update validation schemas accordingly

### Example Evolution

```json
{
  "id": "01HN8ZQJKM9XVQZJKM9XVQZJKM",
  "from": "agent-1",
  "to": "agent-2",
  "type": "request",
  "ts": "2024-01-01T12:00:00Z",
  "metadata": {
    "schema_version": "1.1",
    "new_feature": "enabled",
    "compatibility_mode": "backward"
  }
}
```

## NATS JetStream Integration

### Stream Configuration

AgentFlow uses three NATS JetStream streams for message persistence:

#### AF_MESSAGES Stream
- **Subjects**: `workflows.*.*`, `agents.*.*`
- **Storage**: File storage
- **Retention**: 7 days (168 hours)
- **Max Size**: 10GB
- **Replicas**: 1 (configurable)

#### AF_TOOLS Stream
- **Subjects**: `tools.*`
- **Storage**: File storage
- **Retention**: 30 days (720 hours)
- **Max Size**: 5GB
- **Replicas**: 1 (configurable)

#### AF_SYSTEM Stream
- **Subjects**: `system.*`
- **Storage**: File storage
- **Retention**: 1 day (24 hours)
- **Max Size**: 1GB
- **Replicas**: 1 (configurable)

### Consumer Configuration

Durable consumers are created automatically with the following settings:

- **Delivery Policy**: Deliver all messages
- **Ack Policy**: Explicit acknowledgment required
- **Replay Policy**: Instant replay
- **Max In-Flight**: Configurable per consumer

### Message Replay

Messages can be replayed in chronological order:

```go
bus, _ := messaging.NewNATSBus(config)
messages, err := bus.Replay(ctx, "workflow-123", time.Now().Add(-1*time.Hour))
```

### Connection Retry

The NATS client implements exponential backoff with jitter:

- **Base Delay**: 2 seconds
- **Max Delay**: 30 seconds
- **Jitter**: ±25% of calculated delay
- **Max Attempts**: Configurable (default: 10)

## Environment Variables

### Message Bus Configuration

- `AF_BUS_URL`: NATS server URL (default: `nats://localhost:4222`)
- `AF_BUS_MAX_RECONNECT`: Maximum reconnection attempts (default: `10`)
- `AF_BUS_RECONNECT_WAIT`: Wait time between reconnections (default: `2s`)
- `AF_BUS_ACK_WAIT`: Message acknowledgment timeout (default: `30s`)
- `AF_BUS_MAX_IN_FLIGHT`: Maximum in-flight messages (default: `1000`)
- `AF_BUS_CONNECT_TIMEOUT`: Connection timeout (default: `5s`)
- `AF_BUS_REQUEST_TIMEOUT`: Request timeout (default: `10s`)

## Performance Guidelines

### Message Size

- Keep message payloads under 1MB for optimal performance
- Use references or external storage for large data
- Consider compression for large payloads

### Latency Targets

- Message routing: p95 < 15ms
- Serialization: p95 < 1ms
- Hash computation: p95 < 0.5ms

### Retry Policies

- Use exponential backoff with jitter
- Maximum retry attempts: 3-5 depending on message type
- Implement circuit breakers for failing consumers

## OpenTelemetry Tracing Integration

### Trace Context Propagation

AgentFlow automatically propagates OpenTelemetry trace context across message boundaries:

1. **Injection**: Outgoing messages receive trace context in headers and message fields
2. **Extraction**: Incoming messages continue the distributed trace
3. **Span Creation**: All message operations create appropriate spans
4. **Correlation**: Messages are correlated with their originating traces

### Trace Attribute Conventions

#### Standard Message Attributes

Use these attribute keys for message-related spans:

- `messaging.system`: Always set to "nats"
- `messaging.destination.name`: NATS subject name
- `messaging.message.id`: Message ID (ULID)
- `messaging.message.type`: Message type (request/response/event/control)
- `messaging.message.from`: Source agent identifier
- `messaging.message.to`: Target agent identifier

#### AgentFlow-Specific Attributes

- `agentflow.workflow.id`: Workflow identifier from metadata
- `agentflow.agent.id`: Agent identifier from metadata
- `agentflow.message.envelope_hash`: Message integrity hash
- `agentflow.message.cost.tokens`: Token cost for processing
- `agentflow.message.cost.dollars`: Dollar cost for processing

### Span Naming Conventions

- **Publish Operations**: `messaging.publish <subject>`
- **Consume Operations**: `messaging.consume <subject>`
- **Replay Operations**: `messaging.replay <workflow_id>`

### Usage Examples

#### Creating Spans with Attributes

```go
// Start a publish span
ctx, span := tracing.StartPublishSpan(ctx, "agents.classifier.in", message)
defer span.End()

// Add custom attributes
span.SetAttributes(
    attribute.String("agentflow.workflow.id", workflowID),
    attribute.String("agentflow.agent.id", agentID),
    attribute.Int("agentflow.message.cost.tokens", 150),
    attribute.Float64("agentflow.message.cost.dollars", 0.015),
)

// Record errors
if err != nil {
    span.RecordError(err)
    span.SetStatus(codes.Error, err.Error())
}
```

#### Trace Context Injection and Extraction

```go
// Publishing side - inject trace context
tracing.InjectTraceContext(ctx, message)
err := bus.Publish(ctx, subject, message)

// Consuming side - extract trace context
extractedCtx := tracing.ExtractTraceContext(message)
consumeCtx, span := tracing.StartConsumeSpan(extractedCtx, subject, message)
defer span.End()

// Process message with trace context
err := processMessage(consumeCtx, message)
```

### Tracing Configuration

#### Environment Variables

- `AF_TRACING_ENABLED`: Enable/disable tracing (default: `true`)
- `AF_OTEL_EXPORTER_OTLP_ENDPOINT`: OTLP endpoint URL (default: `http://localhost:4318`)
- `AF_SERVICE_NAME`: Service name for traces (default: `agentflow-messaging`)
- `AF_TRACE_SAMPLE_RATE`: Sampling rate 0.0-1.0 (default: `1.0`)

#### Configuration Example

```go
config := &TracingConfig{
    Enabled:      true,
    OTLPEndpoint: "http://jaeger:4318",
    ServiceName:  "agentflow-worker",
    SampleRate:   0.1, // Sample 10% of traces in production
}

tracing, err := NewTracingMiddleware(config)
```

### Trace Continuity Verification

To verify trace continuity across message hops:

1. **Root Span**: Create a root span for the workflow
2. **Message Chain**: Each message should continue the same trace
3. **Span Hierarchy**: Spans should form a proper parent-child relationship
4. **Trace ID**: All spans in a workflow should share the same trace ID

#### Example Trace Structure

```
customer-support-workflow (root)
├── messaging.publish agents.classifier.in
├── messaging.consume agents.classifier.in
│   └── messaging.publish agents.knowledge-base.in
├── messaging.consume agents.knowledge-base.in
│   └── messaging.publish agents.response-generator.in
├── messaging.consume agents.response-generator.in
│   └── messaging.publish agents.customer-portal.out
└── messaging.consume agents.customer-portal.out
```

### Jaeger UI Verification

To verify traces in Jaeger:

1. **Access UI**: Open http://localhost:16686
2. **Select Service**: Choose your service name
3. **Search Traces**: Use trace ID or time range
4. **Verify Structure**: Check span hierarchy and timing
5. **Check Attributes**: Verify all required attributes are present

#### Expected Trace Attributes

For each messaging span, verify these attributes exist:

- `messaging.system` = "nats"
- `messaging.destination.name` = subject name
- `messaging.message.id` = message ULID
- `messaging.message.type` = message type
- `agentflow.workflow.id` = workflow identifier (if applicable)

### Performance Considerations

#### Sampling

- **Development**: Use 100% sampling (`SampleRate: 1.0`)
- **Production**: Use lower sampling rates (`SampleRate: 0.1` or less)
- **High Volume**: Consider adaptive sampling based on error rates

## Structured Logging Baseline

AgentFlow now provides a structured JSON logging baseline that ensures consistent, correlatable logs across messaging operations. Key points:

- Implementation: `internal/logging/logger.go` (CorrelatedLogger with `NewLoggerWithWriter()`)
- Correlation fields: `trace_id`, `span_id`, `message_id`, `workflow_id`, `agent_id` are automatically enriched from context and preserved across goroutines and message boundaries.
- Field governance: Reserved fields are validated to prevent accidental overrides (`trace_id`, `span_id`, `message_id`, `workflow_id`, `agent_id`, `timestamp`, `level`, `message`).
- Integration: Structured logging is integrated across messaging operations in `pkg/messaging/nats_bus.go` (publish, consume, replay).
- Tests: Unit and integration tests live in `pkg/messaging/logging_integration_test.go` and context-preservation tests in `pkg/messaging/ping_pong_manual_test.go` (manual). All tests passing as of 2025-08-17.

Quick verification

1. Run the logging integration tests:

  go test ./pkg/messaging -run TestStructuredLoggingIntegration

2. Run the manual ping-pong test to observe logs during message exchanges:

  go test ./pkg/messaging -run TestPingPongManual -v

3. Tail logs produced by `af` or worker binaries and filter by `trace_id` to see correlated events.

Notes

- Logs are emitted as compact JSON objects to enable efficient ingestion by log aggregators.
- The logger enforces deterministic field ordering to make envelope-level comparisons and reduce noise in diffs.
- For production deployments, configure log output destination via `NewLoggerWithWriter()` or the consuming service's logging configuration.

#### Span Attributes

- **Essential Only**: Include only essential attributes to reduce overhead
- **Batch Operations**: Use batch span processors for better performance
- **Resource Limits**: Set appropriate memory and CPU limits for trace exporters

### Troubleshooting Tracing

#### Common Issues

1. **Missing Traces**: Check OTLP endpoint configuration
2. **Broken Continuity**: Verify trace context injection/extraction
3. **High Overhead**: Reduce sampling rate or attribute count
4. **Export Failures**: Check network connectivity to trace backend

#### Debug Commands

```bash
# Test OTLP endpoint connectivity
curl -X POST http://localhost:4318/v1/traces \
  -H "Content-Type: application/json" \
  -d '{"resourceSpans":[]}'

# Enable trace debugging
AF_OTEL_LOG_LEVEL=debug go run main.go

# Verify trace export
AF_OTEL_EXPORTER_OTLP_TRACES_ENDPOINT=http://localhost:4318/v1/traces \
  go test -v -run TestTraceContinuity
```

## Logging Standards

### Reserved Log Fields

The following fields are reserved for correlation and must not be used for other purposes:

- `trace_id`: OpenTelemetry trace ID
- `span_id`: OpenTelemetry span ID  
- `message_id`: Message ID
- `workflow_id`: Workflow ID
- `agent_id`: Agent ID
- `timestamp`: Log timestamp
- `level`: Log level
- `msg`: Log message

### Log Enrichment

All log entries related to message processing are automatically enriched with:

```json
{
  "trace_id": "abcdef1234567890abcdef1234567890",
  "span_id": "1234567890abcdef",
  "message_id": "01HN8ZQJKM9XVQZJKM9XVQZJKM",
  "workflow_id": "wf-123",
  "agent_id": "agent-1",
  "timestamp": "2024-01-01T12:00:00.123456789Z",
  "level": "info",
  "msg": "Message processed successfully"
}
```

## Performance Harness

AgentFlow includes a comprehensive performance harness for measuring message routing latency and throughput using a ping-pong pattern. The harness provides configurable test parameters, statistical analysis, and CI integration with threshold enforcement.

### Performance Configuration

The performance harness supports extensive configuration:

```go
config := &PerformanceConfig{
    MessageCount:    1000,                    // Number of messages to send
    Concurrency:     10,                      // Number of concurrent senders
    PayloadSize:     1024,                    // Message payload size in bytes
    WarmupMessages:  100,                     // Warmup messages (excluded from stats)
    TestDuration:    60 * time.Second,        // Maximum test duration
    P95Threshold:    15 * time.Millisecond,   // P95 latency threshold
    P50Threshold:    5 * time.Millisecond,    // P50 latency threshold
    EnableTracing:   false,                   // Enable OpenTelemetry tracing
    Subject:         "test.performance",      // NATS subject for testing
    ReportInterval:  5 * time.Second,         // Progress reporting interval
}
```

### Running Performance Tests

#### Benchmark Tests

```bash
# Run standard ping-pong benchmark
go test -bench=BenchmarkPingPongLatency -benchtime=10s ./pkg/messaging

# Run concurrency benchmarks
go test -bench=BenchmarkPingPongConcurrency -benchtime=5s ./pkg/messaging

# Run payload size benchmarks
go test -bench=BenchmarkPingPongPayloadSize -benchtime=5s ./pkg/messaging
```

#### Threshold Tests

```bash
# Run performance threshold validation
go test -v ./pkg/messaging -run TestPerformanceThresholds

# Run with custom thresholds
AF_PERF_P95_THRESHOLD_MS=20 \
AF_PERF_P50_THRESHOLD_MS=8 \
go test -v ./pkg/messaging -run TestPerformanceThresholds
```

#### Manual Performance Testing

```bash
# Run comprehensive manual benchmark
go test -tags=manual -v ./pkg/messaging -run TestManualPerformanceBenchmark

# Record performance baseline
go test -tags=manual -v ./pkg/messaging -run TestManualBaselineRecording
```

### CI Integration

The performance harness integrates with CI/CD pipelines through dedicated scripts:

#### Linux/macOS
```bash
# Run CI performance tests
./scripts/test-performance.sh

# Run with custom thresholds
AF_PERF_P95_THRESHOLD_MS=25 ./scripts/test-performance.sh

# Run local performance tests
AF_PERFORMANCE_MODE=local ./scripts/test-performance.sh

# Record baseline metrics
AF_PERFORMANCE_MODE=baseline ./scripts/test-performance.sh
```

#### Windows
```powershell
# Run CI performance tests
.\scripts\test-performance.ps1

# Run with custom thresholds
.\scripts\test-performance.ps1 -P95ThresholdMs 25

# Run local performance tests
.\scripts\test-performance.ps1 -Mode local

# Record baseline metrics
.\scripts\test-performance.ps1 -Mode baseline
```

### Performance Metrics

The harness measures and reports comprehensive performance statistics:

#### Latency Metrics
- **Min Latency**: Fastest message roundtrip time
- **Average Latency**: Mean roundtrip time across all messages
- **P50 Latency**: 50th percentile (median) latency
- **P95 Latency**: 95th percentile latency
- **P99 Latency**: 99th percentile latency
- **Max Latency**: Slowest message roundtrip time

#### Throughput Metrics
- **Messages/Second**: Total throughput in messages per second
- **Completion Rate**: Percentage of messages successfully processed
- **Error Rate**: Percentage of messages that failed processing

#### Error Tracking
- **Publish Errors**: Failed message publications
- **Consume Errors**: Failed message consumption
- **Timeout Errors**: Messages that exceeded timeout thresholds

#### Latency Distribution
The harness provides histogram buckets for latency analysis:
- `<1ms`: Sub-millisecond responses
- `1-5ms`: Fast responses
- `5-10ms`: Normal responses
- `10-15ms`: Acceptable responses
- `15-25ms`: Slow responses
- `25-50ms`: Very slow responses
- `50-100ms`: Concerning responses
- `>100ms`: Unacceptable responses

### Performance Thresholds

#### Default Thresholds
- **P95 Latency**: < 15ms (Linux CI environment)
- **P50 Latency**: < 5ms (Linux CI environment)
- **Throughput**: > 100 messages/second
- **Error Rate**: < 1%
- **Completion Rate**: > 99%

#### Environment-Specific Overrides

Use environment variables to adjust thresholds for different environments:

```bash
# Latency thresholds
export AF_PERF_P95_THRESHOLD_MS=25    # P95 latency threshold in milliseconds
export AF_PERF_P50_THRESHOLD_MS=10    # P50 latency threshold in milliseconds

# Throughput thresholds
export AF_PERF_MIN_THROUGHPUT=50      # Minimum throughput in messages/second

# Test configuration
export AF_SKIP_PERFORMANCE=false      # Skip performance tests entirely
export AF_PERFORMANCE_MODE=ci         # Test mode: ci, local, or baseline
```

### Performance Scenarios

The harness includes multiple test scenarios:

#### Baseline Scenario
- **Purpose**: Standard performance measurement
- **Configuration**: 1000 messages, 10 concurrent senders, 1KB payload
- **Thresholds**: P95 < 15ms, P50 < 5ms

#### High Concurrency Scenario
- **Purpose**: Test performance under high concurrent load
- **Configuration**: 2000 messages, 50 concurrent senders, 1KB payload
- **Thresholds**: P95 < 25ms, P50 < 10ms (relaxed for concurrency)

#### Large Payload Scenario
- **Purpose**: Test performance with large message payloads
- **Configuration**: 500 messages, 5 concurrent senders, 16KB payload
- **Thresholds**: P95 < 30ms, P50 < 15ms (relaxed for payload size)

#### Tracing Overhead Scenario
- **Purpose**: Measure OpenTelemetry tracing overhead
- **Configuration**: 500 messages, 5 concurrent senders, 1KB payload, tracing enabled
- **Thresholds**: P95 < 20ms, P50 < 8ms (accounts for tracing overhead)

### Regression Detection

The performance harness supports regression detection by comparing current results with historical baselines:

#### Baseline Recording
```bash
# Record new baseline
go test -tags=manual -v ./pkg/messaging -run TestManualBaselineRecording
```

#### Regression Testing
```bash
# Run regression detection
go test -v ./pkg/messaging -run TestPerformanceRegression
```

#### Regression Criteria
- **P95 Degradation**: > 10% increase from baseline
- **Throughput Degradation**: > 10% decrease from baseline
- **Error Rate Increase**: > 1% absolute increase

### Performance Tuning Guidelines

#### Message Size Optimization
- Keep message payloads under 1MB for optimal performance
- Use message references for large data instead of embedding
- Consider payload compression for large messages

#### Concurrency Tuning
- Optimal concurrency depends on system resources and NATS configuration
- Start with 10-20 concurrent senders and adjust based on results
- Monitor CPU and memory usage during high concurrency tests

#### NATS Configuration
- Increase `max_in_flight` for higher throughput scenarios
- Adjust `ack_wait` timeout based on processing requirements
- Configure appropriate stream retention and storage settings

#### System Optimization
- Ensure adequate CPU and memory resources
- Use SSD storage for NATS file storage
- Configure network buffers for high throughput scenarios

### Troubleshooting Performance Issues

#### High Latency
1. Check NATS server resource usage
2. Verify network connectivity and latency
3. Review message payload sizes
4. Check for CPU or memory constraints

#### Low Throughput
1. Increase concurrency levels
2. Optimize message serialization
3. Review NATS stream configuration
4. Check for network bandwidth limitations

#### High Error Rates
1. Review NATS server logs
2. Check connection stability
3. Verify message format compliance
4. Monitor resource exhaustion

### Performance Monitoring

#### Continuous Monitoring
- Run performance tests in CI/CD pipelines
- Set up alerts for threshold violations
- Track performance trends over time
- Monitor production message latencies

#### Metrics Collection
- Export performance results to monitoring systems
- Create dashboards for performance visualization
- Set up automated regression detection
- Track performance across different environments

### Example Performance Report

```markdown
# Performance Test Report

**Date:** 2025-08-19 15:30:00 UTC
**Mode:** ci
**Commit:** abc123def456
**Branch:** main

## Configuration
- P95 Threshold: 15ms
- P50 Threshold: 5ms
- Min Throughput: 100 msg/sec

## Test Results

### Baseline Scenario
- Messages: 1000 sent, 1000 received (100.0% completion)
- Duration: 8.5s
- Throughput: 117.6 msg/sec
- Latency P50: 4.2ms ✅
- Latency P95: 12.8ms ✅
- Overall: ✅ PASSED

### High Concurrency Scenario
- Messages: 2000 sent, 1998 received (99.9% completion)
- Duration: 15.2s
- Throughput: 131.4 msg/sec
- Latency P50: 8.7ms ✅
- Latency P95: 23.1ms ✅
- Overall: ✅ PASSED
```

## Troubleshooting

### Common Issues

1. **Hash Mismatch**: Message content was modified after hash computation
2. **Validation Failure**: Message doesn't conform to JSON schema
3. **Serialization Error**: Invalid data types in payload or metadata
4. **Subject Mismatch**: Message sent to wrong NATS subject

### Debug Tools

```go
// Enable debug logging
serializer.SetDebugMode(true)

// Inspect message structure
fmt.Println(messaging.PrettyPrintMessage(msg))

// Validate message manually
validator, _ := messaging.NewMessageValidator()
err := validator.Validate(msg)
```

### Performance Debugging

```bash
# Profile message serialization
go test -cpuprofile=cpu.prof -bench=BenchmarkSerialize

# Memory profiling
go test -memprofile=mem.prof -bench=BenchmarkSerialize

# Trace analysis
go tool trace trace.out
```

## Migration Guide

### From Version 1.0 to 1.1

1. No breaking changes in core message structure
2. New optional fields added to metadata
3. Enhanced validation rules for trace IDs
4. Improved error messages in schema validation

### Upgrade Checklist

- [ ] Update message validation schemas
- [ ] Test backward compatibility with existing messages
- [ ] Update documentation and examples
- [ ] Run performance benchmarks
- [ ] Verify trace continuity across versions

## Structured Logging Standards

AgentFlow uses structured JSON logging with automatic correlation field enrichment to enable efficient troubleshooting and distributed tracing across all messaging operations.

### Log Format

All log entries are output in JSON format with consistent field ordering:

```json
{
  "timestamp": "2025-08-17T17:10:20.7927991Z",
  "level": "info",
  "message": "Starting ping-pong workflow",
  "trace_id": "4bf92f3577b34da6a3ce929d0e0e4736",
  "span_id": "00f067aa0ba902b7",
  "message_id": "msg-123",
  "workflow_id": "ping-pong-001",
  "agent_id": "orchestrator",
  "custom_field": "custom_value"
}
```

### Reserved Field Names

The following field names are reserved and cannot be overridden by application code:

- `timestamp`: RFC3339 timestamp of the log entry
- `level`: Log level (debug, info, warn, error)
- `message`: Human-readable log message
- `trace_id`: OpenTelemetry trace ID for distributed tracing
- `span_id`: OpenTelemetry span ID for distributed tracing
- `message_id`: AgentFlow message identifier for correlation
- `workflow_id`: Workflow identifier for correlation
- `agent_id`: Agent identifier for correlation

### Correlation Fields

AgentFlow automatically enriches log entries with correlation identifiers:

#### Trace Context
- `trace_id`: Extracted from OpenTelemetry trace context
- `span_id`: Extracted from OpenTelemetry span context

#### Message Context
- `message_id`: Set when logging within message processing context
- `workflow_id`: Set when logging within workflow context
- `agent_id`: Set when logging within agent context

### Log Levels

- **debug**: Detailed information for debugging purposes
- **info**: General information about system operation
- **warn**: Warning conditions that don't prevent operation
- **error**: Error conditions that may affect operation

### Usage Examples

#### Basic Logging
```go
logger := logging.NewLogger()
logger.Info("Operation completed successfully")
logger.Error("Failed to process request", err)
```

#### Context-Aware Logging
```go
// With trace context
tracedLogger := logger.WithTrace(ctx)
tracedLogger.Info("Processing message")

// With message correlation
messageLogger := logger.WithMessage("msg-123")
messageLogger.Info("Message received")

// With workflow context
workflowLogger := logger.WithWorkflow("workflow-456").WithAgent("agent-789")
workflowLogger.Info("Workflow step completed")
```

#### Chained Context
```go
correlatedLogger := logger.
    WithTrace(ctx).
    WithMessage(msg.ID).
    WithWorkflow("workflow-123").
    WithAgent("agent-456")

correlatedLogger.Info("Processing message", 
    logging.String("operation", "validate"),
    logging.Int("attempt", 1),
)
```

### Field Validation Rules

1. **Reserved Fields**: Cannot use reserved field names in custom fields
2. **Field Keys**: Must be non-empty and not contain only whitespace
3. **Consistent Ordering**: Standard fields appear first, followed by custom fields in alphabetical order
4. **JSON Compatibility**: All field values must be JSON-serializable

### Integration with Messaging

The logging system automatically integrates with the messaging layer:

- Message publishing/consuming operations include trace context
- Message IDs are automatically added to log correlation
- Workflow and agent context is preserved across message boundaries
- Error conditions include message details for debugging

### Performance Considerations

- Log entries are formatted with consistent field ordering for efficient parsing
- Correlation fields are cached to avoid repeated context extraction
- JSON marshaling uses optimized ordering to reduce serialization overhead
- Reserved field validation occurs at logger creation time, not per log entry

### Monitoring and Alerting

Structured logs enable:
- Correlation of events across distributed components using `trace_id`
- Workflow-level debugging using `workflow_id` and `message_id`
- Agent-specific troubleshooting using `agent_id`
- Performance monitoring through timestamp analysis
- Error rate tracking by log level and component
