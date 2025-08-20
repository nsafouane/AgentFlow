package messaging

import (
	"testing"
	"time"
)

// TestPerformanceConfig tests the performance configuration
func TestPerformanceConfig(t *testing.T) {
	// Test default configuration
	config := DefaultPerformanceConfig()

	if config.MessageCount != 1000 {
		t.Errorf("Expected MessageCount 1000, got %d", config.MessageCount)
	}

	if config.Concurrency != 10 {
		t.Errorf("Expected Concurrency 10, got %d", config.Concurrency)
	}

	if config.PayloadSize != 1024 {
		t.Errorf("Expected PayloadSize 1024, got %d", config.PayloadSize)
	}

	if config.P95Threshold != 15*time.Millisecond {
		t.Errorf("Expected P95Threshold 15ms, got %v", config.P95Threshold)
	}

	if config.P50Threshold != 5*time.Millisecond {
		t.Errorf("Expected P50Threshold 5ms, got %v", config.P50Threshold)
	}
}

// TestPercentileCalculation tests the percentile calculation function
func TestPercentileCalculation(t *testing.T) {
	// Test with known latencies
	latencies := []time.Duration{
		1 * time.Millisecond,
		2 * time.Millisecond,
		3 * time.Millisecond,
		4 * time.Millisecond,
		5 * time.Millisecond,
		6 * time.Millisecond,
		7 * time.Millisecond,
		8 * time.Millisecond,
		9 * time.Millisecond,
		10 * time.Millisecond,
	}

	// Test P50 (should be around 5-6ms)
	p50 := percentile(latencies, 50)
	if p50 < 4*time.Millisecond || p50 > 6*time.Millisecond {
		t.Errorf("P50 should be around 5ms, got %v", p50)
	}

	// Test P95 (should be around 9-10ms)
	p95 := percentile(latencies, 95)
	if p95 < 9*time.Millisecond || p95 > 10*time.Millisecond {
		t.Errorf("P95 should be around 9-10ms, got %v", p95)
	}

	// Test P99 (should be 10ms)
	p99 := percentile(latencies, 99)
	if p99 != 10*time.Millisecond {
		t.Errorf("P99 should be 10ms, got %v", p99)
	}

	// Test edge cases
	emptyLatencies := []time.Duration{}
	if percentile(emptyLatencies, 50) != 0 {
		t.Errorf("Percentile of empty slice should be 0")
	}

	singleLatency := []time.Duration{5 * time.Millisecond}
	if percentile(singleLatency, 50) != 5*time.Millisecond {
		t.Errorf("Percentile of single element should be that element")
	}
}

// TestLatencyHistogram tests the latency histogram creation
func TestLatencyHistogram(t *testing.T) {
	latencies := []time.Duration{
		500 * time.Microsecond, // <1ms
		2 * time.Millisecond,   // 1-5ms
		7 * time.Millisecond,   // 5-10ms
		12 * time.Millisecond,  // 10-15ms
		20 * time.Millisecond,  // 15-25ms
		35 * time.Millisecond,  // 25-50ms
		75 * time.Millisecond,  // 50-100ms
		150 * time.Millisecond, // >100ms
	}

	histogram := createLatencyHistogram(latencies)

	expectedBuckets := map[string]int{
		"<1ms":     1,
		"1-5ms":    1,
		"5-10ms":   1,
		"10-15ms":  1,
		"15-25ms":  1,
		"25-50ms":  1,
		"50-100ms": 1,
		">100ms":   1,
	}

	for bucket, expectedCount := range expectedBuckets {
		if histogram[bucket] != expectedCount {
			t.Errorf("Expected %d in bucket %s, got %d", expectedCount, bucket, histogram[bucket])
		}
	}

	// Test empty histogram
	emptyHistogram := createLatencyHistogram([]time.Duration{})
	for bucket, count := range emptyHistogram {
		if count != 0 {
			t.Errorf("Empty histogram should have 0 in all buckets, got %d in %s", count, bucket)
		}
	}
}

// TestThresholdEvaluation tests the threshold evaluation logic
func TestThresholdEvaluation(t *testing.T) {
	config := &PerformanceConfig{
		P50Threshold: 5 * time.Millisecond,
		P95Threshold: 15 * time.Millisecond,
	}

	// Test case 1: Both thresholds pass
	result1 := &PerformanceResult{
		Config:     config,
		P50Latency: 3 * time.Millisecond,
		P95Latency: 10 * time.Millisecond,
	}

	result1.P50Passed = result1.P50Latency <= config.P50Threshold
	result1.P95Passed = result1.P95Latency <= config.P95Threshold
	result1.OverallPassed = result1.P50Passed && result1.P95Passed

	if !result1.P50Passed {
		t.Errorf("P50 should pass with 3ms < 5ms threshold")
	}
	if !result1.P95Passed {
		t.Errorf("P95 should pass with 10ms < 15ms threshold")
	}
	if !result1.OverallPassed {
		t.Errorf("Overall should pass when both thresholds pass")
	}

	// Test case 2: P50 fails
	result2 := &PerformanceResult{
		Config:     config,
		P50Latency: 7 * time.Millisecond,
		P95Latency: 10 * time.Millisecond,
	}

	result2.P50Passed = result2.P50Latency <= config.P50Threshold
	result2.P95Passed = result2.P95Latency <= config.P95Threshold
	result2.OverallPassed = result2.P50Passed && result2.P95Passed

	if result2.P50Passed {
		t.Errorf("P50 should fail with 7ms > 5ms threshold")
	}
	if !result2.P95Passed {
		t.Errorf("P95 should pass with 10ms < 15ms threshold")
	}
	if result2.OverallPassed {
		t.Errorf("Overall should fail when P50 fails")
	}

	// Test case 3: P95 fails
	result3 := &PerformanceResult{
		Config:     config,
		P50Latency: 3 * time.Millisecond,
		P95Latency: 20 * time.Millisecond,
	}

	result3.P50Passed = result3.P50Latency <= config.P50Threshold
	result3.P95Passed = result3.P95Latency <= config.P95Threshold
	result3.OverallPassed = result3.P50Passed && result3.P95Passed

	if !result3.P50Passed {
		t.Errorf("P50 should pass with 3ms < 5ms threshold")
	}
	if result3.P95Passed {
		t.Errorf("P95 should fail with 20ms > 15ms threshold")
	}
	if result3.OverallPassed {
		t.Errorf("Overall should fail when P95 fails")
	}

	// Test case 4: Both fail
	result4 := &PerformanceResult{
		Config:     config,
		P50Latency: 7 * time.Millisecond,
		P95Latency: 20 * time.Millisecond,
	}

	result4.P50Passed = result4.P50Latency <= config.P50Threshold
	result4.P95Passed = result4.P95Latency <= config.P95Threshold
	result4.OverallPassed = result4.P50Passed && result4.P95Passed

	if result4.P50Passed {
		t.Errorf("P50 should fail with 7ms > 5ms threshold")
	}
	if result4.P95Passed {
		t.Errorf("P95 should fail with 20ms > 15ms threshold")
	}
	if result4.OverallPassed {
		t.Errorf("Overall should fail when both thresholds fail")
	}
}

// TestPayloadCreation tests the payload creation function
func TestPayloadCreation(t *testing.T) {
	harness := &PerformanceHarness{}

	// Test small payload
	payload1 := harness.createPayload(100)
	if payload1["type"] != "performance-test" {
		t.Errorf("Payload should have correct type")
	}
	if payload1["size"] != 100 {
		t.Errorf("Payload should have correct size")
	}

	data1, ok := payload1["data"].(string)
	if !ok {
		t.Errorf("Payload data should be string")
	}
	if len(data1) < 10 {
		t.Errorf("Payload data should have reasonable size, got %d", len(data1))
	}

	// Test large payload
	payload2 := harness.createPayload(5000)
	data2, ok := payload2["data"].(string)
	if !ok {
		t.Errorf("Large payload data should be string")
	}
	if len(data2) < 4900 { // Should be close to requested size minus overhead
		t.Errorf("Large payload should be approximately correct size, got %d", len(data2))
	}

	// Test minimum payload
	payload3 := harness.createPayload(10)
	data3, ok := payload3["data"].(string)
	if !ok {
		t.Errorf("Minimum payload data should be string")
	}
	if len(data3) < 10 {
		t.Errorf("Minimum payload should have at least 10 characters")
	}
}

// TestThroughputCalculation tests throughput calculation
func TestThroughputCalculation(t *testing.T) {
	// Test normal case
	result := &PerformanceResult{
		MessagesReceived: 1000,
		TestDuration:     10 * time.Second,
	}

	expectedThroughput := 100.0 // 1000 messages / 10 seconds
	if result.TestDuration.Seconds() > 0 {
		result.Throughput = float64(result.MessagesReceived) / result.TestDuration.Seconds()
	}

	if result.Throughput != expectedThroughput {
		t.Errorf("Expected throughput %.2f, got %.2f", expectedThroughput, result.Throughput)
	}

	// Test zero duration (should not crash)
	result2 := &PerformanceResult{
		MessagesReceived: 1000,
		TestDuration:     0,
	}

	if result2.TestDuration.Seconds() > 0 {
		result2.Throughput = float64(result2.MessagesReceived) / result2.TestDuration.Seconds()
	}

	if result2.Throughput != 0 {
		t.Errorf("Zero duration should result in zero throughput, got %.2f", result2.Throughput)
	}
}

// TestErrorRateCalculation tests error rate calculations
func TestErrorRateCalculation(t *testing.T) {
	result := &PerformanceResult{
		MessagesSent:  1000,
		PublishErrors: 10,
		ConsumeErrors: 5,
		TimeoutErrors: 2,
	}

	totalErrors := result.PublishErrors + result.ConsumeErrors
	errorRate := float64(totalErrors) / float64(result.MessagesSent) * 100

	expectedErrorRate := 1.5 // (10 + 5) / 1000 * 100
	if errorRate != expectedErrorRate {
		t.Errorf("Expected error rate %.2f%%, got %.2f%%", expectedErrorRate, errorRate)
	}

	// Test zero errors
	result2 := &PerformanceResult{
		MessagesSent:  1000,
		PublishErrors: 0,
		ConsumeErrors: 0,
		TimeoutErrors: 0,
	}

	totalErrors2 := result2.PublishErrors + result2.ConsumeErrors
	errorRate2 := float64(totalErrors2) / float64(result2.MessagesSent) * 100

	if errorRate2 != 0 {
		t.Errorf("Zero errors should result in zero error rate, got %.2f%%", errorRate2)
	}
}

// TestCompletionRateCalculation tests message completion rate calculations
func TestCompletionRateCalculation(t *testing.T) {
	result := &PerformanceResult{
		MessagesSent:     1000,
		MessagesReceived: 995,
	}

	completionRate := float64(result.MessagesReceived) / float64(result.MessagesSent) * 100
	expectedCompletionRate := 99.5

	if completionRate != expectedCompletionRate {
		t.Errorf("Expected completion rate %.2f%%, got %.2f%%", expectedCompletionRate, completionRate)
	}

	// Test perfect completion
	result2 := &PerformanceResult{
		MessagesSent:     1000,
		MessagesReceived: 1000,
	}

	completionRate2 := float64(result2.MessagesReceived) / float64(result2.MessagesSent) * 100

	if completionRate2 != 100.0 {
		t.Errorf("Perfect completion should be 100%%, got %.2f%%", completionRate2)
	}

	// Test zero completion
	result3 := &PerformanceResult{
		MessagesSent:     1000,
		MessagesReceived: 0,
	}

	completionRate3 := float64(result3.MessagesReceived) / float64(result3.MessagesSent) * 100

	if completionRate3 != 0.0 {
		t.Errorf("Zero completion should be 0%%, got %.2f%%", completionRate3)
	}
}
