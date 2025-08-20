//go:build manual

package messaging

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// TestLocalPerformanceHarness tests the performance harness without Docker containers
// This test demonstrates the harness functionality using a mock message bus
// Run with: go test -tags=manual -v ./pkg/messaging -run TestLocalPerformanceHarness
func TestLocalPerformanceHarness(t *testing.T) {
	fmt.Println("=== Local Performance Harness Test ===")
	fmt.Println("This test demonstrates the performance harness functionality")
	fmt.Println("using a mock message bus (no Docker required)")
	fmt.Println()

	// Create a mock message bus for testing
	mockBus := &MockMessageBus{
		latency: 2 * time.Millisecond, // Simulate 2ms latency
	}

	// Configure a small performance test
	config := &PerformanceConfig{
		MessageCount:   50,                    // Small number for quick test
		Concurrency:    3,                     // Low concurrency
		PayloadSize:    512,                   // Small payload
		WarmupMessages: 5,                     // Minimal warmup
		TestDuration:   30 * time.Second,      // Short duration
		P95Threshold:   10 * time.Millisecond, // Reasonable threshold
		P50Threshold:   5 * time.Millisecond,  // Reasonable threshold
		EnableTracing:  false,                 // No tracing overhead
		Subject:        "test.local.performance",
		ReportInterval: 2 * time.Second, // Frequent reporting
	}

	fmt.Printf("Test Configuration:\n")
	fmt.Printf("  Messages: %d\n", config.MessageCount)
	fmt.Printf("  Concurrency: %d\n", config.Concurrency)
	fmt.Printf("  Payload Size: %d bytes\n", config.PayloadSize)
	fmt.Printf("  P95 Threshold: %v\n", config.P95Threshold)
	fmt.Printf("  Mock Latency: %v\n", mockBus.latency)
	fmt.Println()

	// Create and run the performance harness
	harness := NewPerformanceHarness(mockBus, config)
	defer harness.Close()

	fmt.Println("Running performance test...")
	result, err := harness.RunPingPongTest()
	if err != nil {
		t.Fatalf("Performance test failed: %v", err)
	}

	// Display results
	fmt.Printf("\n=== Test Results ===\n")
	fmt.Printf("Messages: %d sent, %d received (%.1f%% completion)\n",
		result.MessagesSent, result.MessagesReceived,
		float64(result.MessagesReceived)/float64(result.MessagesSent)*100)
	fmt.Printf("Duration: %v\n", result.TestDuration)
	fmt.Printf("Throughput: %.2f msg/sec\n", result.Throughput)
	fmt.Printf("Latency:\n")
	fmt.Printf("  Min: %v\n", result.MinLatency)
	fmt.Printf("  Avg: %v\n", result.AvgLatency)
	fmt.Printf("  P50: %v (threshold: %v) %s\n", result.P50Latency, config.P50Threshold, passFailIcon(result.P50Passed))
	fmt.Printf("  P95: %v (threshold: %v) %s\n", result.P95Latency, config.P95Threshold, passFailIcon(result.P95Passed))
	fmt.Printf("  P99: %v\n", result.P99Latency)
	fmt.Printf("  Max: %v\n", result.MaxLatency)
	fmt.Printf("Errors: %d publish, %d consume, %d timeout\n",
		result.PublishErrors, result.ConsumeErrors, result.TimeoutErrors)

	// Print latency distribution
	fmt.Printf("Latency Distribution:\n")
	for bucket, count := range result.LatencyDistribution {
		percentage := float64(count) / float64(result.MessagesReceived) * 100
		fmt.Printf("  %8s: %4d (%5.1f%%)\n", bucket, count, percentage)
	}

	fmt.Printf("Overall: %s\n", passFailIcon(result.OverallPassed))

	// Validate results
	if result.MessagesSent != int64(config.MessageCount) {
		t.Errorf("Expected %d messages sent, got %d", config.MessageCount, result.MessagesSent)
	}

	if result.MessagesReceived == 0 {
		t.Errorf("No messages were received")
	}

	if result.TestDuration == 0 {
		t.Errorf("Test duration should be greater than 0")
	}

	if result.Throughput == 0 {
		t.Errorf("Throughput should be greater than 0")
	}

	// Check that latencies are reasonable for mock bus
	if result.MinLatency < time.Millisecond || result.MinLatency > 5*time.Millisecond {
		t.Errorf("Min latency %v seems unreasonable for mock bus", result.MinLatency)
	}

	if result.MaxLatency > 20*time.Millisecond {
		t.Errorf("Max latency %v seems too high for mock bus", result.MaxLatency)
	}

	fmt.Println("\n=== Local Performance Harness Test Complete ===")
}

// MockMessageBus implements MessageBus interface for testing
type MockMessageBus struct {
	latency time.Duration
}

func (m *MockMessageBus) Publish(ctx context.Context, subject string, msg *Message) error {
	// Simulate network latency
	time.Sleep(m.latency / 2) // Half the latency for publish
	return nil
}

func (m *MockMessageBus) Subscribe(ctx context.Context, subject string, handler MessageHandler) (*Subscription, error) {
	// Create a mock subscription that simulates message processing
	sub := &Subscription{
		Subject:  subject,
		Consumer: "mock-consumer",
		IsActive: true,
	}

	// Start a goroutine to simulate message processing
	go func() {
		// This is a simplified mock - in a real test we'd need to coordinate
		// with the publish calls to simulate actual ping-pong behavior
		// For now, we'll just simulate some message processing
		for sub.IsActive {
			select {
			case <-ctx.Done():
				return
			case <-time.After(100 * time.Millisecond):
				// Simulate receiving a message and processing it
				mockMsg := &Message{
					ID:   "mock-msg",
					From: "mock-sender",
					To:   "mock-receiver",
					Type: MessageTypeRequest,
					Metadata: map[string]interface{}{
						"send_time": time.Now().Add(-m.latency).Format(time.RFC3339Nano),
					},
				}

				// Simulate processing latency
				time.Sleep(m.latency / 2)

				// Call the handler
				handler(ctx, mockMsg)
			}
		}
	}()

	return sub, nil
}

func (m *MockMessageBus) Replay(ctx context.Context, workflowID string, from time.Time) ([]Message, error) {
	// Return empty slice for mock
	return []Message{}, nil
}

func (m *MockMessageBus) Close() error {
	return nil
}
