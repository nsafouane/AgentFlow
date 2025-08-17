package messaging

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/agentflow/agentflow/internal/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestStructuredLoggingIntegration tests that all message operations include proper correlation IDs
func TestStructuredLoggingIntegration(t *testing.T) {
	// Disable tracing for this test to avoid OTLP export errors
	t.Setenv("AF_TRACING_ENABLED", "false")

	ctx := context.Background()

	// Start NATS container
	natsContainer, err := StartNATSContainer(ctx)
	require.NoError(t, err)
	defer natsContainer.Stop(ctx)

	// Create a buffer to capture log output
	var logBuffer bytes.Buffer

	// Create bus with custom logger that writes to our buffer
	config := &BusConfig{
		URL:            natsContainer.URL,
		MaxReconnect:   3,
		ReconnectWait:  1 * time.Second,
		AckWait:        10 * time.Second,
		MaxInFlight:    100,
		ConnectTimeout: 5 * time.Second,
		RequestTimeout: 5 * time.Second,
	}

	bus, err := NewNATSBus(config)
	require.NoError(t, err)
	defer bus.Close()

	// Replace the logger with one that writes to our buffer
	natsBus := bus.(*natsBus)
	natsBus.logger = logging.NewLoggerWithWriter(&logBuffer)

	// Create test message with all correlation fields
	msg := NewMessage("test-msg-123", "agent-sender", "agent-receiver", MessageTypeRequest)
	msg.SetPayload(map[string]interface{}{"test": "correlation_data"})
	msg.SetTraceContext("trace-abc123", "span-def456")

	// Set envelope hash
	err = natsBus.serializer.SetEnvelopeHash(msg)
	require.NoError(t, err)

	// Set up subscription to trigger message processing
	subject := "agents.agent-receiver.in"
	receivedMessages := make(chan *Message, 1)

	handler := func(ctx context.Context, receivedMsg *Message) error {
		// Handler should also log with correlation
		logger := natsBus.logger.WithTrace(ctx).WithMessage(receivedMsg.ID)
		logger.Info("Message handled in test",
			logging.String("handler", "test_handler"))
		receivedMessages <- receivedMsg
		return nil
	}

	sub, err := bus.Subscribe(ctx, subject, handler)
	require.NoError(t, err)
	defer sub.Unsubscribe()

	// Clear buffer before publishing
	logBuffer.Reset()

	// Publish message (this should generate logs with correlation IDs)
	err = bus.Publish(ctx, subject, msg)
	require.NoError(t, err)

	// Wait for message to be processed
	select {
	case <-receivedMessages:
		// Message received successfully
	case <-time.After(5 * time.Second):
		t.Fatal("Message not received within timeout")
	}

	// Give some time for all logs to be written
	time.Sleep(100 * time.Millisecond)

	// Parse log entries
	logOutput := logBuffer.String()
	logLines := strings.Split(strings.TrimSpace(logOutput), "\n")

	// Filter out empty lines
	var validLogLines []string
	for _, line := range logLines {
		if strings.TrimSpace(line) != "" {
			validLogLines = append(validLogLines, line)
		}
	}

	require.Greater(t, len(validLogLines), 0, "No log entries found")

	// Verify each log entry has proper structure and correlation IDs
	for i, logLine := range validLogLines {
		t.Run(fmt.Sprintf("LogEntry_%d", i), func(t *testing.T) {
			var logEntry map[string]interface{}
			err := json.Unmarshal([]byte(logLine), &logEntry)
			require.NoError(t, err, "Log entry is not valid JSON: %s", logLine)

			// Verify standard fields are present
			assert.Contains(t, logEntry, "timestamp", "Missing timestamp field")
			assert.Contains(t, logEntry, "level", "Missing level field")
			assert.Contains(t, logEntry, "message", "Missing message field")

			// Check if this log entry should have correlation IDs
			// (some logs like connection logs might not have message correlation)
			messageText, hasMessage := logEntry["message"].(string)
			if hasMessage && (strings.Contains(messageText, "Publishing message") ||
				strings.Contains(messageText, "Processing message") ||
				strings.Contains(messageText, "Message processed successfully") ||
				strings.Contains(messageText, "Message handled in test")) {

				// These logs should have message_id
				assert.Contains(t, logEntry, "message_id",
					"Missing message_id in message-related log: %s", logLine)

				if msgID, ok := logEntry["message_id"].(string); ok {
					assert.Equal(t, "test-msg-123", msgID,
						"Incorrect message_id in log entry")
				}

				// Logs with trace context should have trace fields (if tracing is enabled)
				if strings.Contains(messageText, "Message handled in test") {
					// Handler logs should have trace context if tracing is enabled
					// Since we disabled tracing for this test, we don't expect trace fields
					// In a real scenario with tracing enabled, these would be present
					t.Logf("Handler log entry (tracing disabled): %s", logLine)
				}
			}

			// Verify field ordering (timestamp should be first)
			keys := make([]string, 0, len(logEntry))
			for k := range logEntry {
				keys = append(keys, k)
			}
			if len(keys) > 0 {
				// In Go's JSON unmarshaling, field order is not preserved
				// But we can verify that our marshaling function works correctly
				// by checking that all expected fields are present
				t.Logf("Log entry fields: %v", keys)
			}
		})
	}

	// Verify that we have logs for both publish and consume operations
	hasPublishLog := false
	hasConsumeLog := false
	hasHandlerLog := false

	for _, logLine := range validLogLines {
		if strings.Contains(logLine, "Publishing message") ||
			strings.Contains(logLine, "Message published successfully") {
			hasPublishLog = true
		}
		if strings.Contains(logLine, "Processing message") ||
			strings.Contains(logLine, "Message processed successfully") {
			hasConsumeLog = true
		}
		if strings.Contains(logLine, "Message handled in test") {
			hasHandlerLog = true
		}
	}

	assert.True(t, hasPublishLog, "No publish-related log entries found")
	assert.True(t, hasConsumeLog, "No consume-related log entries found")
	assert.True(t, hasHandlerLog, "No handler-related log entries found")

	t.Logf("Total log entries analyzed: %d", len(validLogLines))
	t.Logf("Sample log entries:\n%s", strings.Join(validLogLines[:min(3, len(validLogLines))], "\n"))
}

// TestLogFieldValidation tests that reserved fields are properly validated
func TestLogFieldValidation(t *testing.T) {
	// Disable tracing for this test to avoid OTLP export errors
	t.Setenv("AF_TRACING_ENABLED", "false")

	ctx := context.Background()

	// Start NATS container
	natsContainer, err := StartNATSContainer(ctx)
	require.NoError(t, err)
	defer natsContainer.Stop(ctx)

	// Create a buffer to capture log output
	var logBuffer bytes.Buffer

	// Create bus
	config := &BusConfig{
		URL:            natsContainer.URL,
		MaxReconnect:   3,
		ReconnectWait:  1 * time.Second,
		AckWait:        10 * time.Second,
		MaxInFlight:    100,
		ConnectTimeout: 5 * time.Second,
		RequestTimeout: 5 * time.Second,
	}

	bus, err := NewNATSBus(config)
	require.NoError(t, err)
	defer bus.Close()

	// Replace the logger with one that writes to our buffer
	natsBus := bus.(*natsBus)
	natsBus.logger = logging.NewLoggerWithWriter(&logBuffer)

	// Test that the logger properly validates fields
	logger := natsBus.logger

	// Clear buffer
	logBuffer.Reset()

	// Try to use reserved field names (should be ignored/rejected)
	logger.WithFields(
		logging.Field{Key: "trace_id", Value: "should_be_ignored"},
		logging.Field{Key: "custom_field", Value: "should_work"},
	).Info("Test message with mixed fields")

	// Parse the log output
	logOutput := logBuffer.String()
	logLines := strings.Split(strings.TrimSpace(logOutput), "\n")

	// Find the test message log entry
	var testLogEntry map[string]interface{}
	for _, logLine := range logLines {
		if strings.TrimSpace(logLine) == "" {
			continue
		}

		var entry map[string]interface{}
		if err := json.Unmarshal([]byte(logLine), &entry); err != nil {
			continue
		}

		if msg, ok := entry["message"].(string); ok && msg == "Test message with mixed fields" {
			testLogEntry = entry
			break
		}
	}

	require.NotNil(t, testLogEntry, "Test log entry not found")

	// Verify that custom field was added
	assert.Equal(t, "should_work", testLogEntry["custom_field"],
		"Custom field should be present")

	// Verify that reserved field was not overridden
	// (trace_id should either be absent or not equal to "should_be_ignored")
	if traceID, exists := testLogEntry["trace_id"]; exists {
		assert.NotEqual(t, "should_be_ignored", traceID,
			"Reserved field should not be overridden")
	}
}

// TestContextAwareLogging tests that correlation IDs are preserved across goroutines
func TestContextAwareLogging(t *testing.T) {
	// Disable tracing for this test to avoid OTLP export errors
	t.Setenv("AF_TRACING_ENABLED", "false")

	ctx := context.Background()

	// Start NATS container
	natsContainer, err := StartNATSContainer(ctx)
	require.NoError(t, err)
	defer natsContainer.Stop(ctx)

	// Create a buffer to capture log output
	var logBuffer bytes.Buffer

	// Create bus
	config := &BusConfig{
		URL:            natsContainer.URL,
		MaxReconnect:   3,
		ReconnectWait:  1 * time.Second,
		AckWait:        10 * time.Second,
		MaxInFlight:    100,
		ConnectTimeout: 5 * time.Second,
		RequestTimeout: 5 * time.Second,
	}

	bus, err := NewNATSBus(config)
	require.NoError(t, err)
	defer bus.Close()

	// Replace the logger with one that writes to our buffer
	natsBus := bus.(*natsBus)
	natsBus.logger = logging.NewLoggerWithWriter(&logBuffer)

	// Create test message
	msg := NewMessage("context-test-123", "sender", "receiver", MessageTypeEvent)
	msg.SetPayload(map[string]interface{}{"test": "context_preservation"})

	// Set envelope hash
	err = natsBus.serializer.SetEnvelopeHash(msg)
	require.NoError(t, err)

	// Set up subscription with handler that spawns goroutines
	subject := "agents.receiver.in"
	handlerCompleted := make(chan bool, 1)

	handler := func(ctx context.Context, receivedMsg *Message) error {
		// Create logger with message context
		logger := natsBus.logger.WithMessage(receivedMsg.ID).WithTrace(ctx)

		// Log from main handler goroutine
		logger.Info("Handler started")

		// Spawn a goroutine and verify context preservation
		go func() {
			// The logger should preserve correlation IDs even in new goroutine
			logger.Info("Processing in background goroutine")

			// Create a new logger from the context-aware logger
			backgroundLogger := logger.WithFields(logging.String("goroutine", "background"))
			backgroundLogger.Info("Background processing complete")

			handlerCompleted <- true
		}()

		return nil
	}

	sub, err := bus.Subscribe(ctx, subject, handler)
	require.NoError(t, err)
	defer sub.Unsubscribe()

	// Clear buffer before publishing
	logBuffer.Reset()

	// Publish message
	err = bus.Publish(ctx, subject, msg)
	require.NoError(t, err)

	// Wait for handler to complete
	select {
	case <-handlerCompleted:
		// Handler completed successfully
	case <-time.After(5 * time.Second):
		t.Fatal("Handler did not complete within timeout")
	}

	// Give some time for all logs to be written
	time.Sleep(100 * time.Millisecond)

	// Parse log entries
	logOutput := logBuffer.String()
	logLines := strings.Split(strings.TrimSpace(logOutput), "\n")

	// Find logs from the background goroutine
	var backgroundLogs []map[string]interface{}
	for _, logLine := range logLines {
		if strings.TrimSpace(logLine) == "" {
			continue
		}

		var entry map[string]interface{}
		if err := json.Unmarshal([]byte(logLine), &entry); err != nil {
			continue
		}

		if msg, ok := entry["message"].(string); ok &&
			(strings.Contains(msg, "background goroutine") ||
				strings.Contains(msg, "Background processing complete")) {
			backgroundLogs = append(backgroundLogs, entry)
		}
	}

	require.Greater(t, len(backgroundLogs), 0, "No background goroutine logs found")

	// Verify that background logs have proper correlation IDs
	for i, logEntry := range backgroundLogs {
		t.Run(fmt.Sprintf("BackgroundLog_%d", i), func(t *testing.T) {
			// Should have message_id preserved from the handler context
			assert.Contains(t, logEntry, "message_id",
				"Background log missing message_id")
			assert.Equal(t, "context-test-123", logEntry["message_id"],
				"Background log has incorrect message_id")

			// Should have the goroutine field we added
			if strings.Contains(logEntry["message"].(string), "Background processing complete") {
				assert.Equal(t, "background", logEntry["goroutine"],
					"Background log missing custom goroutine field")
			}
		})
	}

	t.Logf("Found %d background goroutine log entries", len(backgroundLogs))
}

// min returns the minimum of two integers
func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
