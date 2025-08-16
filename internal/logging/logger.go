// Package logging provides structured logging utilities for AgentFlow
package logging

import "log"

// Logger provides structured logging interface
type Logger interface {
	Info(msg string, fields ...interface{})
	Error(msg string, fields ...interface{})
	Debug(msg string, fields ...interface{})
}

// NewLogger creates a new structured logger
func NewLogger() Logger {
	// Structured logging implementation will be added
	return &defaultLogger{}
}

type defaultLogger struct{}

func (l *defaultLogger) Info(msg string, fields ...interface{}) {
	log.Printf("INFO: %s", msg)
}

func (l *defaultLogger) Error(msg string, fields ...interface{}) {
	log.Printf("ERROR: %s", msg)
}

func (l *defaultLogger) Debug(msg string, fields ...interface{}) {
	log.Printf("DEBUG: %s", msg)
}
