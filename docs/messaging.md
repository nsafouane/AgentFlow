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

### Benchmark Configuration

The messaging system includes a performance harness for measuring latency:

```bash
# Run ping-pong benchmark
go test -bench=BenchmarkPingPong -benchtime=10s

# Configure test parameters
AF_BENCH_MESSAGE_COUNT=1000 \
AF_BENCH_CONCURRENCY=10 \
AF_BENCH_PAYLOAD_SIZE=1024 \
go test -bench=BenchmarkPingPong
```

### CI Thresholds

- **p95 Latency**: < 15ms (Linux CI environment)
- **Throughput**: > 1000 messages/second
- **Memory Usage**: < 100MB for 10,000 messages

### Environment-Specific Overrides

Use environment variables to adjust thresholds for different environments:

- `AF_PERF_P95_THRESHOLD_MS`: p95 latency threshold in milliseconds
- `AF_PERF_THROUGHPUT_MIN`: Minimum throughput in messages/second
- `AF_PERF_MEMORY_MAX_MB`: Maximum memory usage in MB

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