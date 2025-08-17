// Manual test for structured logging during ping-pong scenario
// This file demonstrates the logging functionality with correlation IDs
// Run with: go run internal/logging/manual_demo.go

package main

import (
	"context"
	"fmt"
	"time"

	"github.com/agentflow/agentflow/internal/logging"
	"github.com/agentflow/agentflow/pkg/messaging"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/trace"
)

func main() {
	fmt.Println("=== AgentFlow Structured Logging Manual Test ===")
	fmt.Println("This test demonstrates structured logging with correlation IDs during a ping-pong scenario")
	fmt.Println("All log entries should include trace_id, span_id, message_id, and other correlation fields")
	fmt.Println()

	// Create a tracer for the test
	tracer := otel.Tracer("manual-test")

	// Create base logger
	logger := logging.NewLogger()

	// Simulate a ping-pong message exchange scenario
	simulatePingPongScenario(tracer, logger)

	fmt.Println()
	fmt.Println("=== Manual Test Complete ===")
	fmt.Println("Review the JSON log entries above to verify:")
	fmt.Println("1. All entries are in JSON format")
	fmt.Println("2. Correlation fields (trace_id, span_id, message_id) are present")
	fmt.Println("3. Field ordering is consistent")
	fmt.Println("4. Workflow and agent context is preserved")
}

func simulatePingPongScenario(tracer trace.Tracer, baseLogger logging.Logger) {
	// Start a root span for the ping-pong workflow
	ctx, rootSpan := tracer.Start(context.Background(), "ping-pong-workflow")
	defer rootSpan.End()

	// Create workflow-scoped logger
	workflowLogger := baseLogger.
		WithTrace(ctx).
		WithWorkflow("ping-pong-001").
		WithAgent("orchestrator")

	workflowLogger.Info("Starting ping-pong workflow",
		logging.String("scenario", "manual_test"),
		logging.Int("expected_messages", 4),
	)

	// Simulate Agent A sending ping
	simulateAgentMessage(ctx, tracer, baseLogger, "agent-a", "agent-b", "ping", 1)

	// Simulate Agent B responding with pong
	simulateAgentMessage(ctx, tracer, baseLogger, "agent-b", "agent-a", "pong", 2)

	// Simulate Agent A sending another ping
	simulateAgentMessage(ctx, tracer, baseLogger, "agent-a", "agent-b", "ping", 3)

	// Simulate Agent B responding with final pong
	simulateAgentMessage(ctx, tracer, baseLogger, "agent-b", "agent-a", "pong", 4)

	// Log workflow completion
	workflowLogger.Info("Ping-pong workflow completed successfully",
		logging.String("status", "completed"),
		logging.Float64("duration_seconds", 0.1),
	)
}

func simulateAgentMessage(parentCtx context.Context, tracer trace.Tracer, baseLogger logging.Logger, fromAgent, toAgent, messageType string, sequence int) {
	// Create a child span for this message
	ctx, span := tracer.Start(parentCtx, fmt.Sprintf("%s-message-%d", messageType, sequence))
	defer span.End()

	// Create a message
	msg := messaging.NewMessage(
		fmt.Sprintf("msg-%d", sequence),
		fromAgent,
		toAgent,
		messaging.MessageTypeRequest,
	)

	// Set trace context on message
	spanContext := span.SpanContext()
	if spanContext.IsValid() {
		msg.SetTraceContext(
			spanContext.TraceID().String(),
			spanContext.SpanID().String(),
		)
	}

	// Set message payload
	msg.SetPayload(map[string]interface{}{
		"type":     messageType,
		"sequence": sequence,
		"data":     fmt.Sprintf("Hello from %s!", fromAgent),
	})

	// Create agent-specific logger with message correlation
	agentLogger := baseLogger.
		WithTrace(ctx).
		WithMessage(msg.ID).
		WithWorkflow("ping-pong-001").
		WithAgent(fromAgent)

	// Log message sending
	agentLogger.Info("Sending message",
		logging.String("to_agent", toAgent),
		logging.String("message_type", messageType),
		logging.Int("sequence", sequence),
		logging.String("payload_type", "ping_pong_data"),
	)

	// Simulate some processing time
	time.Sleep(10 * time.Millisecond)

	// Log message processing on receiving side
	receiverLogger := baseLogger.
		WithTrace(ctx).
		WithMessage(msg.ID).
		WithWorkflow("ping-pong-001").
		WithAgent(toAgent)

	receiverLogger.Info("Received and processing message",
		logging.String("from_agent", fromAgent),
		logging.String("message_type", messageType),
		logging.Int("sequence", sequence),
		logging.Bool("processing_success", true),
	)

	// Simulate potential error scenario (every 3rd message)
	if sequence%3 == 0 {
		receiverLogger.Warn("Simulated processing warning",
			logging.String("warning_type", "rate_limit_approaching"),
			logging.Int("remaining_quota", 100-sequence*10),
		)
	}

	// Log successful processing
	receiverLogger.Debug("Message processing completed",
		logging.String("processing_result", "success"),
		logging.Float64("processing_time_ms", 8.5),
		logging.Int("tokens_used", 15+sequence*2),
	)
}
