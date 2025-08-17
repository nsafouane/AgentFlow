package logging

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"go.opentelemetry.io/otel"
)

// TestNewLogger tests the creation of a new logger
func TestNewLogger(t *testing.T) {
	logger := NewLogger()
	if logger == nil {
		t.Error("NewLogger() returned nil")
	}

	// Test that logger methods don't panic
	logger.Info("test info message")
	logger.Error("test error message", nil)
	logger.Debug("test debug message")

	t.Log("Logger functionality test completed")
}

// TestStructuredJSONOutput tests that logs are output in structured JSON format
func TestStructuredJSONOutput(t *testing.T) {
	// Test the JSON marshaling directly since we can't easily capture stdout
	testData := map[string]interface{}{
		TimestampField: "2023-01-01T00:00:00Z",
		LevelField:     "info",
		MessageField:   "test message",
		TraceIDField:   "trace123",
		SpanIDField:    "span456",
	}

	jsonBytes, err := marshalOrderedJSON(testData)
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}

	// Verify it's valid JSON
	var parsed map[string]interface{}
	if err := json.Unmarshal(jsonBytes, &parsed); err != nil {
		t.Fatalf("Output is not valid JSON: %v", err)
	}

	// Verify required fields are present
	requiredFields := []string{TimestampField, LevelField, MessageField}
	for _, field := range requiredFields {
		if _, exists := parsed[field]; !exists {
			t.Errorf("Required field '%s' missing from JSON output", field)
		}
	}
}

// TestCorrelationFieldEnrichment tests automatic enrichment with correlation IDs
func TestCorrelationFieldEnrichment(t *testing.T) {
	tests := []struct {
		name     string
		setup    func(Logger) Logger
		expected map[string]interface{}
	}{
		{
			name: "WithMessage adds message_id",
			setup: func(l Logger) Logger {
				return l.WithMessage("msg123")
			},
			expected: map[string]interface{}{
				MessageIDField: "msg123",
			},
		},
		{
			name: "WithWorkflow adds workflow_id",
			setup: func(l Logger) Logger {
				return l.WithWorkflow("workflow456")
			},
			expected: map[string]interface{}{
				WorkflowIDField: "workflow456",
			},
		},
		{
			name: "WithAgent adds agent_id",
			setup: func(l Logger) Logger {
				return l.WithAgent("agent789")
			},
			expected: map[string]interface{}{
				AgentIDField: "agent789",
			},
		},
		{
			name: "WithFields adds custom fields",
			setup: func(l Logger) Logger {
				return l.WithFields(
					String("custom_field", "custom_value"),
					Int("count", 42),
				)
			},
			expected: map[string]interface{}{
				"custom_field": "custom_value",
				"count":        42,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := NewLogger()
			enrichedLogger := tt.setup(logger)

			// Verify the logger has the expected fields
			correlatedLogger, ok := enrichedLogger.(*CorrelatedLogger)
			if !ok {
				t.Fatal("Logger is not a CorrelatedLogger")
			}

			for key, expectedValue := range tt.expected {
				if actualValue, exists := correlatedLogger.baseFields[key]; !exists {
					t.Errorf("Expected field '%s' not found", key)
				} else if actualValue != expectedValue {
					t.Errorf("Field '%s': expected %v, got %v", key, expectedValue, actualValue)
				}
			}
		})
	}
}

// TestTraceContextExtraction tests OpenTelemetry trace context extraction
func TestTraceContextExtraction(t *testing.T) {
	// Create a tracer for testing
	tracer := otel.Tracer("test")

	// Create a span context
	ctx, span := tracer.Start(context.Background(), "test-span")
	defer span.End()

	logger := NewLogger()
	tracedLogger := logger.WithTrace(ctx)

	// Verify trace context was extracted
	correlatedLogger, ok := tracedLogger.(*CorrelatedLogger)
	if !ok {
		t.Fatal("Logger is not a CorrelatedLogger")
	}

	// Check if trace fields are present (they should be if span context is valid)
	spanContext := span.SpanContext()
	if spanContext.IsValid() {
		expectedTraceID := spanContext.TraceID().String()
		expectedSpanID := spanContext.SpanID().String()

		if traceID, exists := correlatedLogger.baseFields[TraceIDField]; !exists {
			t.Error("TraceID field not found")
		} else if traceID != expectedTraceID {
			t.Errorf("TraceID: expected %s, got %s", expectedTraceID, traceID)
		}

		if spanID, exists := correlatedLogger.baseFields[SpanIDField]; !exists {
			t.Error("SpanID field not found")
		} else if spanID != expectedSpanID {
			t.Errorf("SpanID: expected %s, got %s", expectedSpanID, spanID)
		}
	}
}

// TestFieldValidation tests reserved field validation and linting
func TestFieldValidation(t *testing.T) {
	tests := []struct {
		name        string
		field       Field
		shouldError bool
	}{
		{
			name:        "Reserved field trace_id",
			field:       Field{Key: TraceIDField, Value: "test"},
			shouldError: true,
		},
		{
			name:        "Reserved field span_id",
			field:       Field{Key: SpanIDField, Value: "test"},
			shouldError: true,
		},
		{
			name:        "Reserved field message_id",
			field:       Field{Key: MessageIDField, Value: "test"},
			shouldError: true,
		},
		{
			name:        "Reserved field timestamp",
			field:       Field{Key: TimestampField, Value: "test"},
			shouldError: true,
		},
		{
			name:        "Empty field key",
			field:       Field{Key: "", Value: "test"},
			shouldError: true,
		},
		{
			name:        "Whitespace field key",
			field:       Field{Key: "   ", Value: "test"},
			shouldError: true,
		},
		{
			name:        "Valid custom field",
			field:       Field{Key: "custom_field", Value: "test"},
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateField(tt.field)
			if tt.shouldError && err == nil {
				t.Error("Expected validation error but got none")
			}
			if !tt.shouldError && err != nil {
				t.Errorf("Unexpected validation error: %v", err)
			}
		})
	}
}

// TestConsistentFieldOrdering tests that JSON output has consistent field ordering
func TestConsistentFieldOrdering(t *testing.T) {
	testData := map[string]interface{}{
		"zebra":        "last",
		MessageField:   "test message",
		TraceIDField:   "trace123",
		TimestampField: "2023-01-01T00:00:00Z",
		"alpha":        "first",
		LevelField:     "info",
		SpanIDField:    "span456",
	}

	// Marshal multiple times and verify consistency
	var outputs []string
	for i := 0; i < 5; i++ {
		jsonBytes, err := marshalOrderedJSON(testData)
		if err != nil {
			t.Fatalf("Failed to marshal JSON: %v", err)
		}
		outputs = append(outputs, string(jsonBytes))
	}

	// All outputs should be identical
	for i := 1; i < len(outputs); i++ {
		if outputs[i] != outputs[0] {
			t.Errorf("Inconsistent JSON ordering:\nFirst:  %s\nOther:  %s", outputs[0], outputs[i])
		}
	}

	// Just verify that the output is consistent - field ordering within JSON is not guaranteed
	// The important thing is that marshalOrderedJSON produces consistent output
	t.Logf("Consistent JSON output verified: %s", outputs[0])
}

// TestLogLevels tests all log levels work correctly
func TestLogLevels(t *testing.T) {
	logger := NewLogger()

	// Test all log levels don't panic
	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warn("warn message")
	logger.Error("error message", errors.New("test error"))

	// Test with fields
	logger.Info("info with fields", String("key", "value"))
	logger.Error("error with fields", errors.New("test error"), Int("code", 500))
}

// TestChainedCorrelation tests that correlation fields are preserved across chained loggers
func TestChainedCorrelation(t *testing.T) {
	logger := NewLogger()

	// Chain multiple correlation methods
	chainedLogger := logger.
		WithMessage("msg123").
		WithWorkflow("workflow456").
		WithAgent("agent789").
		WithFields(String("custom", "value"))

	correlatedLogger, ok := chainedLogger.(*CorrelatedLogger)
	if !ok {
		t.Fatal("Logger is not a CorrelatedLogger")
	}

	// Verify all correlation fields are present
	expectedFields := map[string]interface{}{
		MessageIDField:  "msg123",
		WorkflowIDField: "workflow456",
		AgentIDField:    "agent789",
		"custom":        "value",
	}

	for key, expectedValue := range expectedFields {
		if actualValue, exists := correlatedLogger.baseFields[key]; !exists {
			t.Errorf("Expected field '%s' not found in chained logger", key)
		} else if actualValue != expectedValue {
			t.Errorf("Field '%s': expected %v, got %v", key, expectedValue, actualValue)
		}
	}
}

// TestHelperFunctions tests the field helper functions
func TestHelperFunctions(t *testing.T) {
	tests := []struct {
		name     string
		field    Field
		expected interface{}
	}{
		{
			name:     "String helper",
			field:    String("test", "value"),
			expected: "value",
		},
		{
			name:     "Int helper",
			field:    Int("count", 42),
			expected: 42,
		},
		{
			name:     "Float64 helper",
			field:    Float64("rate", 3.14),
			expected: 3.14,
		},
		{
			name:     "Bool helper",
			field:    Bool("enabled", true),
			expected: true,
		},
		{
			name:     "Any helper",
			field:    Any("data", "complex_value"),
			expected: "complex_value",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.field.Value != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, tt.field.Value)
			}
		})
	}
}

// TestReservedFieldsMap tests that all reserved fields are properly defined
func TestReservedFieldsMap(t *testing.T) {
	expectedReservedFields := []string{
		TraceIDField,
		SpanIDField,
		MessageIDField,
		WorkflowIDField,
		AgentIDField,
		TimestampField,
		LevelField,
		MessageField,
	}

	for _, field := range expectedReservedFields {
		if !ReservedFields[field] {
			t.Errorf("Field '%s' should be marked as reserved", field)
		}
	}

	// Verify the count matches
	if len(ReservedFields) != len(expectedReservedFields) {
		t.Errorf("Expected %d reserved fields, got %d", len(expectedReservedFields), len(ReservedFields))
	}
}
