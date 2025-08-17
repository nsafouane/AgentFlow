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

## Environment Variables

### Message Bus Configuration

- `AF_BUS_URL`: NATS server URL (default: `nats://localhost:4222`)
- `AF_BUS_MAX_RECONNECT`: Maximum reconnection attempts (default: `10`)
- `AF_BUS_RECONNECT_WAIT`: Wait time between reconnections (default: `2s`)
- `AF_BUS_ACK_WAIT`: Message acknowledgment timeout (default: `30s`)
- `AF_BUS_MAX_IN_FLIGHT`: Maximum in-flight messages (default: `1000`)

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

## Trace Attribute Conventions

### Standard Attributes

When creating spans for message operations, use these attribute keys:

- `agentflow.message.id`: Message ID
- `agentflow.message.type`: Message type
- `agentflow.message.from`: Source agent
- `agentflow.message.to`: Target agent
- `agentflow.message.subject`: NATS subject
- `agentflow.message.size`: Message size in bytes
- `agentflow.workflow.id`: Workflow ID (from metadata)

### Usage Example

```go
span.SetAttributes(
    attribute.String("agentflow.message.id", msg.ID),
    attribute.String("agentflow.message.type", string(msg.Type)),
    attribute.String("agentflow.message.from", msg.From),
    attribute.String("agentflow.message.to", msg.To),
)
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