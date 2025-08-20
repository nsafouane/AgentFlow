//go:build manual

package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"
)

// TestManualPerformanceBenchmark runs a comprehensive performance benchmark manually
// Run with: go test -tags=manual -v ./pkg/messaging -run TestManualPerformanceBenchmark
func TestManualPerformanceBenchmark(t *testing.T) {
	fmt.Println("=== AgentFlow Messaging Performance Benchmark ===")
	fmt.Println("This test measures message routing performance using ping-pong pattern")
	fmt.Println()

	ctx := context.Background()

	// Start NATS container
	fmt.Println("Starting NATS container...")
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

	fmt.Printf("Connected to NATS at %s\n", config.URL)
	fmt.Println()

	// Run multiple benchmark scenarios
	scenarios := []struct {
		name        string
		config      *PerformanceConfig
		description string
	}{
		{
			name: "Baseline",
			config: &PerformanceConfig{
				MessageCount:   1000,
				Concurrency:    10,
				PayloadSize:    1024,
				WarmupMessages: 100,
				TestDuration:   60 * time.Second,
				P95Threshold:   15 * time.Millisecond,
				P50Threshold:   5 * time.Millisecond,
				EnableTracing:  false,
				Subject:        "test.performance.baseline",
				ReportInterval: 5 * time.Second,
			},
			description: "Standard benchmark with 1KB messages, 10 concurrent senders",
		},
		{
			name: "HighConcurrency",
			config: &PerformanceConfig{
				MessageCount:   2000,
				Concurrency:    50,
				PayloadSize:    1024,
				WarmupMessages: 200,
				TestDuration:   60 * time.Second,
				P95Threshold:   25 * time.Millisecond, // Higher threshold for high concurrency
				P50Threshold:   10 * time.Millisecond,
				EnableTracing:  false,
				Subject:        "test.performance.highconcurrency",
				ReportInterval: 5 * time.Second,
			},
			description: "High concurrency test with 50 concurrent senders",
		},
		{
			name: "LargePayload",
			config: &PerformanceConfig{
				MessageCount:   500,
				Concurrency:    5,
				PayloadSize:    16384, // 16KB
				WarmupMessages: 50,
				TestDuration:   60 * time.Second,
				P95Threshold:   30 * time.Millisecond, // Higher threshold for large payloads
				P50Threshold:   15 * time.Millisecond,
				EnableTracing:  false,
				Subject:        "test.performance.largepayload",
				ReportInterval: 5 * time.Second,
			},
			description: "Large payload test with 16KB messages",
		},
		{
			name: "WithTracing",
			config: &PerformanceConfig{
				MessageCount:   500,
				Concurrency:    5,
				PayloadSize:    1024,
				WarmupMessages: 50,
				TestDuration:   60 * time.Second,
				P95Threshold:   20 * time.Millisecond, // Higher threshold with tracing overhead
				P50Threshold:   8 * time.Millisecond,
				EnableTracing:  true,
				Subject:        "test.performance.tracing",
				ReportInterval: 5 * time.Second,
			},
			description: "Performance with OpenTelemetry tracing enabled",
		},
	}

	results := make(map[string]*PerformanceResult)

	for _, scenario := range scenarios {
		fmt.Printf("=== Running %s Scenario ===\n", scenario.name)
		fmt.Printf("Description: %s\n", scenario.description)
		fmt.Printf("Configuration:\n")
		fmt.Printf("  Messages: %d\n", scenario.config.MessageCount)
		fmt.Printf("  Concurrency: %d\n", scenario.config.Concurrency)
		fmt.Printf("  Payload Size: %d bytes\n", scenario.config.PayloadSize)
		fmt.Printf("  P95 Threshold: %v\n", scenario.config.P95Threshold)
		fmt.Printf("  Tracing: %t\n", scenario.config.EnableTracing)
		fmt.Println()

		// Run the test
		harness := NewPerformanceHarness(bus, scenario.config)
		result, err := harness.RunPingPongTest()
		harness.Close()

		if err != nil {
			fmt.Printf("❌ Scenario %s FAILED: %v\n", scenario.name, err)
			continue
		}

		results[scenario.name] = result

		// Print results
		fmt.Printf("Results for %s:\n", scenario.name)
		fmt.Printf("  Messages: %d sent, %d received (%.1f%% completion)\n",
			result.MessagesSent, result.MessagesReceived,
			float64(result.MessagesReceived)/float64(result.MessagesSent)*100)
		fmt.Printf("  Duration: %v\n", result.TestDuration)
		fmt.Printf("  Throughput: %.2f msg/sec\n", result.Throughput)
		fmt.Printf("  Latency:\n")
		fmt.Printf("    Min: %v\n", result.MinLatency)
		fmt.Printf("    Avg: %v\n", result.AvgLatency)
		fmt.Printf("    P50: %v (threshold: %v) %s\n", result.P50Latency, scenario.config.P50Threshold, passFailIcon(result.P50Passed))
		fmt.Printf("    P95: %v (threshold: %v) %s\n", result.P95Latency, scenario.config.P95Threshold, passFailIcon(result.P95Passed))
		fmt.Printf("    P99: %v\n", result.P99Latency)
		fmt.Printf("    Max: %v\n", result.MaxLatency)
		fmt.Printf("  Errors: %d publish, %d consume, %d timeout\n",
			result.PublishErrors, result.ConsumeErrors, result.TimeoutErrors)

		// Print latency distribution
		fmt.Printf("  Latency Distribution:\n")
		for bucket, count := range result.LatencyDistribution {
			percentage := float64(count) / float64(result.MessagesReceived) * 100
			fmt.Printf("    %8s: %4d (%5.1f%%)\n", bucket, count, percentage)
		}

		fmt.Printf("  Overall: %s\n", passFailIcon(result.OverallPassed))
		fmt.Println()
	}

	// Print summary comparison
	fmt.Println("=== Performance Summary ===")
	fmt.Printf("%-15s %10s %10s %10s %10s %8s\n", "Scenario", "Throughput", "P50", "P95", "Errors", "Status")
	fmt.Printf("%-15s %10s %10s %10s %10s %8s\n", "--------", "----------", "---", "---", "------", "------")

	for _, scenario := range scenarios {
		result, ok := results[scenario.name]
		if !ok {
			fmt.Printf("%-15s %10s %10s %10s %10s %8s\n", scenario.name, "FAILED", "-", "-", "-", "❌")
			continue
		}

		errorRate := float64(result.PublishErrors+result.ConsumeErrors) / float64(result.MessagesSent) * 100
		fmt.Printf("%-15s %8.1f/s %8.1fms %8.1fms %7.1f%% %8s\n",
			scenario.name,
			result.Throughput,
			float64(result.P50Latency.Nanoseconds())/1e6,
			float64(result.P95Latency.Nanoseconds())/1e6,
			errorRate,
			passFailIcon(result.OverallPassed))
	}

	fmt.Println()

	// Save detailed results to file
	allResults := map[string]interface{}{
		"timestamp": time.Now().UTC(),
		"scenarios": results,
		"environment": map[string]interface{}{
			"nats_url":  config.URL,
			"test_type": "manual_benchmark",
		},
	}

	resultJSON, err := json.MarshalIndent(allResults, "", "  ")
	if err != nil {
		fmt.Printf("Failed to marshal results: %v\n", err)
	} else {
		filename := fmt.Sprintf("performance_results_%s.json", time.Now().Format("20060102_150405"))
		err = os.WriteFile(filename, resultJSON, 0644)
		if err != nil {
			fmt.Printf("Failed to write results file: %v\n", err)
		} else {
			fmt.Printf("Detailed results saved to: %s\n", filename)
		}
	}

	// Check if any critical thresholds were exceeded
	criticalFailures := 0
	for name, result := range results {
		if !result.OverallPassed {
			criticalFailures++
			fmt.Printf("⚠️  Critical failure in %s scenario\n", name)
		}
	}

	if criticalFailures > 0 {
		fmt.Printf("\n❌ %d scenarios failed critical thresholds\n", criticalFailures)
	} else {
		fmt.Printf("\n✅ All scenarios passed performance thresholds\n")
	}

	fmt.Println("\n=== Manual Performance Benchmark Complete ===")
}

// TestManualBaselineRecording records baseline performance metrics
// Run with: go test -tags=manual -v ./pkg/messaging -run TestManualBaselineRecording
func TestManualBaselineRecording(t *testing.T) {
	fmt.Println("=== Recording Performance Baseline ===")
	fmt.Println("This test records baseline performance metrics for regression detection")
	fmt.Println()

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

	// Run multiple iterations to get stable baseline
	iterations := 5
	fmt.Printf("Running %d iterations for stable baseline...\n", iterations)

	var allResults []*PerformanceResult

	for i := 0; i < iterations; i++ {
		fmt.Printf("Iteration %d/%d...\n", i+1, iterations)

		perfConfig := &PerformanceConfig{
			MessageCount:   2000,
			Concurrency:    20,
			PayloadSize:    2048,
			WarmupMessages: 200,
			TestDuration:   60 * time.Second,
			P95Threshold:   15 * time.Millisecond,
			P50Threshold:   5 * time.Millisecond,
			EnableTracing:  false,
			Subject:        fmt.Sprintf("test.baseline.%d", i),
			ReportInterval: 10 * time.Second,
		}

		harness := NewPerformanceHarness(bus, perfConfig)
		result, err := harness.RunPingPongTest()
		harness.Close()

		if err != nil {
			fmt.Printf("Iteration %d failed: %v\n", i+1, err)
			continue
		}

		allResults = append(allResults, result)

		fmt.Printf("  P95: %v, Throughput: %.1f msg/sec\n",
			result.P95Latency, result.Throughput)

		// Small delay between iterations
		time.Sleep(2 * time.Second)
	}

	if len(allResults) == 0 {
		t.Fatalf("No successful iterations")
	}

	// Calculate baseline statistics
	var totalThroughput float64
	var totalP50, totalP95 time.Duration

	for _, result := range allResults {
		totalThroughput += result.Throughput
		totalP50 += result.P50Latency
		totalP95 += result.P95Latency
	}

	baseline := map[string]interface{}{
		"timestamp":       time.Now().UTC(),
		"iterations":      len(allResults),
		"avg_throughput":  totalThroughput / float64(len(allResults)),
		"avg_p50_latency": totalP50 / time.Duration(len(allResults)),
		"avg_p95_latency": totalP95 / time.Duration(len(allResults)),
		"all_results":     allResults,
		"test_config": map[string]interface{}{
			"message_count": 2000,
			"concurrency":   20,
			"payload_size":  2048,
		},
	}

	// Save baseline
	baselineJSON, err := json.MarshalIndent(baseline, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal baseline: %v", err)
	}

	filename := fmt.Sprintf("performance_baseline_%s.json", time.Now().Format("20060102_150405"))
	err = os.WriteFile(filename, baselineJSON, 0644)
	if err != nil {
		t.Fatalf("Failed to write baseline file: %v", err)
	}

	fmt.Printf("\nBaseline Results:\n")
	fmt.Printf("  Iterations: %d\n", len(allResults))
	fmt.Printf("  Avg Throughput: %.2f msg/sec\n", baseline["avg_throughput"])
	fmt.Printf("  Avg P50 Latency: %v\n", baseline["avg_p50_latency"])
	fmt.Printf("  Avg P95 Latency: %v\n", baseline["avg_p95_latency"])
	fmt.Printf("  Baseline saved to: %s\n", filename)

	fmt.Println("\n=== Baseline Recording Complete ===")
}

// passFailIcon returns an icon indicating pass/fail status
func passFailIcon(passed bool) string {
	if passed {
		return "✅"
	}
	return "❌"
}
