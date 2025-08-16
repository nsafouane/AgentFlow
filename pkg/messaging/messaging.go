// Package messaging provides message bus abstractions for AgentFlow
package messaging

import "context"

// MessageBus provides messaging interface for AgentFlow
type MessageBus interface {
	Publish(ctx context.Context, subject string, data []byte) error
	Subscribe(ctx context.Context, subject string, handler MessageHandler) error
	Close() error
}

// MessageHandler handles incoming messages
type MessageHandler func(msg Message) error

// Message represents a message in the system
type Message struct {
	Subject   string
	Data      []byte
	Headers   map[string]string
	Timestamp int64
}

// Publisher provides message publishing interface
type Publisher interface {
	Publish(ctx context.Context, subject string, data []byte) error
}

// Subscriber provides message subscription interface
type Subscriber interface {
	Subscribe(ctx context.Context, subject string, handler MessageHandler) error
}
