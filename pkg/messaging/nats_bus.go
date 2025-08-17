package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"strings"
	"time"

	"github.com/agentflow/agentflow/internal/logging"
	"github.com/nats-io/nats.go"
)

// Stream configuration constants
const (
	StreamAFMessages = "AF_MESSAGES"
	StreamAFTools    = "AF_TOOLS"
	StreamAFSystem   = "AF_SYSTEM"
)

// natsBus implements MessageBus using NATS JetStream
type natsBus struct {
	conn       *nats.Conn
	js         nats.JetStreamContext
	config     *BusConfig
	serializer *CanonicalSerializer
	tracing    *TracingMiddleware
	logger     logging.Logger
}

// NewNATSBus creates a new NATS JetStream message bus
func NewNATSBus(config *BusConfig) (MessageBus, error) {
	if config == nil {
		config = DefaultBusConfig()
	}

	// Apply environment variable overrides
	if err := applyEnvConfig(config); err != nil {
		return nil, fmt.Errorf("failed to apply environment config: %w", err)
	}

	// Create NATS connection with retry policy
	conn, err := connectWithRetry(config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to NATS: %w", err)
	}

	// Create JetStream context
	js, err := conn.JetStream()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create JetStream context: %w", err)
	}

	// Initialize serializer
	serializer, err := NewCanonicalSerializer()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create serializer: %w", err)
	}

	// Initialize tracing middleware
	tracing, err := NewTracingMiddleware(DefaultTracingConfig())
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create tracing middleware: %w", err)
	}

	bus := &natsBus{
		conn:       conn,
		js:         js,
		config:     config,
		serializer: serializer,
		tracing:    tracing,
		logger:     logging.NewLogger(),
	}

	// Initialize streams
	if err := bus.initializeStreams(); err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to initialize streams: %w", err)
	}

	return bus, nil
}

// connectWithRetry establishes NATS connection with exponential backoff
func connectWithRetry(config *BusConfig) (*nats.Conn, error) {
	opts := []nats.Option{
		nats.MaxReconnects(config.MaxReconnect),
		nats.ReconnectWait(config.ReconnectWait),
		nats.Timeout(config.ConnectTimeout),
		nats.DisconnectErrHandler(func(nc *nats.Conn, err error) {
			if err != nil {
				logger := logging.NewLogger()
				logger.Error("NATS disconnected", err, logging.String("url", nc.ConnectedUrl()))
			}
		}),
		nats.ReconnectHandler(func(nc *nats.Conn) {
			logger := logging.NewLogger()
			logger.Info("NATS reconnected", logging.String("url", nc.ConnectedUrl()))
		}),
	}

	var conn *nats.Conn
	var err error

	// Retry with exponential backoff
	for attempt := 0; attempt < config.MaxReconnect; attempt++ {
		conn, err = nats.Connect(config.URL, opts...)
		if err == nil {
			return conn, nil
		}

		if attempt < config.MaxReconnect-1 {
			backoff := time.Duration(math.Pow(2, float64(attempt))) * config.ReconnectWait
			// Add jitter (±25%)
			jitter := time.Duration(rand.Float64() * 0.5 * float64(backoff))
			if rand.Float64() < 0.5 {
				backoff -= jitter
			} else {
				backoff += jitter
			}

			logger := logging.NewLogger()
			logger.Warn("NATS connection attempt failed, retrying",
				logging.Int("attempt", attempt+1),
				logging.String("backoff", backoff.String()),
				logging.String("error", err.Error()))
			time.Sleep(backoff)
		}
	}

	return nil, fmt.Errorf("failed to connect after %d attempts: %w", config.MaxReconnect, err)
}

// initializeStreams creates the required JetStream streams
func (nb *natsBus) initializeStreams() error {
	streams := []struct {
		name     string
		subjects []string
		maxAge   time.Duration
		maxBytes int64
		replicas int
	}{
		{
			name:     StreamAFMessages,
			subjects: []string{"workflows.*.*", "agents.*.*"},
			maxAge:   168 * time.Hour,         // 7 days
			maxBytes: 10 * 1024 * 1024 * 1024, // 10GB
			replicas: 1,                       // Default to 1 for development, can be overridden
		},
		{
			name:     StreamAFTools,
			subjects: []string{"tools.*"},
			maxAge:   720 * time.Hour,        // 30 days
			maxBytes: 5 * 1024 * 1024 * 1024, // 5GB
			replicas: 1,
		},
		{
			name:     StreamAFSystem,
			subjects: []string{"system.*"},
			maxAge:   24 * time.Hour,     // 1 day
			maxBytes: 1024 * 1024 * 1024, // 1GB
			replicas: 1,
		},
	}

	for _, stream := range streams {
		cfg := &nats.StreamConfig{
			Name:      stream.name,
			Subjects:  stream.subjects,
			Storage:   nats.FileStorage,
			MaxAge:    stream.maxAge,
			MaxBytes:  stream.maxBytes,
			Replicas:  stream.replicas,
			Retention: nats.LimitsPolicy,
		}

		// Try to create or update the stream
		_, err := nb.js.AddStream(cfg)
		if err != nil {
			// If stream exists, try to update it
			if strings.Contains(err.Error(), "stream name already in use") {
				_, err = nb.js.UpdateStream(cfg)
				if err != nil {
					return fmt.Errorf("failed to update stream %s: %w", stream.name, err)
				}
			} else {
				return fmt.Errorf("failed to create stream %s: %w", stream.name, err)
			}
		}
	}

	return nil
}

// Publish publishes a message to the specified subject
func (nb *natsBus) Publish(ctx context.Context, subject string, msg *Message) error {
	// Start publish span
	ctx, span := nb.tracing.StartPublishSpan(ctx, subject, msg)
	defer span.End()

	// Create logger with trace and message context
	logger := nb.logger.WithTrace(ctx).WithMessage(msg.ID)

	// Inject trace context into message
	nb.tracing.InjectTraceContext(ctx, msg)

	logger.Debug("Publishing message",
		logging.String("subject", subject),
		logging.String("message_type", string(msg.Type)),
		logging.String("from", msg.From),
		logging.String("to", msg.To))

	// Serialize the message
	data, err := nb.serializer.Serialize(msg)
	if err != nil {
		span.RecordError(err)
		logger.Error("Failed to serialize message", err)
		return fmt.Errorf("failed to serialize message: %w", err)
	}

	// Publish with context
	_, err = nb.js.PublishAsync(subject, data)
	if err != nil {
		span.RecordError(err)
		logger.Error("Failed to publish message", err, logging.String("subject", subject))
		return fmt.Errorf("failed to publish message to subject %s: %w", subject, err)
	}

	logger.Info("Message published successfully",
		logging.String("subject", subject),
		logging.Int("payload_size", len(data)))

	return nil
}

// Subscribe creates a subscription to the specified subject
func (nb *natsBus) Subscribe(ctx context.Context, subject string, handler MessageHandler) (*Subscription, error) {
	// Determine the appropriate stream based on subject
	streamName := nb.getStreamForSubject(subject)
	if streamName == "" {
		return nil, fmt.Errorf("no stream found for subject: %s", subject)
	}

	// Create a unique consumer name (NATS consumer names must be valid identifiers)
	cleanSubject := strings.ReplaceAll(strings.ReplaceAll(subject, "*", "wildcard"), ".", "_")
	consumerName := fmt.Sprintf("consumer_%s_%d", cleanSubject, time.Now().UnixNano())

	// Configure the consumer
	consumerConfig := &nats.ConsumerConfig{
		Durable:       consumerName,
		DeliverPolicy: nats.DeliverAllPolicy,
		AckPolicy:     nats.AckExplicitPolicy,
		AckWait:       nb.config.AckWait,
		MaxAckPending: nb.config.MaxInFlight,
		FilterSubject: subject,
		ReplayPolicy:  nats.ReplayInstantPolicy,
	}

	// Create the consumer
	_, err := nb.js.AddConsumer(streamName, consumerConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create consumer: %w", err)
	}

	// Create the subscription
	sub, err := nb.js.PullSubscribe(subject, consumerName)
	if err != nil {
		return nil, fmt.Errorf("failed to create subscription: %w", err)
	}

	subscription := &Subscription{
		Subject:  subject,
		Consumer: consumerName,
		IsActive: true,
		unsubscribe: func() error {
			return sub.Unsubscribe()
		},
	}

	// Start message processing goroutine
	go nb.processMessages(ctx, sub, handler, subscription)

	return subscription, nil
}

// processMessages handles incoming messages for a subscription
func (nb *natsBus) processMessages(ctx context.Context, sub *nats.Subscription, handler MessageHandler, subscription *Subscription) {
	baseLogger := nb.logger.WithFields(
		logging.String("subject", subscription.Subject),
		logging.String("consumer", subscription.Consumer))

	baseLogger.Info("Starting message processing")

	for subscription.IsActive {
		select {
		case <-ctx.Done():
			baseLogger.Info("Message processing stopped due to context cancellation")
			return
		default:
			// Fetch messages with timeout
			msgs, err := sub.Fetch(1, nats.MaxWait(1*time.Second))
			if err != nil {
				if err == nats.ErrTimeout {
					continue // Normal timeout, keep polling
				}
				baseLogger.Error("Error fetching messages", err)
				continue
			}

			for _, natsMsg := range msgs {
				// Deserialize the message
				var msg Message
				if err := json.Unmarshal(natsMsg.Data, &msg); err != nil {
					baseLogger.Error("Error deserializing message", err,
						logging.Int("data_size", len(natsMsg.Data)))
					natsMsg.Nak()
					continue
				}

				// Create message-specific logger
				msgLogger := baseLogger.WithMessage(msg.ID).WithFields(
					logging.String("message_type", string(msg.Type)),
					logging.String("from", msg.From),
					logging.String("to", msg.To))

				msgLogger.Debug("Processing message")

				// Verify message hash
				if err := nb.serializer.ValidateHash(&msg); err != nil {
					msgLogger.Error("Message hash validation failed", err)
					natsMsg.Nak()
					continue
				}

				// Extract trace context and start consume span
				traceCtx := nb.tracing.ExtractTraceContext(&msg)
				traceCtx, span := nb.tracing.StartConsumeSpan(traceCtx, subscription.Subject, &msg)

				// Create trace-aware logger
				traceLogger := msgLogger.WithTrace(traceCtx)

				// Handle the message
				if err := handler(traceCtx, &msg); err != nil {
					span.RecordError(err)
					span.End()
					traceLogger.Error("Message handler error", err)
					natsMsg.Nak()
					continue
				}

				span.End()
				traceLogger.Info("Message processed successfully")

				// Acknowledge successful processing
				natsMsg.Ack()
			}
		}
	}
}

// Replay retrieves messages for a workflow in chronological order
func (nb *natsBus) Replay(ctx context.Context, workflowID string, from time.Time) ([]Message, error) {
	// Start replay span
	ctx, span := nb.tracing.StartReplaySpan(ctx, workflowID)
	defer span.End()

	// Create logger with trace and workflow context
	logger := nb.logger.WithTrace(ctx).WithWorkflow(workflowID)

	logger.Info("Starting message replay",
		logging.String("from_time", from.Format(time.RFC3339)))

	// Build subject pattern for the workflow
	subjectPattern := fmt.Sprintf("workflows.%s.*", workflowID)

	// Create a temporary consumer for replay
	consumerName := fmt.Sprintf("replay_%s_%d", workflowID, time.Now().UnixNano())

	consumerConfig := &nats.ConsumerConfig{
		DeliverPolicy: nats.DeliverByStartTimePolicy,
		OptStartTime:  &from,
		AckPolicy:     nats.AckExplicitPolicy,
		FilterSubject: subjectPattern,
		ReplayPolicy:  nats.ReplayInstantPolicy,
	}

	// Create temporary consumer
	_, err := nb.js.AddConsumer(StreamAFMessages, consumerConfig)
	if err != nil {
		span.RecordError(err)
		logger.Error("Failed to create replay consumer", err,
			logging.String("consumer_name", consumerName))
		return nil, fmt.Errorf("failed to create replay consumer: %w", err)
	}

	// Clean up consumer when done
	defer func() {
		if err := nb.js.DeleteConsumer(StreamAFMessages, consumerName); err != nil {
			logger.Warn("Failed to cleanup replay consumer",
				logging.String("consumer_name", consumerName),
				logging.String("error", err.Error()))
		}
	}()

	// Create subscription for replay
	sub, err := nb.js.PullSubscribe(subjectPattern, consumerName)
	if err != nil {
		span.RecordError(err)
		logger.Error("Failed to create replay subscription", err)
		return nil, fmt.Errorf("failed to create replay subscription: %w", err)
	}
	defer sub.Unsubscribe()

	var messages []Message

	// Fetch all available messages
	for {
		msgs, err := sub.Fetch(100, nats.MaxWait(1*time.Second))
		if err != nil {
			if err == nats.ErrTimeout {
				break // No more messages
			}
			span.RecordError(err)
			logger.Error("Failed to fetch replay messages", err)
			return nil, fmt.Errorf("failed to fetch replay messages: %w", err)
		}

		if len(msgs) == 0 {
			break
		}

		logger.Debug("Fetched replay messages batch", logging.Int("count", len(msgs)))

		for _, natsMsg := range msgs {
			var msg Message
			if err := json.Unmarshal(natsMsg.Data, &msg); err != nil {
				logger.Error("Error deserializing replay message", err,
					logging.Int("data_size", len(natsMsg.Data)))
				continue
			}

			// Verify message hash
			if err := nb.serializer.ValidateHash(&msg); err != nil {
				logger.Error("Replay message hash validation failed", err,
					logging.String("message_id", msg.ID))
				continue
			}

			messages = append(messages, msg)
			natsMsg.Ack()
		}
	}

	// Sort messages by timestamp to ensure chronological order
	sort.Slice(messages, func(i, j int) bool {
		return messages[i].Timestamp.Before(messages[j].Timestamp)
	})

	logger.Info("Message replay completed",
		logging.Int("message_count", len(messages)),
		logging.String("time_range", fmt.Sprintf("%s to %s",
			from.Format(time.RFC3339),
			time.Now().Format(time.RFC3339))))

	return messages, nil
}

// Close closes the NATS connection
func (nb *natsBus) Close() error {
	if nb.conn != nil {
		nb.conn.Close()
	}
	return nil
}

// getStreamForSubject determines which stream a subject belongs to
func (nb *natsBus) getStreamForSubject(subject string) string {
	if strings.HasPrefix(subject, "workflows.") || strings.HasPrefix(subject, "agents.") {
		return StreamAFMessages
	}
	if strings.HasPrefix(subject, "tools.") {
		return StreamAFTools
	}
	if strings.HasPrefix(subject, "system.") {
		return StreamAFSystem
	}
	return ""
}

// applyEnvConfig applies environment variable configuration
func applyEnvConfig(config *BusConfig) error {
	// This is a simplified version - in a real implementation,
	// you would use a proper environment variable parsing library
	// For now, we'll use the defaults
	return nil
}
