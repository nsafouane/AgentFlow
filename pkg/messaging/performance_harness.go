package messaging

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/agentflow/agentflow/internal/logging"
)

// PerformanceConfig holds configuration for performance testing
type PerformanceConfig struct {
	MessageCount   int           `json:"message_count"`   // Number of messages to send
	Concurrency    int           `json:"concurrency"`     // Number of concurrent senders
	PayloadSize    int           `json:"payload_size"`    // Size of message payload in bytes
	WarmupMessages int           `json:"warmup_messages"` // Number of warmup messages (excluded from stats)
	TestDuration   time.Duration `json:"test_duration"`   // Maximum test duration
	P95Threshold   time.Duration `json:"p95_threshold"`   // P95 latency threshold for pass/fail
	P50Threshold   time.Duration `json:"p50_threshold"`   // P50 latency threshold for pass/fail
	EnableTracing  bool          `json:"enable_tracing"`  // Enable OpenTelemetry tracing during test
	Subject        string        `json:"subject"`         // NATS subject for testing
	ReportInterval time.Duration `json:"report_interval"` // Interval for progress reporting
}

// DefaultPerformanceConfig returns default performance test configuration
func DefaultPerformanceConfig() *PerformanceConfig {
	return &PerformanceConfig{
		MessageCount:   1000,
		Concurrency:    10,
		PayloadSize:    1024, // 1KB payload
		WarmupMessages: 100,
		TestDuration:   60 * time.Second,
		P95Threshold:   15 * time.Millisecond,
		P50Threshold:   5 * time.Millisecond,
		EnableTracing:  false, // Disabled by default for performance
		Subject:        "test.performance.ping-pong",
		ReportInterval: 5 * time.Second,
	}
}

// PerformanceResult holds the results of a performance test
type PerformanceResult struct {
	Config           *PerformanceConfig `json:"config"`
	MessagesSent     int64              `json:"messages_sent"`
	MessagesReceived int64              `json:"messages_received"`
	TestDuration     time.Duration      `json:"test_duration"`
	Throughput       float64            `json:"throughput_msg_per_sec"`

	// Latency statistics (in nanoseconds for precision)
	MinLatency time.Duration `json:"min_latency"`
	MaxLatency time.Duration `json:"max_latency"`
	AvgLatency time.Duration `json:"avg_latency"`
	P50Latency time.Duration `json:"p50_latency"`
	P95Latency time.Duration `json:"p95_latency"`
	P99Latency time.Duration `json:"p99_latency"`

	// Pass/fail status
	P50Passed     bool `json:"p50_passed"`
	P95Passed     bool `json:"p95_passed"`
	OverallPassed bool `json:"overall_passed"`

	// Error statistics
	PublishErrors int64 `json:"publish_errors"`
	ConsumeErrors int64 `json:"consume_errors"`
	TimeoutErrors int64 `json:"timeout_errors"`

	// Additional metrics
	LatencyDistribution map[string]int `json:"latency_distribution"` // Histogram buckets
}

// PerformanceHarness manages performance testing
type PerformanceHarness struct {
	bus    MessageBus
	config *PerformanceConfig
	logger logging.Logger

	// Test state
	startTime     time.Time
	latencies     []time.Duration
	latenciesMu   sync.RWMutex
	messagesSent  int64
	messagesRecv  int64
	publishErrors int64
	consumeErrors int64
	timeoutErrors int64

	// Synchronization
	done       chan struct{}
	wg         sync.WaitGroup
	ctx        context.Context
	cancelFunc context.CancelFunc
}

// NewPerformanceHarness creates a new performance harness
func NewPerformanceHarness(bus MessageBus, config *PerformanceConfig) *PerformanceHarness {
	if config == nil {
		config = DefaultPerformanceConfig()
	}

	ctx, cancel := context.WithTimeout(context.Background(), config.TestDuration)

	return &PerformanceHarness{
		bus:        bus,
		config:     config,
		logger:     logging.NewLogger(),
		latencies:  make([]time.Duration, 0, config.MessageCount),
		done:       make(chan struct{}),
		ctx:        ctx,
		cancelFunc: cancel,
	}
}

// RunPingPongTest executes a ping-pong performance test
func (ph *PerformanceHarness) RunPingPongTest() (*PerformanceResult, error) {
	ph.logger.Info("Starting ping-pong performance test",
		logging.Int("message_count", ph.config.MessageCount),
		logging.Int("concurrency", ph.config.Concurrency),
		logging.Int("payload_size", ph.config.PayloadSize),
		logging.String("p95_threshold", ph.config.P95Threshold.String()))

	ph.startTime = time.Now()

	// Set up ping receiver (responds with pong)
	pingSubject := ph.config.Subject + ".ping"
	pongSubject := ph.config.Subject + ".pong"

	// Start pong receiver first
	err := ph.startPongReceiver(pongSubject)
	if err != nil {
		return nil, fmt.Errorf("failed to start pong receiver: %w", err)
	}

	// Start ping responder
	err = ph.startPingResponder(pingSubject, pongSubject)
	if err != nil {
		return nil, fmt.Errorf("failed to start ping responder: %w", err)
	}

	// Wait for subscriptions to be ready
	time.Sleep(100 * time.Millisecond)

	// Start progress reporter
	go ph.reportProgress()

	// Run warmup if configured
	if ph.config.WarmupMessages > 0 {
		ph.logger.Info("Running warmup phase", logging.Int("warmup_messages", ph.config.WarmupMessages))
		err = ph.sendPingMessages(pingSubject, ph.config.WarmupMessages, true)
		if err != nil {
			return nil, fmt.Errorf("warmup failed: %w", err)
		}

		// Reset counters after warmup
		ph.resetCounters()
		ph.startTime = time.Now()
	}

	// Run actual test
	ph.logger.Info("Starting main test phase")
	err = ph.sendPingMessages(pingSubject, ph.config.MessageCount, false)
	if err != nil {
		return nil, fmt.Errorf("test failed: %w", err)
	}

	// Wait for all messages to complete or timeout
	ph.waitForCompletion()

	// Calculate and return results
	return ph.calculateResults(), nil
}

// startPongReceiver sets up the receiver that collects pong responses and measures latency
func (ph *PerformanceHarness) startPongReceiver(pongSubject string) error {
	handler := func(ctx context.Context, msg *Message) error {
		// Extract send time from metadata
		if sendTimeStr, ok := msg.Metadata["send_time"].(string); ok {
			if sendTime, err := time.Parse(time.RFC3339Nano, sendTimeStr); err == nil {
				latency := time.Since(sendTime)

				ph.latenciesMu.Lock()
				ph.latencies = append(ph.latencies, latency)
				ph.latenciesMu.Unlock()
			}
		}

		atomic.AddInt64(&ph.messagesRecv, 1)

		// Check if we've received all expected messages
		if atomic.LoadInt64(&ph.messagesRecv) >= int64(ph.config.MessageCount) {
			select {
			case ph.done <- struct{}{}:
			default:
			}
		}

		return nil
	}

	_, err := ph.bus.Subscribe(ph.ctx, pongSubject, handler)
	return err
}

// startPingResponder sets up the responder that receives pings and sends pongs
func (ph *PerformanceHarness) startPingResponder(pingSubject, pongSubject string) error {
	handler := func(ctx context.Context, msg *Message) error {
		// Create pong response
		pongMsg := NewMessage(
			fmt.Sprintf("pong-%d", time.Now().UnixNano()),
			"ping-responder",
			"pong-receiver",
			MessageTypeResponse)

		// Copy payload and metadata from ping
		pongMsg.SetPayload(msg.Payload)
		for k, v := range msg.Metadata {
			pongMsg.AddMetadata(k, v)
		}

		// Preserve trace context if tracing is enabled
		if ph.config.EnableTracing {
			pongMsg.SetTraceContext(msg.TraceID, msg.SpanID)
		}

		// Send pong response
		err := ph.bus.Publish(ctx, pongSubject, pongMsg)
		if err != nil {
			atomic.AddInt64(&ph.consumeErrors, 1)
			ph.logger.Error("Failed to send pong response", err,
				logging.String("ping_id", msg.ID))
		}

		return err
	}

	_, err := ph.bus.Subscribe(ph.ctx, pingSubject, handler)
	return err
}

// sendPingMessages sends ping messages with specified concurrency
func (ph *PerformanceHarness) sendPingMessages(pingSubject string, messageCount int, isWarmup bool) error {
	messagesPerWorker := messageCount / ph.config.Concurrency
	remainder := messageCount % ph.config.Concurrency

	// Create payload of specified size
	payload := ph.createPayload(ph.config.PayloadSize)

	// Start worker goroutines
	for i := 0; i < ph.config.Concurrency; i++ {
		workerMessages := messagesPerWorker
		if i < remainder {
			workerMessages++
		}

		ph.wg.Add(1)
		go ph.pingWorker(pingSubject, workerMessages, payload, i, isWarmup)
	}

	ph.wg.Wait()
	return nil
}

// pingWorker sends ping messages from a single worker
func (ph *PerformanceHarness) pingWorker(pingSubject string, messageCount int, payload map[string]interface{}, workerID int, isWarmup bool) {
	defer ph.wg.Done()

	for i := 0; i < messageCount; i++ {
		select {
		case <-ph.ctx.Done():
			atomic.AddInt64(&ph.timeoutErrors, 1)
			return
		default:
		}

		// Create ping message
		msgID := fmt.Sprintf("ping-%d-%d-%d", workerID, i, time.Now().UnixNano())
		pingMsg := NewMessage(msgID, "ping-sender", "ping-responder", MessageTypeRequest)

		// Set payload
		pingMsg.SetPayload(payload)

		// Add send time for latency measurement
		sendTime := time.Now()
		pingMsg.AddMetadata("send_time", sendTime.Format(time.RFC3339Nano))
		pingMsg.AddMetadata("worker_id", workerID)
		pingMsg.AddMetadata("sequence", i)
		pingMsg.AddMetadata("is_warmup", isWarmup)

		// Send ping
		err := ph.bus.Publish(ph.ctx, pingSubject, pingMsg)
		if err != nil {
			atomic.AddInt64(&ph.publishErrors, 1)
			ph.logger.Error("Failed to send ping message", err,
				logging.String("message_id", msgID),
				logging.Int("worker_id", workerID))
			continue
		}

		if !isWarmup {
			atomic.AddInt64(&ph.messagesSent, 1)
		}
	}
}

// createPayload creates a payload of the specified size
func (ph *PerformanceHarness) createPayload(size int) map[string]interface{} {
	// Create a string of approximately the right size
	dataSize := size - 100 // Reserve space for other fields
	if dataSize <= 0 {
		dataSize = 10
	}

	data := make([]byte, dataSize)
	for i := range data {
		data[i] = byte('A' + (i % 26))
	}

	return map[string]interface{}{
		"type":      "performance-test",
		"data":      string(data),
		"timestamp": time.Now().UTC(),
		"size":      size,
	}
}

// waitForCompletion waits for all messages to be processed or timeout
func (ph *PerformanceHarness) waitForCompletion() {
	timeout := time.After(ph.config.TestDuration)

	for {
		select {
		case <-ph.done:
			ph.logger.Info("All messages completed")
			return
		case <-timeout:
			atomic.AddInt64(&ph.timeoutErrors, 1)
			ph.logger.Warn("Test timed out waiting for completion",
				logging.Int("messages_sent", int(atomic.LoadInt64(&ph.messagesSent))),
				logging.Int("messages_received", int(atomic.LoadInt64(&ph.messagesRecv))))
			return
		case <-ph.ctx.Done():
			return
		}
	}
}

// reportProgress periodically reports test progress
func (ph *PerformanceHarness) reportProgress() {
	ticker := time.NewTicker(ph.config.ReportInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			sent := atomic.LoadInt64(&ph.messagesSent)
			received := atomic.LoadInt64(&ph.messagesRecv)
			elapsed := time.Since(ph.startTime)

			ph.logger.Info("Performance test progress",
				logging.Int("messages_sent", int(sent)),
				logging.Int("messages_received", int(received)),
				logging.String("elapsed", elapsed.String()),
				logging.Float64("throughput_msg_per_sec", float64(received)/elapsed.Seconds()))

		case <-ph.ctx.Done():
			return
		case <-ph.done:
			return
		}
	}
}

// resetCounters resets all counters (used after warmup)
func (ph *PerformanceHarness) resetCounters() {
	atomic.StoreInt64(&ph.messagesSent, 0)
	atomic.StoreInt64(&ph.messagesRecv, 0)
	atomic.StoreInt64(&ph.publishErrors, 0)
	atomic.StoreInt64(&ph.consumeErrors, 0)
	atomic.StoreInt64(&ph.timeoutErrors, 0)

	ph.latenciesMu.Lock()
	ph.latencies = ph.latencies[:0]
	ph.latenciesMu.Unlock()
}

// calculateResults computes final performance statistics
func (ph *PerformanceHarness) calculateResults() *PerformanceResult {
	testDuration := time.Since(ph.startTime)
	messagesSent := atomic.LoadInt64(&ph.messagesSent)
	messagesReceived := atomic.LoadInt64(&ph.messagesRecv)

	result := &PerformanceResult{
		Config:           ph.config,
		MessagesSent:     messagesSent,
		MessagesReceived: messagesReceived,
		TestDuration:     testDuration,
		PublishErrors:    atomic.LoadInt64(&ph.publishErrors),
		ConsumeErrors:    atomic.LoadInt64(&ph.consumeErrors),
		TimeoutErrors:    atomic.LoadInt64(&ph.timeoutErrors),
	}

	// Calculate throughput
	if testDuration.Seconds() > 0 {
		result.Throughput = float64(messagesReceived) / testDuration.Seconds()
	}

	// Calculate latency statistics
	ph.latenciesMu.RLock()
	latencies := make([]time.Duration, len(ph.latencies))
	copy(latencies, ph.latencies)
	ph.latenciesMu.RUnlock()

	if len(latencies) > 0 {
		// Sort latencies for percentile calculations
		sort.Slice(latencies, func(i, j int) bool {
			return latencies[i] < latencies[j]
		})

		result.MinLatency = latencies[0]
		result.MaxLatency = latencies[len(latencies)-1]

		// Calculate average
		var total time.Duration
		for _, latency := range latencies {
			total += latency
		}
		result.AvgLatency = total / time.Duration(len(latencies))

		// Calculate percentiles
		result.P50Latency = percentile(latencies, 50)
		result.P95Latency = percentile(latencies, 95)
		result.P99Latency = percentile(latencies, 99)

		// Check thresholds
		result.P50Passed = result.P50Latency <= ph.config.P50Threshold
		result.P95Passed = result.P95Latency <= ph.config.P95Threshold
		result.OverallPassed = result.P50Passed && result.P95Passed

		// Create latency distribution histogram
		result.LatencyDistribution = createLatencyHistogram(latencies)
	}

	return result
}

// percentile calculates the specified percentile from sorted latencies
func percentile(sortedLatencies []time.Duration, p int) time.Duration {
	if len(sortedLatencies) == 0 {
		return 0
	}

	index := int(math.Ceil(float64(len(sortedLatencies)*p)/100.0)) - 1
	if index < 0 {
		index = 0
	}
	if index >= len(sortedLatencies) {
		index = len(sortedLatencies) - 1
	}

	return sortedLatencies[index]
}

// createLatencyHistogram creates a histogram of latency distribution
func createLatencyHistogram(latencies []time.Duration) map[string]int {
	buckets := map[string]int{
		"<1ms":     0,
		"1-5ms":    0,
		"5-10ms":   0,
		"10-15ms":  0,
		"15-25ms":  0,
		"25-50ms":  0,
		"50-100ms": 0,
		">100ms":   0,
	}

	for _, latency := range latencies {
		ms := float64(latency.Nanoseconds()) / 1e6

		switch {
		case ms < 1:
			buckets["<1ms"]++
		case ms < 5:
			buckets["1-5ms"]++
		case ms < 10:
			buckets["5-10ms"]++
		case ms < 15:
			buckets["10-15ms"]++
		case ms < 25:
			buckets["15-25ms"]++
		case ms < 50:
			buckets["25-50ms"]++
		case ms < 100:
			buckets["50-100ms"]++
		default:
			buckets[">100ms"]++
		}
	}

	return buckets
}

// Close cleans up the performance harness
func (ph *PerformanceHarness) Close() {
	ph.cancelFunc()
	close(ph.done)
}
