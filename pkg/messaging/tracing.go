package messaging

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
	oteltrace "go.opentelemetry.io/otel/trace"
)

const (
	// Trace attribute keys following OpenTelemetry semantic conventions
	AttrMessageID      = "messaging.message.id"
	AttrMessageType    = "messaging.message.type"
	AttrMessageFrom    = "messaging.message.from"
	AttrMessageTo      = "messaging.message.to"
	AttrMessageSubject = "messaging.destination.name"
	AttrMessageSystem  = "messaging.system"
	AttrWorkflowID     = "agentflow.workflow.id"
	AttrAgentID        = "agentflow.agent.id"
	AttrMessageCost    = "agentflow.message.cost"
	AttrEnvelopeHash   = "agentflow.message.envelope_hash"
)

// TracingConfig holds configuration for OpenTelemetry tracing
type TracingConfig struct {
	Enabled      bool    `env:"AF_TRACING_ENABLED"`
	OTLPEndpoint string  `env:"AF_OTEL_EXPORTER_OTLP_ENDPOINT"`
	ServiceName  string  `env:"AF_SERVICE_NAME"`
	SampleRate   float64 `env:"AF_TRACE_SAMPLE_RATE"`
}

// DefaultTracingConfig returns default tracing configuration
func DefaultTracingConfig() *TracingConfig {
	return &TracingConfig{
		Enabled:      true,
		OTLPEndpoint: "http://localhost:4318", // OTLP HTTP endpoint for Jaeger
		ServiceName:  "agentflow-messaging",
		SampleRate:   1.0, // Sample all traces in development
	}
}

// TracingMiddleware provides OpenTelemetry tracing integration for messaging
type TracingMiddleware struct {
	tracer     oteltrace.Tracer
	propagator propagation.TextMapPropagator
	config     *TracingConfig
}

// NewTracingMiddleware creates a new tracing middleware with OTLP exporter
func NewTracingMiddleware(config *TracingConfig) (*TracingMiddleware, error) {
	return NewTracingMiddlewareWithProvider(config, nil)
}

// NewTracingMiddlewareWithProvider creates a new tracing middleware with custom trace provider (for testing)
func NewTracingMiddlewareWithProvider(config *TracingConfig, tp *trace.TracerProvider) (*TracingMiddleware, error) {
	if config == nil {
		config = DefaultTracingConfig()
	}

	// Apply environment variable overrides
	applyTracingEnvConfig(config)

	if !config.Enabled {
		// Return a no-op middleware
		return &TracingMiddleware{
			tracer:     otel.Tracer("agentflow-messaging-noop"),
			propagator: otel.GetTextMapPropagator(),
			config:     config,
		}, nil
	}

	// Use provided trace provider or create a new one
	if tp == nil {
		// Parse OTLP endpoint to handle full URLs correctly
		endpoint, urlPath := parseOTLPEndpoint(config.OTLPEndpoint)
		
		// Create OTLP HTTP exporter with proper endpoint and path handling
		opts := []otlptracehttp.Option{
			otlptracehttp.WithEndpoint(endpoint),
			otlptracehttp.WithInsecure(), // Use HTTP instead of HTTPS for local development
		}
		if urlPath != "" {
			opts = append(opts, otlptracehttp.WithURLPath(urlPath))
		}
		
		exporter, err := otlptracehttp.New(context.Background(), opts...)
		if err != nil {
			return nil, fmt.Errorf("failed to create OTLP exporter: %w", err)
		}

		// Create resource with service information
		res, err := resource.New(context.Background(),
			resource.WithAttributes(
				semconv.ServiceName(config.ServiceName),
				semconv.ServiceVersion("1.0.0"),
				attribute.String("messaging.system", "nats"),
			),
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create resource: %w", err)
		}

		// Create trace provider with sampling
		tp = trace.NewTracerProvider(
			trace.WithBatcher(exporter),
			trace.WithResource(res),
			trace.WithSampler(trace.TraceIDRatioBased(config.SampleRate)),
		)

		// Set global trace provider
		otel.SetTracerProvider(tp)
	}

	// Set global propagator for trace context propagation
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return &TracingMiddleware{
		tracer:     tp.Tracer("agentflow-messaging"),
		propagator: otel.GetTextMapPropagator(),
		config:     config,
	}, nil
}

// InjectTraceContext injects OpenTelemetry trace context into outgoing message
func (tm *TracingMiddleware) InjectTraceContext(ctx context.Context, msg *Message) {
	if !tm.config.Enabled {
		return
	}

	span := oteltrace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		msg.TraceID = span.SpanContext().TraceID().String()
		msg.SpanID = span.SpanContext().SpanID().String()

		// Add trace context to metadata for propagation
		if msg.Metadata == nil {
			msg.Metadata = make(map[string]interface{})
		}

		// Use a map carrier to inject trace context
		carrier := make(map[string]string)
		tm.propagator.Inject(ctx, propagation.MapCarrier(carrier))

		// Store trace context in metadata
		for key, value := range carrier {
			msg.Metadata["trace."+key] = value
		}
	}
}

// ExtractTraceContext extracts OpenTelemetry trace context from incoming message
func (tm *TracingMiddleware) ExtractTraceContext(msg *Message) context.Context {
	if !tm.config.Enabled || msg.Metadata == nil {
		return context.Background()
	}

	// Extract trace context from metadata
	carrier := make(map[string]string)
	for key, value := range msg.Metadata {
		if len(key) > 6 && key[:6] == "trace." {
			if valueStr, ok := value.(string); ok {
				carrier[key[6:]] = valueStr
			}
		}
	}

	// Extract context using propagator
	ctx := tm.propagator.Extract(context.Background(), propagation.MapCarrier(carrier))

	return ctx
}

// StartPublishSpan creates a span for message publishing operations
func (tm *TracingMiddleware) StartPublishSpan(ctx context.Context, subject string, msg *Message) (context.Context, oteltrace.Span) {
	if !tm.config.Enabled {
		return ctx, oteltrace.SpanFromContext(ctx)
	}

	spanName := fmt.Sprintf("messaging.publish %s", subject)
	ctx, span := tm.tracer.Start(ctx, spanName,
		oteltrace.WithSpanKind(oteltrace.SpanKindProducer),
		oteltrace.WithAttributes(
			attribute.String(AttrMessageSystem, "nats"),
			attribute.String(AttrMessageSubject, subject),
			attribute.String(AttrMessageID, msg.ID),
			attribute.String(AttrMessageType, string(msg.Type)),
			attribute.String(AttrMessageFrom, msg.From),
			attribute.String(AttrMessageTo, msg.To),
			attribute.String(AttrEnvelopeHash, msg.EnvelopeHash),
		),
	)

	// Add workflow and agent context if available
	if workflowID, ok := msg.Metadata["workflow_id"].(string); ok && workflowID != "" {
		span.SetAttributes(attribute.String(AttrWorkflowID, workflowID))
	}
	if agentID, ok := msg.Metadata["agent_id"].(string); ok && agentID != "" {
		span.SetAttributes(attribute.String(AttrAgentID, agentID))
	}

	// Add cost information
	if msg.Cost.Tokens > 0 || msg.Cost.Dollars > 0 {
		span.SetAttributes(
			attribute.Int("agentflow.message.cost.tokens", msg.Cost.Tokens),
			attribute.Float64("agentflow.message.cost.dollars", msg.Cost.Dollars),
		)
	}

	return ctx, span
}

// StartConsumeSpan creates a span for message consumption operations
func (tm *TracingMiddleware) StartConsumeSpan(ctx context.Context, subject string, msg *Message) (context.Context, oteltrace.Span) {
	if !tm.config.Enabled {
		return ctx, oteltrace.SpanFromContext(ctx)
	}

	spanName := fmt.Sprintf("messaging.consume %s", subject)
	ctx, span := tm.tracer.Start(ctx, spanName,
		oteltrace.WithSpanKind(oteltrace.SpanKindConsumer),
		oteltrace.WithAttributes(
			attribute.String(AttrMessageSystem, "nats"),
			attribute.String(AttrMessageSubject, subject),
			attribute.String(AttrMessageID, msg.ID),
			attribute.String(AttrMessageType, string(msg.Type)),
			attribute.String(AttrMessageFrom, msg.From),
			attribute.String(AttrMessageTo, msg.To),
			attribute.String(AttrEnvelopeHash, msg.EnvelopeHash),
		),
	)

	// Add workflow and agent context if available
	if workflowID, ok := msg.Metadata["workflow_id"].(string); ok && workflowID != "" {
		span.SetAttributes(attribute.String(AttrWorkflowID, workflowID))
	}
	if agentID, ok := msg.Metadata["agent_id"].(string); ok && agentID != "" {
		span.SetAttributes(attribute.String(AttrAgentID, agentID))
	}

	return ctx, span
}

// StartReplaySpan creates a span for message replay operations
func (tm *TracingMiddleware) StartReplaySpan(ctx context.Context, workflowID string) (context.Context, oteltrace.Span) {
	if !tm.config.Enabled {
		return ctx, oteltrace.SpanFromContext(ctx)
	}

	spanName := fmt.Sprintf("messaging.replay %s", workflowID)
	ctx, span := tm.tracer.Start(ctx, spanName,
		oteltrace.WithSpanKind(oteltrace.SpanKindClient),
		oteltrace.WithAttributes(
			attribute.String(AttrMessageSystem, "nats"),
			attribute.String(AttrWorkflowID, workflowID),
		),
	)

	return ctx, span
}

// applyTracingEnvConfig applies environment variable configuration for tracing
func applyTracingEnvConfig(config *TracingConfig) {
	if val := os.Getenv("AF_TRACING_ENABLED"); val != "" {
		config.Enabled = val == "true" || val == "1"
	}
	if val := os.Getenv("AF_OTEL_EXPORTER_OTLP_ENDPOINT"); val != "" {
		config.OTLPEndpoint = val
	}
	if val := os.Getenv("AF_SERVICE_NAME"); val != "" {
		config.ServiceName = val
	}
	// Note: Sample rate parsing would need proper float parsing in production
}

// parseOTLPEndpoint parses an OTLP endpoint URL and returns the endpoint and path components
func parseOTLPEndpoint(endpoint string) (string, string) {
	// If the endpoint doesn't contain a scheme, return as-is
	if !strings.Contains(endpoint, "://") {
		return endpoint, ""
	}
	
	// Parse the URL
	u, err := url.Parse(endpoint)
	if err != nil {
		// If parsing fails, return the original endpoint
		return endpoint, ""
	}
	
	// Extract host:port
	host := u.Host
	if host == "" {
		return endpoint, ""
	}
	
	// Extract path (remove leading slash for OTLP)
	path := strings.TrimPrefix(u.Path, "/")
	
	return host, path
}
