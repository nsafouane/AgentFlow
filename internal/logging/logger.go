// Package logging provides structured logging utilities for AgentFlow
package logging

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
	"time"

	"go.opentelemetry.io/otel/trace"
)

// Standard correlation field names
const (
	TraceIDField    = "trace_id"
	SpanIDField     = "span_id"
	MessageIDField  = "message_id"
	WorkflowIDField = "workflow_id"
	AgentIDField    = "agent_id"
	TimestampField  = "timestamp"
	LevelField      = "level"
	MessageField    = "message"
)

// Reserved field names that cannot be overridden
var ReservedFields = map[string]bool{
	TraceIDField:    true,
	SpanIDField:     true,
	MessageIDField:  true,
	WorkflowIDField: true,
	AgentIDField:    true,
	TimestampField:  true,
	LevelField:      true,
	MessageField:    true,
}

// LogLevel represents the severity level of a log entry
type LogLevel string

const (
	LevelDebug LogLevel = "debug"
	LevelInfo  LogLevel = "info"
	LevelWarn  LogLevel = "warn"
	LevelError LogLevel = "error"
)

// Field represents a structured log field
type Field struct {
	Key   string
	Value interface{}
}

// Logger provides structured logging interface with correlation support
type Logger interface {
	Info(msg string, fields ...Field)
	Error(msg string, err error, fields ...Field)
	Debug(msg string, fields ...Field)
	Warn(msg string, fields ...Field)
	WithMessage(messageID string) Logger
	WithTrace(ctx context.Context) Logger
	WithWorkflow(workflowID string) Logger
	WithAgent(agentID string) Logger
	WithFields(fields ...Field) Logger
}

// CorrelatedLogger implements Logger with automatic correlation field enrichment
type CorrelatedLogger struct {
	baseFields map[string]interface{}
	output     *os.File
	writer     interface{ Write([]byte) (int, error) }
}

// NewLogger creates a new structured logger with JSON output
func NewLogger() Logger {
	return &CorrelatedLogger{
		baseFields: make(map[string]interface{}),
		output:     os.Stdout,
	}
}

// NewLoggerWithOutput creates a logger with custom output destination
func NewLoggerWithOutput(output *os.File) Logger {
	return &CorrelatedLogger{
		baseFields: make(map[string]interface{}),
		output:     output,
	}
}

// NewLoggerWithWriter creates a logger with custom io.Writer destination
func NewLoggerWithWriter(writer interface{ Write([]byte) (int, error) }) Logger {
	return &CorrelatedLogger{
		baseFields: make(map[string]interface{}),
		writer:     writer,
	}
}

// Info logs an info level message
func (l *CorrelatedLogger) Info(msg string, fields ...Field) {
	l.log(LevelInfo, msg, nil, fields...)
}

// Error logs an error level message with optional error
func (l *CorrelatedLogger) Error(msg string, err error, fields ...Field) {
	l.log(LevelError, msg, err, fields...)
}

// Debug logs a debug level message
func (l *CorrelatedLogger) Debug(msg string, fields ...Field) {
	l.log(LevelDebug, msg, nil, fields...)
}

// Warn logs a warning level message
func (l *CorrelatedLogger) Warn(msg string, fields ...Field) {
	l.log(LevelWarn, msg, nil, fields...)
}

// WithMessage returns a logger with message ID correlation
func (l *CorrelatedLogger) WithMessage(messageID string) Logger {
	return l.withField(MessageIDField, messageID)
}

// WithTrace returns a logger with trace context correlation
func (l *CorrelatedLogger) WithTrace(ctx context.Context) Logger {
	span := trace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		newLogger := l.withField(TraceIDField, span.SpanContext().TraceID().String())
		correlatedLogger, ok := newLogger.(*CorrelatedLogger)
		if !ok {
			return newLogger
		}
		return correlatedLogger.withField(SpanIDField, span.SpanContext().SpanID().String())
	}
	return l
}

// WithWorkflow returns a logger with workflow ID correlation
func (l *CorrelatedLogger) WithWorkflow(workflowID string) Logger {
	return l.withField(WorkflowIDField, workflowID)
}

// WithAgent returns a logger with agent ID correlation
func (l *CorrelatedLogger) WithAgent(agentID string) Logger {
	return l.withField(AgentIDField, agentID)
}

// WithFields returns a logger with additional fields
func (l *CorrelatedLogger) WithFields(fields ...Field) Logger {
	newLogger := &CorrelatedLogger{
		baseFields: make(map[string]interface{}),
		output:     l.output,
		writer:     l.writer,
	}

	// Copy existing fields
	for k, v := range l.baseFields {
		newLogger.baseFields[k] = v
	}

	// Add new fields with validation
	for _, field := range fields {
		if err := validateField(field); err != nil {
			// Log validation error but continue
			l.Error("Invalid log field", err, Field{Key: "field_key", Value: field.Key})
			continue
		}
		newLogger.baseFields[field.Key] = field.Value
	}

	return newLogger
}

// withField is a helper to create a new logger with an additional field
func (l *CorrelatedLogger) withField(key string, value interface{}) Logger {
	newLogger := &CorrelatedLogger{
		baseFields: make(map[string]interface{}),
		output:     l.output,
		writer:     l.writer,
	}

	// Copy existing fields
	for k, v := range l.baseFields {
		newLogger.baseFields[k] = v
	}

	// Add new field
	newLogger.baseFields[key] = value

	return newLogger
}

// log is the core logging method that outputs structured JSON
func (l *CorrelatedLogger) log(level LogLevel, msg string, err error, fields ...Field) {
	entry := make(map[string]interface{})

	// Add standard fields
	entry[TimestampField] = time.Now().UTC().Format(time.RFC3339Nano)
	entry[LevelField] = string(level)
	entry[MessageField] = msg

	// Add error if present
	if err != nil {
		entry["error"] = err.Error()
	}

	// Add base correlation fields
	for k, v := range l.baseFields {
		entry[k] = v
	}

	// Add additional fields with validation
	for _, field := range fields {
		if err := validateField(field); err != nil {
			// Skip invalid fields but don't fail the log entry
			continue
		}
		entry[field.Key] = field.Value
	}

	// Serialize to JSON with consistent field ordering
	jsonBytes, err := marshalOrderedJSON(entry)
	if err != nil {
		// Fallback to basic logging if JSON marshaling fails
		fmt.Fprintf(l.output, "LOG_ERROR: Failed to marshal log entry: %v\n", err)
		return
	}

	// Write to output
	if l.writer != nil {
		l.writer.Write(append(jsonBytes, '\n'))
	} else {
		fmt.Fprintln(l.output, string(jsonBytes))
	}
}

// validateField validates that a field doesn't use reserved names
func validateField(field Field) error {
	if ReservedFields[field.Key] {
		return fmt.Errorf("field key '%s' is reserved and cannot be used", field.Key)
	}

	// Additional validation rules can be added here
	if strings.TrimSpace(field.Key) == "" {
		return fmt.Errorf("field key cannot be empty or whitespace")
	}

	return nil
}

// marshalOrderedJSON marshals a map to JSON with consistent field ordering
func marshalOrderedJSON(data map[string]interface{}) ([]byte, error) {
	// Create ordered map for consistent output
	orderedMap := make(map[string]interface{})

	// Define field order priority (standard fields first)
	fieldOrder := []string{
		TimestampField,
		LevelField,
		MessageField,
		TraceIDField,
		SpanIDField,
		MessageIDField,
		WorkflowIDField,
		AgentIDField,
	}

	// Add standard fields in order
	for _, key := range fieldOrder {
		if value, exists := data[key]; exists {
			orderedMap[key] = value
		}
	}

	// Add remaining fields in alphabetical order
	var remainingKeys []string
	for key := range data {
		found := false
		for _, standardKey := range fieldOrder {
			if key == standardKey {
				found = true
				break
			}
		}
		if !found {
			remainingKeys = append(remainingKeys, key)
		}
	}

	sort.Strings(remainingKeys)
	for _, key := range remainingKeys {
		orderedMap[key] = data[key]
	}

	return json.Marshal(orderedMap)
}

// Helper functions for creating fields
func String(key, value string) Field {
	return Field{Key: key, Value: value}
}

func Int(key string, value int) Field {
	return Field{Key: key, Value: value}
}

func Float64(key string, value float64) Field {
	return Field{Key: key, Value: value}
}

func Bool(key string, value bool) Field {
	return Field{Key: key, Value: value}
}

func Any(key string, value interface{}) Field {
	return Field{Key: key, Value: value}
}
