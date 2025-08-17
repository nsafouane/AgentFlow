package messaging

import (
	"context"
	"testing"

	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
	oteltrace "go.opentelemetry.io/otel/trace"
)

func TestTracingMiddleware_InjectTraceContext(t *testing.T) {
	// Create a test tracer with in-memory exporter
	exporter := tracetest.NewInMemoryExporter()
	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithSampler(trace.AlwaysSample()),
	)

	// Create tracing middleware
	config := &TracingConfig{
		Enabled:      true,
		OTLPEndpoint: "http://localhost:4318/v1/traces",
		ServiceName:  "test-service",
		SampleRate:   1.0,
	}

	tracing, err := NewTracingMiddlewareWithProvider(config, tp)
	if err != nil {
		t.Fatalf("Failed to create tracing middleware: %v", err)
	}

	// Create a test span
	tracer := tp.Tracer("test")
	ctx, span := tracer.Start(context.Background(), "test-span")
	defer span.End()

	// Create a test message
	msg := NewMessage("test-id", "agent1", "agent2", MessageTypeRequest)

	// Inject trace context
	tracing.InjectTraceContext(ctx, msg)

	// Verify trace context was injected
	if msg.TraceID == "" {
		t.Error("TraceID was not injected into message")
	}
	if msg.SpanID == "" {
		t.Error("SpanID was not injected into message")
	}

	// Verify trace context is in metadata
	found := false
	for key := range msg.Metadata {
		if len(key) > 6 && key[:6] == "trace." {
			found = true
			break
		}
	}
	if !found {
		t.Error("Trace context was not added to message metadata")
	}
}

func TestTracingMiddleware_ExtractTraceContext(t *testing.T) {
	// Create a test tracer with in-memory exporter
	exporter := tracetest.NewInMemoryExporter()
	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithSampler(trace.AlwaysSample()),
	)

	// Create tracing middleware
	config := &TracingConfig{
		Enabled:      true,
		OTLPEndpoint: "http://localhost:4318/v1/traces",
		ServiceName:  "test-service",
		SampleRate:   1.0,
	}

	tracing, err := NewTracingMiddlewareWithProvider(config, tp)
	if err != nil {
		t.Fatalf("Failed to create tracing middleware: %v", err)
	}

	// Create a test message with trace context
	msg := NewMessage("test-id", "agent1", "agent2", MessageTypeRequest)
	msg.TraceID = "test-trace-id"
	msg.SpanID = "test-span-id"
	msg.Metadata["trace.traceparent"] = "00-test-trace-id-test-span-id-01"

	// Extract trace context
	ctx := tracing.ExtractTraceContext(msg)

	// Verify context is not nil
	if ctx == nil {
		t.Error("Extracted context is nil")
	}

	// For this test, we just verify that extraction doesn't panic and returns a context
	// The actual trace propagation is tested in the continuity test
}

func TestTracingMiddleware_StartPublishSpan(t *testing.T) {
	// Create a test tracer with in-memory exporter
	exporter := tracetest.NewInMemoryExporter()
	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithSampler(trace.AlwaysSample()),
	)

	// Create tracing middleware
	config := &TracingConfig{
		Enabled:      true,
		OTLPEndpoint: "http://localhost:4318/v1/traces",
		ServiceName:  "test-service",
		SampleRate:   1.0,
	}

	tracing, err := NewTracingMiddlewareWithProvider(config, tp)
	if err != nil {
		t.Fatalf("Failed to create tracing middleware: %v", err)
	}

	// Create a test message
	msg := NewMessage("test-id", "agent1", "agent2", MessageTypeRequest)
	msg.AddMetadata("workflow_id", "test-workflow")
	msg.AddMetadata("agent_id", "test-agent")
	msg.SetCost(100, 0.01)

	// Start publish span
	ctx, span := tracing.StartPublishSpan(context.Background(), "test.subject", msg)
	span.End()

	// Force export of spans
	tp.ForceFlush(context.Background())

	// Verify span was created
	spans := exporter.GetSpans()
	if len(spans) == 0 {
		t.Fatal("No spans were created")
	}

	publishSpan := spans[len(spans)-1] // Get the most recent span
	if publishSpan.Name != "messaging.publish test.subject" {
		t.Errorf("Expected span name 'messaging.publish test.subject', got '%s'", publishSpan.Name)
	}

	// Verify span kind
	if publishSpan.SpanKind != oteltrace.SpanKindProducer {
		t.Errorf("Expected span kind Producer, got %v", publishSpan.SpanKind)
	}

	// Verify some key attributes exist
	attrs := publishSpan.Attributes
	foundMessageID := false
	foundSubject := false
	for _, attr := range attrs {
		if string(attr.Key) == AttrMessageID && attr.Value.AsString() == "test-id" {
			foundMessageID = true
		}
		if string(attr.Key) == AttrMessageSubject && attr.Value.AsString() == "test.subject" {
			foundSubject = true
		}
	}
	if !foundMessageID {
		t.Error("Message ID attribute not found")
	}
	if !foundSubject {
		t.Error("Message subject attribute not found")
	}

	// Verify context is not nil
	if ctx == nil {
		t.Error("Returned context is nil")
	}
}

func TestTracingMiddleware_StartConsumeSpan(t *testing.T) {
	// Create a test tracer with in-memory exporter
	exporter := tracetest.NewInMemoryExporter()
	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithSampler(trace.AlwaysSample()),
	)

	// Create tracing middleware
	config := &TracingConfig{
		Enabled:      true,
		OTLPEndpoint: "http://localhost:4318/v1/traces",
		ServiceName:  "test-service",
		SampleRate:   1.0,
	}

	tracing, err := NewTracingMiddlewareWithProvider(config, tp)
	if err != nil {
		t.Fatalf("Failed to create tracing middleware: %v", err)
	}

	// Create a test message
	msg := NewMessage("test-id", "agent1", "agent2", MessageTypeResponse)

	// Start consume span
	ctx, span := tracing.StartConsumeSpan(context.Background(), "test.subject", msg)
	span.End()

	// Force export of spans
	tp.ForceFlush(context.Background())

	// Verify span was created
	spans := exporter.GetSpans()
	if len(spans) == 0 {
		t.Fatal("No spans were created")
	}

	consumeSpan := spans[len(spans)-1] // Get the most recent span
	if consumeSpan.Name != "messaging.consume test.subject" {
		t.Errorf("Expected span name 'messaging.consume test.subject', got '%s'", consumeSpan.Name)
	}

	// Verify span kind
	if consumeSpan.SpanKind != oteltrace.SpanKindConsumer {
		t.Errorf("Expected span kind Consumer, got %v", consumeSpan.SpanKind)
	}

	// Verify context is not nil
	if ctx == nil {
		t.Error("Returned context is nil")
	}
}

func TestTracingMiddleware_StartReplaySpan(t *testing.T) {
	// Create a test tracer with in-memory exporter
	exporter := tracetest.NewInMemoryExporter()
	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithSampler(trace.AlwaysSample()),
	)

	// Create tracing middleware
	config := &TracingConfig{
		Enabled:      true,
		OTLPEndpoint: "http://localhost:4318/v1/traces",
		ServiceName:  "test-service",
		SampleRate:   1.0,
	}

	tracing, err := NewTracingMiddlewareWithProvider(config, tp)
	if err != nil {
		t.Fatalf("Failed to create tracing middleware: %v", err)
	}

	// Start replay span
	ctx, span := tracing.StartReplaySpan(context.Background(), "test-workflow")
	span.End()

	// Force export of spans
	tp.ForceFlush(context.Background())

	// Verify span was created
	spans := exporter.GetSpans()
	if len(spans) == 0 {
		t.Fatal("No spans were created")
	}

	replaySpan := spans[len(spans)-1] // Get the most recent span
	if replaySpan.Name != "messaging.replay test-workflow" {
		t.Errorf("Expected span name 'messaging.replay test-workflow', got '%s'", replaySpan.Name)
	}

	// Verify span kind
	if replaySpan.SpanKind != oteltrace.SpanKindClient {
		t.Errorf("Expected span kind Client, got %v", replaySpan.SpanKind)
	}

	// Verify workflow ID attribute
	found := false
	for _, attr := range replaySpan.Attributes {
		if string(attr.Key) == AttrWorkflowID {
			found = true
			if attr.Value.AsString() != "test-workflow" {
				t.Errorf("Expected workflow ID to be 'test-workflow', got '%s'", attr.Value.AsString())
			}
			break
		}
	}
	if !found {
		t.Error("Workflow ID attribute not found")
	}

	// Verify context is not nil
	if ctx == nil {
		t.Error("Returned context is nil")
	}
}

func TestTracingMiddleware_Disabled(t *testing.T) {
	// Create tracing middleware with tracing disabled
	config := &TracingConfig{
		Enabled:      false,
		OTLPEndpoint: "http://localhost:4318/v1/traces",
		ServiceName:  "test-service",
		SampleRate:   1.0,
	}

	tracing, err := NewTracingMiddleware(config)
	if err != nil {
		t.Fatalf("Failed to create tracing middleware: %v", err)
	}

	// Create a test message
	msg := NewMessage("test-id", "agent1", "agent2", MessageTypeRequest)

	// Test inject (should be no-op)
	tracing.InjectTraceContext(context.Background(), msg)
	if msg.TraceID != "" {
		t.Error("TraceID should not be set when tracing is disabled")
	}

	// Test extract (should return background context)
	ctx := tracing.ExtractTraceContext(msg)
	if ctx != context.Background() {
		t.Error("Should return background context when tracing is disabled")
	}

	// Test spans (should be no-op)
	ctx, span := tracing.StartPublishSpan(context.Background(), "test.subject", msg)
	span.End()
	if ctx == nil {
		t.Error("Context should not be nil even when tracing is disabled")
	}
}

func TestTraceContinuity(t *testing.T) {
	// This test verifies that trace context is properly propagated across message hops

	// Create a test tracer with in-memory exporter
	exporter := tracetest.NewInMemoryExporter()
	tp := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithSampler(trace.AlwaysSample()),
	)

	// Create tracing middleware
	config := &TracingConfig{
		Enabled:      true,
		OTLPEndpoint: "http://localhost:4318/v1/traces",
		ServiceName:  "test-service",
		SampleRate:   1.0,
	}

	tracing, err := NewTracingMiddlewareWithProvider(config, tp)
	if err != nil {
		t.Fatalf("Failed to create tracing middleware: %v", err)
	}

	// Create root span (simulating the start of a workflow)
	tracer := tp.Tracer("test")
	rootCtx, rootSpan := tracer.Start(context.Background(), "workflow-start")
	rootTraceID := rootSpan.SpanContext().TraceID().String()

	// Step 1: Agent A publishes a message
	msg1 := NewMessage("msg1", "agentA", "agentB", MessageTypeRequest)
	publishCtx, publishSpan := tracing.StartPublishSpan(rootCtx, "agents.agentB.in", msg1)
	tracing.InjectTraceContext(publishCtx, msg1)
	publishSpan.End()

	// Step 2: Agent B receives and processes the message
	extractedCtx := tracing.ExtractTraceContext(msg1)
	consumeCtx, consumeSpan := tracing.StartConsumeSpan(extractedCtx, "agents.agentB.in", msg1)

	// Step 3: Agent B publishes a response message
	msg2 := NewMessage("msg2", "agentB", "agentC", MessageTypeResponse)
	responseCtx, responseSpan := tracing.StartPublishSpan(consumeCtx, "agents.agentC.in", msg2)
	tracing.InjectTraceContext(responseCtx, msg2)
	responseSpan.End()
	consumeSpan.End()

	// Step 4: Agent C receives the response
	extractedCtx2 := tracing.ExtractTraceContext(msg2)
	finalCtx, finalSpan := tracing.StartConsumeSpan(extractedCtx2, "agents.agentC.in", msg2)
	finalSpan.End()

	rootSpan.End()

	// Force export of spans
	tp.ForceFlush(context.Background())

	// Verify all spans belong to the same trace
	spans := exporter.GetSpans()
	if len(spans) < 4 {
		t.Fatalf("Expected at least 4 spans, got %d", len(spans))
	}

	// All spans should have the same trace ID
	for i, span := range spans {
		spanTraceID := span.SpanContext.TraceID().String()
		if spanTraceID != rootTraceID {
			t.Errorf("Span %d has different trace ID: expected %s, got %s", i, rootTraceID, spanTraceID)
		}
	}

	// Verify trace context was properly injected into messages
	if msg1.TraceID != rootTraceID {
		t.Errorf("Message 1 trace ID mismatch: expected %s, got %s", rootTraceID, msg1.TraceID)
	}
	if msg2.TraceID != rootTraceID {
		t.Errorf("Message 2 trace ID mismatch: expected %s, got %s", rootTraceID, msg2.TraceID)
	}

	// Verify contexts are properly linked
	if rootCtx == nil || publishCtx == nil || consumeCtx == nil || responseCtx == nil || finalCtx == nil {
		t.Error("One or more contexts are nil")
	}
}

func TestDefaultTracingConfig(t *testing.T) {
	config := DefaultTracingConfig()

	if !config.Enabled {
		t.Error("Default config should have tracing enabled")
	}
	if config.OTLPEndpoint != "http://localhost:4318" {
		t.Errorf("Expected default OTLP endpoint 'http://localhost:4318', got '%s'", config.OTLPEndpoint)
	}
	if config.ServiceName != "agentflow-messaging" {
		t.Errorf("Expected default service name 'agentflow-messaging', got '%s'", config.ServiceName)
	}
	if config.SampleRate != 1.0 {
		t.Errorf("Expected default sample rate 1.0, got %f", config.SampleRate)
	}
}
