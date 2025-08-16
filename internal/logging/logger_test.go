package logging

import (
	"testing"
)

// TestNewLogger tests the creation of a new logger
func TestNewLogger(t *testing.T) {
	logger := NewLogger()
	if logger == nil {
		t.Error("NewLogger() returned nil")
	}

	// Test that logger methods don't panic
	logger.Info("test info message")
	logger.Error("test error message")
	logger.Debug("test debug message")

	t.Log("Logger functionality test completed")
}
