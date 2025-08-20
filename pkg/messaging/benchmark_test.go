package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"testing"
	"time"
)

// BenchmarkPingPongLatency benchmarks message routing latency using ping-pong pattern
func BenchmarkPingPongLatency(b *testing.B) {
	// Skip if running in short mode
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}

	ctx := context.Background()

	// Start NATS container for testing
	natsContainer, err := StartNATSContainer(ctx)
	if err != nil {
		b.Fatalf("Failed to start NATS container: %v", err)
	}
	defer natsContainer.Stop(ctx)

	// Create bus
	config := &BusConfig{
		URL:            natsContainer.URL,
		MaxReconnect:   3,
		ReconnectWait:  1 * time.Second,
		AckWait:        10 * time.Second,
		MaxInFlight:    1000,
		ConnectTimeout: 5 * time.Second,
		RequestTimeout: 5 * time.Second,
	}

	bus, err := NewNATSBus(config)
	if err != nil {
		b.Fatalf("Failed to create NATS bus: %v", err)
	}
	defer bus.Close()

	// Configure performance test
	perfConfig := DefaultPerformanceConfig()
	perfConfig.MessageCount = b.N
	perfConfig.Concurrency = 1 // Single threaded for benchmark
	perfConfig.PayloadSize = 1024
	perfConfig.WarmupMessages = 10
	perfConfig.EnableTracing = false // Disable tracing for performance
	perfConfig.TestDuration = 60 * time.Second

	// Run performance test
	harness := NewPerformanceHarness(bus, perfConfig)
	defer harness.Close()

	b.ResetTimer()
	result, err := harness.RunPingPongTest()
	b.StopTimer()

	if err != nil {
		b.Fatalf("Performance test failed: %v", err)
	}

	// Report metrics
	b.ReportMetric(float64(result.P50Latency.Nanoseconds())/1e6, "p50_latency_ms")
	b.ReportMetric(float64(result.P95Latency.Nanoseconds())/1e6, "p95_latency_ms")
	b.ReportMetric(result.Throughput, "throughput_msg_per_sec")
	b.ReportMetric(float64(result.MessagesReceived)/float64(result.MessagesSent)*100, "success_rate_percent")

	// Log detailed results
	b.Logf("Performance Results:")
	b.Logf("  Messages: %d sent, %d received", result.MessagesSent, result.MessagesReceived)
	b.Logf("  Duration: %v", result.TestDuration)
	b.Logf("  Throughput: %.2f msg/sec", result.Throughput)
	b.Logf("  Latency P50: %v", result.P50Latency)
	b.Logf("  Latency P95: %v", result.P95Latency)
	b.Logf("  Latency P99: %v", result.P99Latency)
	b.Logf("  P95 Threshold: %v (Passed: %t)", perfConfig.P95Threshold, result.P95Passed)
}

// BenchmarkPingPongConcurrency benchmarks with different concurrency levels
func BenchmarkPingPongConcurrency(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}

	concurrencyLevels := []int{1, 5, 10, 20, 50}

	for _, concurrency := range concurrencyLevels {
		b.Run(fmt.Sprintf("Concurrency%d", concurrency), func(b *testing.B) {
			ctx := context.Background()

			natsContainer, err := StartNATSContainer(ctx)
			if err != nil {
				b.Fatalf("Failed to start NATS container: %v", err)
			}
			defer natsContainer.Stop(ctx)

			config := &BusConfig{
				URL:            natsContainer.URL,
				MaxReconnect:   3,
				ReconnectWait:  1 * time.Second,
				AckWait:        10 * time.Second,
				MaxInFlight:    1000,
				ConnectTimeout: 5 * time.Second,
				RequestTimeout: 5 * time.Second,
			}

			bus, err := NewNATSBus(config)
			if err != nil {
				b.Fatalf("Failed to create NATS bus: %v", err)
			}
			defer bus.Close()

			perfConfig := DefaultPerformanceConfig()
			perfConfig.MessageCount = 1000 // Fixed message count for concurrency test
			perfConfig.Concurrency = concurrency
			perfConfig.PayloadSize = 1024
			perfConfig.WarmupMessages = 50
			perfConfig.EnableTracing = false
			perfConfig.TestDuration = 60 * time.Second

			harness := NewPerformanceHarness(bus, perfConfig)
			defer harness.Close()

			b.ResetTimer()
			result, err := harness.RunPingPongTest()
			b.StopTimer()

			if err != nil {
				b.Fatalf("Performance test failed: %v", err)
			}

			b.ReportMetric(float64(result.P95Latency.Nanoseconds())/1e6, "p95_latency_ms")
			b.ReportMetric(result.Throughput, "throughput_msg_per_sec")
		})
	}
}

// BenchmarkPingPongPayloadSize benchmarks with different payload sizes
func BenchmarkPingPongPayloadSize(b *testing.B) {
	if testing.Short() {
		b.Skip("Skipping benchmark in short mode")
	}

	payloadSizes := []int{100, 1024, 4096, 16384, 65536} // 100B to 64KB

	for _, payloadSize := range payloadSizes {
		b.Run(fmt.Sprintf("Payload%dB", payloadSize), func(b *testing.B) {
			ctx := context.Background()

			natsContainer, err := StartNATSContainer(ctx)
			if err != nil {
				b.Fatalf("Failed to start NATS container: %v", err)
			}
			defer natsContainer.Stop(ctx)

			config := &BusConfig{
				URL:            natsContainer.URL,
				MaxReconnect:   3,
				ReconnectWait:  1 * time.Second,
				AckWait:        10 * time.Second,
				MaxInFlight:    1000,
				ConnectTimeout: 5 * time.Second,
				RequestTimeout: 5 * time.Second,
			}

			bus, err := NewNATSBus(config)
			if err != nil {
				b.Fatalf("Failed to create NATS bus: %v", err)
			}
			defer bus.Close()

			perfConfig := DefaultPerformanceConfig()
			perfConfig.MessageCount = 500 // Smaller count for large payloads
			perfConfig.Concurrency = 5
			perfConfig.PayloadSize = payloadSize
			perfConfig.WarmupMessages = 25
			perfConfig.EnableTracing = false
			perfConfig.TestDuration = 60 * time.Second

			harness := NewPerformanceHarness(bus, perfConfig)
			defer harness.Close()

			b.ResetTimer()
			result, err := harness.RunPingPongTest()
			b.StopTimer()

			if err != nil {
				b.Fatalf("Performance test failed: %v", err)
			}

			b.ReportMetric(float64(result.P95Latency.Nanoseconds())/1e6, "p95_latency_ms")
			b.ReportMetric(result.Throughput, "throughput_msg_per_sec")
			b.ReportMetric(float64(payloadSize*int(result.MessagesReceived))/result.TestDuration.Seconds()/1024/1024, "throughput_mb_per_sec")
		})
	}
}

// TestPerformanceThresholds tests that performance meets required thresholds
func TestPerformanceThresholds(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance threshold test in short mode")
	}

	ctx := context.Background()

	// Start NATS container
	natsContainer, err := StartNATSContainer(ctx)
	if err != nil {
		t.Fatalf("Failed to start NATS container: %v", err)
	}
	defer natsContainer.Stop(ctx)

	// Create bus
	config := &BusConfig{
		URL:            natsContainer.URL,
		MaxReconnect:   3,
		ReconnectWait:  1 * time.Second,
		AckWait:        10 * time.Second,
		MaxInFlight:    1000,
		ConnectTimeout: 5 * time.Second,
		RequestTimeout: 5 * time.Second,
	}

	bus, err := NewNATSBus(config)
	if err != nil {
		t.Fatalf("Failed to create NATS bus: %v", err)
	}
	defer bus.Close()

	// Configure performance test with CI thresholds
	perfConfig := DefaultPerformanceConfig()
	perfConfig.MessageCount = 1000
	perfConfig.Concurrency = 10
	perfConfig.PayloadSize = 1024
	perfConfig.WarmupMessages = 100
	perfConfig.EnableTracing = false

	// Allow environment override of thresholds
	if p95Str := os.Getenv("AF_PERF_P95_THRESHOLD_MS"); p95Str != "" {
		if p95Ms, err := strconv.Atoi(p95Str); err == nil {
			perfConfig.P95Threshold = time.Duration(p95Ms) * time.Millisecond
		}
	}

	if p50Str := os.Getenv("AF_PERF_P50_THRESHOLD_MS"); p50Str != "" {
		if p50Ms, err := strconv.Atoi(p50Str); err == nil {
			perfConfig.P50Threshold = time.Duration(p50Ms) * time.Millisecond
		}
	}

	// Run performance test
	harness := NewPerformanceHarness(bus, perfConfig)
	defer harness.Close()

	result, err := harness.RunPingPongTest()
	if err != nil {
		t.Fatalf("Performance test failed: %v", err)
	}

	// Log detailed results
	t.Logf("Performance Test Results:")
	t.Logf("  Messages: %d sent, %d received", result.MessagesSent, result.MessagesReceived)
	t.Logf("  Duration: %v", result.TestDuration)
	t.Logf("  Throughput: %.2f msg/sec", result.Throughput)
	t.Logf("  Latency Min: %v", result.MinLatency)
	t.Logf("  Latency Avg: %v", result.AvgLatency)
	t.Logf("  Latency P50: %v (threshold: %v)", result.P50Latency, perfConfig.P50Threshold)
	t.Logf("  Latency P95: %v (threshold: %v)", result.P95Latency, perfConfig.P95Threshold)
	t.Logf("  Latency P99: %v", result.P99Latency)
	t.Logf("  Latency Max: %v", result.MaxLatency)
	t.Logf("  Errors: %d publish, %d consume, %d timeout", result.PublishErrors, result.ConsumeErrors, result.TimeoutErrors)

	// Print latency distribution
	t.Logf("  Latency Distribution:")
	for bucket, count := range result.LatencyDistribution {
		percentage := float64(count) / float64(len(result.LatencyDistribution)) * 100
		t.Logf("    %s: %d (%.1f%%)", bucket, count, percentage)
	}

	// Assert thresholds
	if !result.P50Passed {
		t.Errorf("P50 latency %v exceeds threshold %v", result.P50Latency, perfConfig.P50Threshold)
	}

	if !result.P95Passed {
		t.Errorf("P95 latency %v exceeds threshold %v", result.P95Latency, perfConfig.P95Threshold)
	}

	// Check for excessive errors
	totalMessages := result.MessagesSent
	errorRate := float64(result.PublishErrors+result.ConsumeErrors) / float64(totalMessages) * 100
	if errorRate > 1.0 { // Allow up to 1% error rate
		t.Errorf("Error rate %.2f%% exceeds 1%% threshold", errorRate)
	}

	// Check message completion rate
	completionRate := float64(result.MessagesReceived) / float64(result.MessagesSent) * 100
	if completionRate < 99.0 { // Require 99% completion
		t.Errorf("Message completion rate %.2f%% is below 99%% threshold", completionRate)
	}

	if result.OverallPassed {
		t.Logf("✅ Performance test PASSED - all thresholds met")
	} else {
		t.Errorf("❌ Performance test FAILED - thresholds not met")
	}
}

// TestPerformanceRegression detects performance regressions by comparing with baseline
func TestPerformanceRegression(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance regression test in short mode")
	}

	// This test would typically compare against stored baseline metrics
	// For now, we'll just run the test and log results for manual analysis

	ctx := context.Background()

	natsContainer, err := StartNATSContainer(ctx)
	if err != nil {
		t.Fatalf("Failed to start NATS container: %v", err)
	}
	defer natsContainer.Stop(ctx)

	config := &BusConfig{
		URL:            natsContainer.URL,
		MaxReconnect:   3,
		ReconnectWait:  1 * time.Second,
		AckWait:        10 * time.Second,
		MaxInFlight:    1000,
		ConnectTimeout: 5 * time.Second,
		RequestTimeout: 5 * time.Second,
	}

	bus, err := NewNATSBus(config)
	if err != nil {
		t.Fatalf("Failed to create NATS bus: %v", err)
	}
	defer bus.Close()

	perfConfig := DefaultPerformanceConfig()
	perfConfig.MessageCount = 2000
	perfConfig.Concurrency = 20
	perfConfig.PayloadSize = 2048
	perfConfig.WarmupMessages = 200

	harness := NewPerformanceHarness(bus, perfConfig)
	defer harness.Close()

	result, err := harness.RunPingPongTest()
	if err != nil {
		t.Fatalf("Performance test failed: %v", err)
	}

	// Save results to file for regression analysis
	resultJSON, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		t.Logf("Failed to marshal results: %v", err)
	} else {
		// In a real implementation, this would be saved to a baseline file
		t.Logf("Performance baseline results:\n%s", string(resultJSON))
	}

	// For now, just log key metrics for manual comparison
	t.Logf("Regression Test Metrics:")
	t.Logf("  P95 Latency: %v", result.P95Latency)
	t.Logf("  Throughput: %.2f msg/sec", result.Throughput)
	t.Logf("  Error Rate: %.2f%%", float64(result.PublishErrors+result.ConsumeErrors)/float64(result.MessagesSent)*100)

	// TODO: Implement actual regression detection by comparing with stored baseline
	// This would involve:
	// 1. Loading previous baseline results
	// 2. Comparing current results with baseline
	// 3. Failing if performance degrades beyond acceptable threshold (e.g., 10%)
}
