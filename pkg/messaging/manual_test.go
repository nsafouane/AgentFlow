package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"go.opentelemetry.io/otel/attribute"
)

// TestManualSampleMessages creates and inspects sample messages for manual verification
func TestManualSampleMessages(t *testing.T) {
	ManualTestSampleMessages()
}

// ManualTestSampleMessages creates and inspects sample messages for manual verification
func ManualTestSampleMessages() {
	fmt.Println("=== Manual Test: Sample Messages ===")

	serializer, err := NewCanonicalSerializer()
	if err != nil {
		fmt.Printf("Failed to create serializer: %v\n", err)
		return
	}

	// Test 1: Basic message
	fmt.Println("\n1. Basic Message:")
	basicMsg := &Message{
		ID:        "01HN8ZQJKM9XVQZJKM9XVQZJKM",
		From:      "agent-1",
		To:        "agent-2",
		Type:      MessageTypeRequest,
		Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		Cost:      CostInfo{Tokens: 50, Dollars: 0.005},
	}

	err = serializer.SetEnvelopeHash(basicMsg)
	if err != nil {
		fmt.Printf("Failed to set envelope hash: %v\n", err)
		return
	}

	basicData, _ := serializer.Serialize(basicMsg)
	fmt.Printf("Serialized: %s\n", string(basicData))
	fmt.Printf("Hash: %s\n", basicMsg.EnvelopeHash)

	// Test 2: Complex message with nested data
	fmt.Println("\n2. Complex Message with Nested Data:")
	complexMsg := &Message{
		ID:      "01HN8ZQJKM9XVQZJKM9XVQZJKN",
		TraceID: "abcdef1234567890abcdef1234567890",
		SpanID:  "1234567890abcdef",
		From:    "orchestrator",
		To:      "workflow.user-support.in",
		Type:    MessageTypeEvent,
		Payload: map[string]interface{}{
			"event_type": "user_query",
			"data": map[string]interface{}{
				"user_id": "user-123",
				"query":   "How do I reset my password?",
				"context": map[string]interface{}{
					"session_id": "sess-456",
					"timestamp":  "2024-01-01T12:00:00Z",
					"metadata": map[string]interface{}{
						"source":     "web",
						"user_agent": "Mozilla/5.0...",
					},
				},
			},
		},
		Metadata: map[string]interface{}{
			"workflow_id": "wf-user-support-789",
			"step":        "initial-query",
			"priority":    "normal",
			"retry_count": 0,
			"max_retries": 3,
			"timeout_ms":  30000,
		},
		Cost:      CostInfo{Tokens: 150, Dollars: 0.015},
		Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 123456789, time.UTC),
	}

	err = serializer.SetEnvelopeHash(complexMsg)
	if err != nil {
		fmt.Printf("Failed to set envelope hash: %v\n", err)
		return
	}

	complexData, _ := serializer.Serialize(complexMsg)
	fmt.Printf("Serialized: %s\n", string(complexData))
	fmt.Printf("Hash: %s\n", complexMsg.EnvelopeHash)

	// Test 3: Backward compatibility - deserialize and re-serialize
	fmt.Println("\n3. Backward Compatibility Test:")

	// Simulate an "old" message format (without some newer fields)
	oldMessageJSON := `{
		"id": "01HN8ZQJKM9XVQZJKM9XVQZJK0",
		"from": "legacy-agent",
		"to": "new-agent",
		"type": "request",
		"payload": {"action": "legacy_action"},
		"metadata": {},
		"cost": {"tokens": 25, "dollars": 0.0025},
		"ts": "2024-01-01T12:00:00Z",
		"trace_id": "",
		"span_id": "",
		"envelope_hash": ""
	}`

	fmt.Printf("Old format JSON: %s\n", oldMessageJSON)

	// Deserialize old format
	oldMsg, err := serializer.Deserialize([]byte(oldMessageJSON))
	if err != nil {
		fmt.Printf("Failed to deserialize old message: %v\n", err)
		return
	}

	// Set envelope hash for the deserialized message
	err = serializer.SetEnvelopeHash(oldMsg)
	if err != nil {
		fmt.Printf("Failed to set envelope hash for old message: %v\n", err)
		return
	}

	// Re-serialize with new format
	newData, _ := serializer.Serialize(oldMsg)
	fmt.Printf("Re-serialized: %s\n", string(newData))
	fmt.Printf("New hash: %s\n", oldMsg.EnvelopeHash)

	// Test 4: Field ordering independence
	fmt.Println("\n4. Field Ordering Independence Test:")

	// Create two identical messages with different internal map ordering
	msg1 := &Message{
		ID:        "01HN8ZQJKM9XVQZJKM9XVQZJKP",
		From:      "agent-a",
		To:        "agent-b",
		Type:      MessageTypeResponse,
		Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		Payload: map[string]interface{}{
			"result": "success",
			"data":   "test-data",
			"code":   200,
		},
		Metadata: map[string]interface{}{
			"workflow": "test-wf",
			"step":     "final",
			"version":  "1.0",
		},
		Cost: CostInfo{Tokens: 75, Dollars: 0.0075},
	}

	msg2 := &Message{
		ID:        "01HN8ZQJKM9XVQZJKM9XVQZJKP",
		From:      "agent-a",
		To:        "agent-b",
		Type:      MessageTypeResponse,
		Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		Payload: map[string]interface{}{
			"code":   200,
			"result": "success",
			"data":   "test-data",
		},
		Metadata: map[string]interface{}{
			"version":  "1.0",
			"workflow": "test-wf",
			"step":     "final",
		},
		Cost: CostInfo{Tokens: 75, Dollars: 0.0075},
	}

	hash1, _ := serializer.ComputeHash(msg1)
	hash2, _ := serializer.ComputeHash(msg2)

	fmt.Printf("Message 1 hash: %s\n", hash1)
	fmt.Printf("Message 2 hash: %s\n", hash2)
	fmt.Printf("Hashes match: %t\n", hash1 == hash2)

	data1, _ := serializer.Serialize(msg1)
	data2, _ := serializer.Serialize(msg2)
	fmt.Printf("Serializations match: %t\n", string(data1) == string(data2))

	// Test 5: Schema evolution scenario
	fmt.Println("\n5. Schema Evolution Scenario:")

	// Simulate adding a new optional field to the message
	currentMsg := &Message{
		ID:        "01HN8ZQJKM9XVQZJKM9XVQZJKQ",
		From:      "new-agent",
		To:        "another-agent",
		Type:      MessageTypeEvent,
		Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		Cost:      CostInfo{Tokens: 100, Dollars: 0.01},
	}

	// Add some metadata that might represent new fields
	currentMsg.AddMetadata("schema_version", "1.1")
	currentMsg.AddMetadata("new_feature_flag", true)
	currentMsg.AddMetadata("compatibility_mode", "backward")

	err = serializer.SetEnvelopeHash(currentMsg)
	if err != nil {
		fmt.Printf("Failed to set envelope hash: %v\n", err)
		return
	}

	currentData, _ := serializer.Serialize(currentMsg)
	fmt.Printf("Current message: %s\n", string(currentData))

	// Verify it can be deserialized and validated
	deserializedCurrent, err := serializer.Deserialize(currentData)
	if err != nil {
		fmt.Printf("Failed to deserialize current message: %v\n", err)
		return
	}

	err = serializer.ValidateHash(deserializedCurrent)
	if err != nil {
		fmt.Printf("Hash validation failed: %v\n", err)
		return
	}

	fmt.Printf("Schema evolution test passed: message with new metadata fields works correctly\n")

	fmt.Println("\n=== Manual Test Complete ===")
}

// PrettyPrintMessage formats a message for human-readable output
func PrettyPrintMessage(msg *Message) string {
	data, err := json.MarshalIndent(msg, "", "  ")
	if err != nil {
		return fmt.Sprintf("Error formatting message: %v", err)
	}
	return string(data)
}

// TestManualNATSLatency measures NATS server roundtrip latency
// Run with: go test -v -run TestManualNATSLatency
// Note: Requires a local NATS server running on localhost:4222 with JetStream enabled
// Start with: nats-server -js
func TestManualNATSLatency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping manual latency test in short mode")
	}

	ManualTestNATSLatency()
}

// ManualTestNATSLatency measures NATS server roundtrip latency
func ManualTestNATSLatency() {
	fmt.Println("=== Manual Test: NATS Latency ===")

	// Try to connect to local NATS server
	config := &BusConfig{
		URL:            "nats://localhost:4222",
		MaxReconnect:   3,
		ReconnectWait:  1 * time.Second,
		AckWait:        10 * time.Second,
		MaxInFlight:    100,
		ConnectTimeout: 5 * time.Second,
		RequestTimeout: 5 * time.Second,
	}

	fmt.Printf("Connecting to NATS server at %s...\n", config.URL)

	bus, err := NewNATSBus(config)
	if err != nil {
		fmt.Printf("Could not connect to local NATS server: %v\n", err)
		fmt.Println("Please start a local NATS server with: nats-server -js")
		return
	}
	defer bus.Close()

	fmt.Println("Connected successfully!")

	// Test parameters
	const (
		numMessages = 100
		subject     = "test.latency"
	)

	// Set up subscription first
	receivedMessages := make(chan *Message, numMessages)
	latencies := make([]time.Duration, 0, numMessages)

	handler := func(ctx context.Context, msg *Message) error {
		// Extract send time from metadata
		if sendTimeStr, ok := msg.Metadata["send_time"].(string); ok {
			if sendTime, err := time.Parse(time.RFC3339Nano, sendTimeStr); err == nil {
				latency := time.Since(sendTime)
				latencies = append(latencies, latency)
			}
		}
		receivedMessages <- msg
		return nil
	}

	ctx := context.Background()
	sub, err := bus.Subscribe(ctx, subject, handler)
	if err != nil {
		fmt.Printf("Failed to subscribe: %v\n", err)
		return
	}
	defer sub.Unsubscribe()

	// Wait a moment for subscription to be ready
	time.Sleep(100 * time.Millisecond)

	fmt.Printf("Publishing %d messages to measure latency...\n", numMessages)

	// Publish messages and measure latency
	start := time.Now()
	for i := 0; i < numMessages; i++ {
		msg := NewMessage(fmt.Sprintf("latency-test-%d", i), "latency-tester", "latency-receiver", MessageTypeEvent)
		msg.SetPayload(map[string]interface{}{
			"sequence": i,
			"data":     "latency test payload",
		})

		// Add send time to metadata for latency calculation
		sendTime := time.Now()
		msg.AddMetadata("send_time", sendTime.Format(time.RFC3339Nano))

		err := bus.Publish(ctx, subject, msg)
		if err != nil {
			fmt.Printf("Failed to publish message %d: %v\n", i, err)
			return
		}
	}

	// Wait for all messages to be received
	timeout := time.After(10 * time.Second)
	receivedCount := 0

	for receivedCount < numMessages {
		select {
		case <-receivedMessages:
			receivedCount++
		case <-timeout:
			fmt.Printf("Only received %d out of %d messages\n", receivedCount, numMessages)
			return
		}
	}

	totalDuration := time.Since(start)

	// Calculate statistics
	if len(latencies) == 0 {
		fmt.Println("No latency measurements collected")
		return
	}

	// Sort latencies for percentile calculations
	sortedLatencies := make([]time.Duration, len(latencies))
	copy(sortedLatencies, latencies)

	// Simple bubble sort for small dataset
	for i := 0; i < len(sortedLatencies); i++ {
		for j := i + 1; j < len(sortedLatencies); j++ {
			if sortedLatencies[i] > sortedLatencies[j] {
				sortedLatencies[i], sortedLatencies[j] = sortedLatencies[j], sortedLatencies[i]
			}
		}
	}

	// Calculate percentiles
	p50Index := len(sortedLatencies) * 50 / 100
	p95Index := len(sortedLatencies) * 95 / 100
	p99Index := len(sortedLatencies) * 99 / 100

	if p50Index >= len(sortedLatencies) {
		p50Index = len(sortedLatencies) - 1
	}
	if p95Index >= len(sortedLatencies) {
		p95Index = len(sortedLatencies) - 1
	}
	if p99Index >= len(sortedLatencies) {
		p99Index = len(sortedLatencies) - 1
	}

	p50 := sortedLatencies[p50Index]
	p95 := sortedLatencies[p95Index]
	p99 := sortedLatencies[p99Index]
	min := sortedLatencies[0]
	max := sortedLatencies[len(sortedLatencies)-1]

	// Calculate average
	var total time.Duration
	for _, latency := range latencies {
		total += latency
	}
	avg := total / time.Duration(len(latencies))

	// Print results
	fmt.Println("\n=== Latency Results ===")
	fmt.Printf("Messages: %d\n", numMessages)
	fmt.Printf("Total time: %v\n", totalDuration)
	fmt.Printf("Throughput: %.2f msg/sec\n", float64(numMessages)/totalDuration.Seconds())
	fmt.Println()
	fmt.Printf("Latency Statistics:\n")
	fmt.Printf("  Min:     %v\n", min)
	fmt.Printf("  Average: %v\n", avg)
	fmt.Printf("  P50:     %v\n", p50)
	fmt.Printf("  P95:     %v\n", p95)
	fmt.Printf("  P99:     %v\n", p99)
	fmt.Printf("  Max:     %v\n", max)

	// Check if we meet the performance target (p95 < 15ms)
	target := 15 * time.Millisecond
	fmt.Printf("\nPerformance Target: P95 < %v\n", target)
	if p95 < target {
		fmt.Printf("✅ PASSED: P95 latency %v is below target %v\n", p95, target)
	} else {
		fmt.Printf("❌ FAILED: P95 latency %v exceeds target %v\n", p95, target)
	}

	// Additional analysis
	fmt.Println("\n=== Analysis ===")
	slowMessages := 0
	for _, latency := range latencies {
		if latency > target {
			slowMessages++
		}
	}
	fmt.Printf("Messages exceeding %v: %d (%.1f%%)\n", target, slowMessages, float64(slowMessages)*100/float64(len(latencies)))

	fmt.Println("\n=== Manual NATS Latency Test Complete ===")
}

// TestManualTracingJaeger creates traces and provides instructions for Jaeger UI verification
// Run with: go test -v -run TestManualTracingJaeger
// Note: Requires Jaeger running locally. Start with:
// docker run -d --name jaeger -p 16686:16686 -p 14268:14268 jaegertracing/all-in-one:latest
func TestManualTracingJaeger(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping manual tracing test in short mode")
	}

	ManualTestTracingJaeger()
}

// ManualTestTracingJaeger creates traces and provides instructions for Jaeger UI verification
func ManualTestTracingJaeger() {
	fmt.Println("=== Manual Test: Tracing with Jaeger ===")

	// Create tracing middleware with real OTLP exporter
	config := &TracingConfig{
		Enabled:      true,
		OTLPEndpoint: "http://localhost:4318",
		ServiceName:  "agentflow-messaging-manual-test",
		SampleRate:   1.0,
	}

	fmt.Printf("Creating tracing middleware with endpoint: %s\n", config.OTLPEndpoint)

	tracing, err := NewTracingMiddleware(config)
	if err != nil {
		fmt.Printf("Failed to create tracing middleware: %v\n", err)
		fmt.Println("Please ensure Jaeger is running with OTLP support:")
		fmt.Println("docker run -d --name jaeger \\")
		fmt.Println("  -p 16686:16686 \\")
		fmt.Println("  -p 14268:14268 \\")
		fmt.Println("  -p 4317:4317 \\")
		fmt.Println("  -p 4318:4318 \\")
		fmt.Println("  jaegertracing/all-in-one:latest")
		return
	}

	// Try to connect to NATS server
	busConfig := &BusConfig{
		URL:            "nats://localhost:4222",
		MaxReconnect:   3,
		ReconnectWait:  1 * time.Second,
		AckWait:        10 * time.Second,
		MaxInFlight:    100,
		ConnectTimeout: 5 * time.Second,
		RequestTimeout: 5 * time.Second,
	}

	fmt.Printf("Connecting to NATS server at %s...\n", busConfig.URL)

	bus, err := NewNATSBus(busConfig)
	if err != nil {
		fmt.Printf("Could not connect to local NATS server: %v\n", err)
		fmt.Println("Please start a local NATS server with: nats-server -js")
		return
	}
	defer bus.Close()

	fmt.Println("Connected to NATS successfully!")

	// Create a complex workflow scenario with multiple message hops
	ctx := context.Background()

	// Start root span for the workflow
	rootCtx, rootSpan := tracing.tracer.Start(ctx, "customer-support-workflow")
	rootTraceID := rootSpan.SpanContext().TraceID().String()

	fmt.Printf("Started workflow trace: %s\n", rootTraceID)

	// Step 1: Customer query arrives
	fmt.Println("\n1. Customer query processing...")
	customerMsg := NewMessage("msg-001", "customer-portal", "query-classifier", MessageTypeRequest)
	customerMsg.SetPayload(map[string]interface{}{
		"query":       "How do I reset my password?",
		"customer_id": "cust-12345",
		"urgency":     "normal",
	})
	customerMsg.AddMetadata("workflow_id", "support-wf-789")
	customerMsg.AddMetadata("step", "initial-query")
	customerMsg.SetCost(25, 0.0025)

	publishCtx1, publishSpan1 := tracing.StartPublishSpan(rootCtx, "agents.query-classifier.in", customerMsg)
	tracing.InjectTraceContext(publishCtx1, customerMsg)
	err = bus.Publish(publishCtx1, "agents.query-classifier.in", customerMsg)
	if err != nil {
		fmt.Printf("Failed to publish customer message: %v\n", err)
		return
	}
	publishSpan1.End()

	// Step 2: Query classification
	fmt.Println("2. Query classification...")
	time.Sleep(50 * time.Millisecond) // Simulate processing time

	classificationMsg := NewMessage("msg-002", "query-classifier", "knowledge-base", MessageTypeRequest)
	classificationMsg.SetPayload(map[string]interface{}{
		"classification": "password-reset",
		"confidence":     0.95,
		"intent":         "account-access",
	})
	classificationMsg.AddMetadata("workflow_id", "support-wf-789")
	classificationMsg.AddMetadata("step", "classification")
	classificationMsg.AddMetadata("parent_message_id", customerMsg.ID)
	classificationMsg.SetCost(50, 0.005)

	// Extract context from customer message and continue trace
	extractedCtx1 := tracing.ExtractTraceContext(customerMsg)
	consumeCtx1, consumeSpan1 := tracing.StartConsumeSpan(extractedCtx1, "agents.query-classifier.in", customerMsg)

	publishCtx2, publishSpan2 := tracing.StartPublishSpan(consumeCtx1, "agents.knowledge-base.in", classificationMsg)
	tracing.InjectTraceContext(publishCtx2, classificationMsg)
	err = bus.Publish(publishCtx2, "agents.knowledge-base.in", classificationMsg)
	if err != nil {
		fmt.Printf("Failed to publish classification message: %v\n", err)
		return
	}
	publishSpan2.End()
	consumeSpan1.End()

	// Step 3: Knowledge base lookup
	fmt.Println("3. Knowledge base lookup...")
	time.Sleep(100 * time.Millisecond) // Simulate processing time

	kbMsg := NewMessage("msg-003", "knowledge-base", "response-generator", MessageTypeResponse)
	kbMsg.SetPayload(map[string]interface{}{
		"articles": []map[string]interface{}{
			{
				"id":    "kb-001",
				"title": "How to Reset Your Password",
				"url":   "https://help.example.com/password-reset",
				"score": 0.92,
			},
			{
				"id":    "kb-002",
				"title": "Account Security Best Practices",
				"url":   "https://help.example.com/security",
				"score": 0.78,
			},
		},
		"total_results": 2,
	})
	kbMsg.AddMetadata("workflow_id", "support-wf-789")
	kbMsg.AddMetadata("step", "knowledge-lookup")
	kbMsg.AddMetadata("parent_message_id", classificationMsg.ID)
	kbMsg.SetCost(75, 0.0075)

	// Continue trace from classification message
	extractedCtx2 := tracing.ExtractTraceContext(classificationMsg)
	consumeCtx2, consumeSpan2 := tracing.StartConsumeSpan(extractedCtx2, "agents.knowledge-base.in", classificationMsg)

	publishCtx3, publishSpan3 := tracing.StartPublishSpan(consumeCtx2, "agents.response-generator.in", kbMsg)
	tracing.InjectTraceContext(publishCtx3, kbMsg)
	err = bus.Publish(publishCtx3, "agents.response-generator.in", kbMsg)
	if err != nil {
		fmt.Printf("Failed to publish KB message: %v\n", err)
		return
	}
	publishSpan3.End()
	consumeSpan2.End()

	// Step 4: Response generation
	fmt.Println("4. Response generation...")
	time.Sleep(150 * time.Millisecond) // Simulate processing time

	responseMsg := NewMessage("msg-004", "response-generator", "customer-portal", MessageTypeResponse)
	responseMsg.SetPayload(map[string]interface{}{
		"response":   "To reset your password, please visit https://help.example.com/password-reset and follow the instructions. You'll need access to your registered email address.",
		"confidence": 0.88,
		"sources":    []string{"kb-001", "kb-002"},
		"followup_suggestions": []string{
			"Need help with two-factor authentication?",
			"Want to update your security settings?",
		},
	})
	responseMsg.AddMetadata("workflow_id", "support-wf-789")
	responseMsg.AddMetadata("step", "response-generation")
	responseMsg.AddMetadata("parent_message_id", kbMsg.ID)
	responseMsg.AddMetadata("original_query_id", customerMsg.ID)
	responseMsg.SetCost(100, 0.01)

	// Continue trace from KB message
	extractedCtx3 := tracing.ExtractTraceContext(kbMsg)
	consumeCtx3, consumeSpan3 := tracing.StartConsumeSpan(extractedCtx3, "agents.response-generator.in", kbMsg)

	publishCtx4, publishSpan4 := tracing.StartPublishSpan(consumeCtx3, "agents.customer-portal.out", responseMsg)
	tracing.InjectTraceContext(publishCtx4, responseMsg)
	err = bus.Publish(publishCtx4, "agents.customer-portal.out", responseMsg)
	if err != nil {
		fmt.Printf("Failed to publish response message: %v\n", err)
		return
	}
	publishSpan4.End()
	consumeSpan3.End()

	// Step 5: Final delivery confirmation
	fmt.Println("5. Final delivery...")
	time.Sleep(25 * time.Millisecond) // Simulate processing time

	// Continue trace from response message
	extractedCtx4 := tracing.ExtractTraceContext(responseMsg)
	_, finalSpan := tracing.StartConsumeSpan(extractedCtx4, "agents.customer-portal.out", responseMsg)

	// Add some final span attributes
	finalSpan.SetAttributes(
		attribute.String("workflow.status", "completed"),
		attribute.String("workflow.outcome", "success"),
		attribute.Int("workflow.total_messages", 4),
		attribute.Float64("workflow.total_cost", 0.0225),
	)

	finalSpan.End()

	// End root span
	rootSpan.SetAttributes(
		attribute.String("workflow.type", "customer-support"),
		attribute.String("workflow.id", "support-wf-789"),
		attribute.String("customer.id", "cust-12345"),
		attribute.String("query.classification", "password-reset"),
		attribute.Float64("workflow.confidence", 0.88),
	)
	rootSpan.End()

	fmt.Printf("\nWorkflow completed! Trace ID: %s\n", rootTraceID)

	// Test message replay with tracing
	fmt.Println("\n6. Testing message replay with tracing...")
	replayCtx, replaySpan := tracing.StartReplaySpan(ctx, "support-wf-789")

	// Simulate replay from 1 minute ago
	replayFrom := time.Now().Add(-1 * time.Minute)
	messages, err := bus.Replay(replayCtx, "support-wf-789", replayFrom)
	if err != nil {
		fmt.Printf("Replay failed: %v\n", err)
	} else {
		fmt.Printf("Replayed %d messages\n", len(messages))
		replaySpan.SetAttributes(
			attribute.Int("replay.message_count", len(messages)),
			attribute.String("replay.status", "success"),
		)
	}
	replaySpan.End()

	// Give traces time to be exported
	fmt.Println("\nWaiting for traces to be exported...")
	time.Sleep(2 * time.Second)

	// Print verification instructions
	fmt.Println("\n=== Verification Instructions ===")
	fmt.Printf("1. Open Jaeger UI: http://localhost:16686\n")
	fmt.Printf("2. Select service: %s\n", config.ServiceName)
	fmt.Printf("3. Search for trace ID: %s\n", rootTraceID)
	fmt.Println("4. Verify the following spans are present:")
	fmt.Println("   - customer-support-workflow (root span)")
	fmt.Println("   - messaging.publish agents.query-classifier.in")
	fmt.Println("   - messaging.consume agents.query-classifier.in")
	fmt.Println("   - messaging.publish agents.knowledge-base.in")
	fmt.Println("   - messaging.consume agents.knowledge-base.in")
	fmt.Println("   - messaging.publish agents.response-generator.in")
	fmt.Println("   - messaging.consume agents.response-generator.in")
	fmt.Println("   - messaging.publish agents.customer-portal.out")
	fmt.Println("   - messaging.consume agents.customer-portal.out")
	fmt.Println("   - messaging.replay support-wf-789")
	fmt.Println("5. Verify all spans have the same trace ID")
	fmt.Println("6. Check span attributes for message details:")
	fmt.Println("   - messaging.message.id")
	fmt.Println("   - messaging.message.type")
	fmt.Println("   - messaging.destination.name")
	fmt.Println("   - agentflow.workflow.id")
	fmt.Println("   - agentflow.message.cost.tokens")
	fmt.Println("   - agentflow.message.cost.dollars")
	fmt.Println("7. Verify span timing shows realistic message processing latencies")

	fmt.Println("\n=== Expected Trace Structure ===")
	fmt.Println("customer-support-workflow")
	fmt.Println("├── messaging.publish agents.query-classifier.in")
	fmt.Println("├── messaging.consume agents.query-classifier.in")
	fmt.Println("│   └── messaging.publish agents.knowledge-base.in")
	fmt.Println("├── messaging.consume agents.knowledge-base.in")
	fmt.Println("│   └── messaging.publish agents.response-generator.in")
	fmt.Println("├── messaging.consume agents.response-generator.in")
	fmt.Println("│   └── messaging.publish agents.customer-portal.out")
	fmt.Println("├── messaging.consume agents.customer-portal.out")
	fmt.Println("└── messaging.replay support-wf-789")

	fmt.Printf("\nTrace attributes to verify:\n")
	fmt.Printf("- All spans should have trace_id: %s\n", rootTraceID)
	fmt.Printf("- Message IDs: msg-001, msg-002, msg-003, msg-004\n")
	fmt.Printf("- Workflow ID: support-wf-789\n")
	fmt.Printf("- Total cost: $0.0225 (225 tokens)\n")

	fmt.Println("\n=== Manual Tracing Test Complete ===")
	fmt.Println("Please verify the traces in Jaeger UI as described above.")
}
