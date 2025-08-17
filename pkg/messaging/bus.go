package messaging

import (
	"context"
	"time"
)

// MessageBus defines the interface for message bus operations
type MessageBus interface {
	// Publish publishes a message to the specified subject
	Publish(ctx context.Context, subject string, msg *Message) error

	// Subscribe creates a subscription to the specified subject with a message handler
	Subscribe(ctx context.Context, subject string, handler MessageHandler) (*Subscription, error)

	// Replay retrieves messages for a workflow in chronological order
	Replay(ctx context.Context, workflowID string, from time.Time) ([]Message, error)

	// Close closes the message bus connection
	Close() error
}

// MessageHandler defines the function signature for message handlers
type MessageHandler func(ctx context.Context, msg *Message) error

// Subscription represents an active subscription
type Subscription struct {
	Subject     string
	Consumer    string
	IsActive    bool
	unsubscribe func() error
}

// Unsubscribe cancels the subscription
func (s *Subscription) Unsubscribe() error {
	if s.unsubscribe != nil {
		s.IsActive = false
		return s.unsubscribe()
	}
	return nil
}

// BusConfig holds configuration for the message bus
type BusConfig struct {
	URL            string        `env:"AF_BUS_URL"`
	MaxReconnect   int           `env:"AF_BUS_MAX_RECONNECT"`
	ReconnectWait  time.Duration `env:"AF_BUS_RECONNECT_WAIT"`
	AckWait        time.Duration `env:"AF_BUS_ACK_WAIT"`
	MaxInFlight    int           `env:"AF_BUS_MAX_IN_FLIGHT"`
	ConnectTimeout time.Duration `env:"AF_BUS_CONNECT_TIMEOUT"`
	RequestTimeout time.Duration `env:"AF_BUS_REQUEST_TIMEOUT"`
}

// DefaultBusConfig returns default configuration values
func DefaultBusConfig() *BusConfig {
	return &BusConfig{
		URL:            "nats://localhost:4222",
		MaxReconnect:   10,
		ReconnectWait:  2 * time.Second,
		AckWait:        30 * time.Second,
		MaxInFlight:    1000,
		ConnectTimeout: 5 * time.Second,
		RequestTimeout: 10 * time.Second,
	}
}
