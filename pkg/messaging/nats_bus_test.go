package messaging

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
)

// TestNATSContainer manages a NATS server container for testing
type TestNATSContainer struct {
	container testcontainers.Container
	URL       string
}

// StartNATSContainer starts a NATS server container for testing
func StartNATSContainer(ctx context.Context) (*TestNATSContainer, error) {
	req := testcontainers.ContainerRequest{
		Image:        "nats:2.10-alpine",
		ExposedPorts: []string{"4222/tcp"},
		Cmd:          []string{"-js", "-m", "8222"},
		WaitingFor:   wait.ForLog("Server is ready").WithStartupTimeout(30 * time.Second),
	}

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to start NATS container: %w", err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		container.Terminate(ctx)
		return nil, fmt.Errorf("failed to get container host: %w", err)
	}

	port, err := container.MappedPort(ctx, "4222")
	if err != nil {
		container.Terminate(ctx)
		return nil, fmt.Errorf("failed to get container port: %w", err)
	}

	return &TestNATSContainer{
		container: container,
		URL:       fmt.Sprintf("nats://%s:%s", host, port.Port()),
	}, nil
}

// Stop stops and removes the NATS container
func (tnc *TestNATSContainer) Stop(ctx context.Context) error {
	return tnc.container.Terminate(ctx)
}

func TestNATSBus_NewNATSBus(t *testing.T) {
	ctx := context.Background()

	// Start NATS container
	natsContainer, err := StartNATSContainer(ctx)
	require.NoError(t, err)
	defer natsContainer.Stop(ctx)

	// Test successful connection
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
	require.NotNil(t, bus)
	defer bus.Close()

	// Verify streams were created
	natsBus := bus.(*natsBus)
	streams := []string{StreamAFMessages, StreamAFTools, StreamAFSystem}

	for _, streamName := range streams {
		info, err := natsBus.js.StreamInfo(streamName)
		require.NoError(t, err)
		assert.Equal(t, streamName, info.Config.Name)
	}
}

func TestNATSBus_PublishSubscribe(t *testing.T) {
	ctx := context.Background()

	// Start NATS container
	natsContainer, err := StartNATSContainer(ctx)
	require.NoError(t, err)
	defer natsContainer.Stop(ctx)

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

	// Create test message
	msg := NewMessage("test-id", "agent-1", "agent-2", MessageTypeRequest)
	msg.SetPayload(map[string]interface{}{"test": "data"})
	msg.SetTraceContext("trace-123", "span-456")

	// Set envelope hash properly
	natsBus := bus.(*natsBus)
	err = natsBus.serializer.SetEnvelopeHash(msg)
	require.NoError(t, err)

	// Set up subscription
	subject := "agents.agent-2.in"
	receivedMessages := make(chan *Message, 1)

	handler := func(ctx context.Context, receivedMsg *Message) error {
		receivedMessages <- receivedMsg
		return nil
	}

	sub, err := bus.Subscribe(ctx, subject, handler)
	require.NoError(t, err)
	require.NotNil(t, sub)
	defer sub.Unsubscribe()

	// Publish message
	err = bus.Publish(ctx, subject, msg)
	require.NoError(t, err)

	// Wait for message to be received
	select {
	case receivedMsg := <-receivedMessages:
		assert.Equal(t, msg.ID, receivedMsg.ID)
		assert.Equal(t, msg.From, receivedMsg.From)
		assert.Equal(t, msg.To, receivedMsg.To)
		assert.Equal(t, msg.Type, receivedMsg.Type)
		assert.Equal(t, msg.TraceID, receivedMsg.TraceID)
		assert.Equal(t, msg.SpanID, receivedMsg.SpanID)
		assert.Equal(t, msg.EnvelopeHash, receivedMsg.EnvelopeHash)
	case <-time.After(5 * time.Second):
		t.Fatal("Message not received within timeout")
	}
}

func TestNATSBus_MessageOrdering(t *testing.T) {
	ctx := context.Background()

	// Start NATS container
	natsContainer, err := StartNATSContainer(ctx)
	require.NoError(t, err)
	defer natsContainer.Stop(ctx)

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

	natsBus := bus.(*natsBus)
	subject := "agents.test-agent.in"

	// Publish multiple messages with different timestamps
	messages := make([]*Message, 5)
	baseTime := time.Now().UTC()

	for i := 0; i < 5; i++ {
		msg := NewMessage(fmt.Sprintf("msg-%d", i), "sender", "test-agent", MessageTypeEvent)
		msg.Timestamp = baseTime.Add(time.Duration(i) * time.Second)
		msg.SetPayload(map[string]interface{}{"sequence": i})

		// Set envelope hash properly
		err = natsBus.serializer.SetEnvelopeHash(msg)
		require.NoError(t, err)

		messages[i] = msg

		// Publish with small delay to ensure ordering
		err = bus.Publish(ctx, subject, msg)
		require.NoError(t, err)
		time.Sleep(10 * time.Millisecond)
	}

	// Wait a bit for messages to be stored
	time.Sleep(100 * time.Millisecond)

	// Set up subscription to collect messages
	receivedMessages := make([]*Message, 0, 5)
	receivedChan := make(chan *Message, 5)

	handler := func(ctx context.Context, receivedMsg *Message) error {
		receivedChan <- receivedMsg
		return nil
	}

	sub, err := bus.Subscribe(ctx, subject, handler)
	require.NoError(t, err)
	defer sub.Unsubscribe()

	// Collect all messages
	timeout := time.After(5 * time.Second)
	for len(receivedMessages) < 5 {
		select {
		case msg := <-receivedChan:
			receivedMessages = append(receivedMessages, msg)
		case <-timeout:
			t.Fatalf("Only received %d out of 5 messages", len(receivedMessages))
		}
	}

	// Verify messages are in chronological order
	for i := 1; i < len(receivedMessages); i++ {
		assert.True(t, receivedMessages[i-1].Timestamp.Before(receivedMessages[i].Timestamp) ||
			receivedMessages[i-1].Timestamp.Equal(receivedMessages[i].Timestamp),
			"Messages not in chronological order: %v >= %v",
			receivedMessages[i-1].Timestamp, receivedMessages[i].Timestamp)
	}
}

func TestNATSBus_Replay(t *testing.T) {
	ctx := context.Background()

	// Start NATS container
	natsContainer, err := StartNATSContainer(ctx)
	require.NoError(t, err)
	defer natsContainer.Stop(ctx)

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

	natsBus := bus.(*natsBus)
	workflowID := "test-workflow"

	// Publish messages to workflow subjects
	baseTime := time.Now().UTC()
	expectedMessages := 3

	for i := 0; i < expectedMessages; i++ {
		msg := NewMessage(fmt.Sprintf("workflow-msg-%d", i), "sender", workflowID, MessageTypeEvent)
		msg.Timestamp = baseTime.Add(time.Duration(i) * time.Second)
		msg.SetPayload(map[string]interface{}{"step": i})

		// Set envelope hash properly
		err = natsBus.serializer.SetEnvelopeHash(msg)
		require.NoError(t, err)

		subject := fmt.Sprintf("workflows.%s.in", workflowID)
		err = bus.Publish(ctx, subject, msg)
		require.NoError(t, err)
	}

	// Wait for messages to be stored
	time.Sleep(200 * time.Millisecond)

	// Replay messages from the beginning
	replayFrom := baseTime.Add(-1 * time.Second)
	replayedMessages, err := bus.Replay(ctx, workflowID, replayFrom)
	require.NoError(t, err)

	// Verify we got all messages in chronological order
	assert.Len(t, replayedMessages, expectedMessages)

	for i := 0; i < len(replayedMessages); i++ {
		assert.Equal(t, fmt.Sprintf("workflow-msg-%d", i), replayedMessages[i].ID)
		if i > 0 {
			assert.True(t, replayedMessages[i-1].Timestamp.Before(replayedMessages[i].Timestamp))
		}
	}
}

func TestNATSBus_ConnectionRetry(t *testing.T) {
	// Test connection retry with invalid URL
	config := &BusConfig{
		URL:            "nats://invalid-host:4222",
		MaxReconnect:   2,
		ReconnectWait:  100 * time.Millisecond,
		AckWait:        5 * time.Second,
		MaxInFlight:    100,
		ConnectTimeout: 1 * time.Second,
		RequestTimeout: 5 * time.Second,
	}

	start := time.Now()
	bus, err := NewNATSBus(config)
	duration := time.Since(start)

	// Should fail after retries
	assert.Error(t, err)
	assert.Nil(t, bus)

	// Should have taken at least the retry time
	expectedMinDuration := time.Duration(config.MaxReconnect-1) * config.ReconnectWait
	assert.True(t, duration >= expectedMinDuration,
		"Connection retry took %v, expected at least %v", duration, expectedMinDuration)
}

func TestNATSBus_StreamConfiguration(t *testing.T) {
	ctx := context.Background()

	// Start NATS container
	natsContainer, err := StartNATSContainer(ctx)
	require.NoError(t, err)
	defer natsContainer.Stop(ctx)

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

	natsBus := bus.(*natsBus)

	// Verify AF_MESSAGES stream configuration
	info, err := natsBus.js.StreamInfo(StreamAFMessages)
	require.NoError(t, err)
	assert.Equal(t, StreamAFMessages, info.Config.Name)
	assert.Contains(t, info.Config.Subjects, "workflows.*.*")
	assert.Contains(t, info.Config.Subjects, "agents.*.*")
	assert.Equal(t, nats.FileStorage, info.Config.Storage)
	assert.Equal(t, 168*time.Hour, info.Config.MaxAge)

	// Verify AF_TOOLS stream configuration
	info, err = natsBus.js.StreamInfo(StreamAFTools)
	require.NoError(t, err)
	assert.Equal(t, StreamAFTools, info.Config.Name)
	assert.Contains(t, info.Config.Subjects, "tools.*")
	assert.Equal(t, 720*time.Hour, info.Config.MaxAge)

	// Verify AF_SYSTEM stream configuration
	info, err = natsBus.js.StreamInfo(StreamAFSystem)
	require.NoError(t, err)
	assert.Equal(t, StreamAFSystem, info.Config.Name)
	assert.Contains(t, info.Config.Subjects, "system.*")
	assert.Equal(t, 24*time.Hour, info.Config.MaxAge)
}

func TestNATSBus_MessageValidation(t *testing.T) {
	ctx := context.Background()

	// Start NATS container
	natsContainer, err := StartNATSContainer(ctx)
	require.NoError(t, err)
	defer natsContainer.Stop(ctx)

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

	natsBus := bus.(*natsBus)
	subject := "agents.test-agent.in"

	// Create message with valid hash
	msg := NewMessage("test-id", "sender", "test-agent", MessageTypeRequest)
	msg.SetPayload(map[string]interface{}{"test": "data"})

	// Set envelope hash properly
	err = natsBus.serializer.SetEnvelopeHash(msg)
	require.NoError(t, err)

	// Set up subscription that tracks validation results
	validMessages := make(chan *Message, 1)
	invalidMessages := make(chan bool, 1)

	handler := func(ctx context.Context, receivedMsg *Message) error {
		validMessages <- receivedMsg
		return nil
	}

	sub, err := bus.Subscribe(ctx, subject, handler)
	require.NoError(t, err)
	defer sub.Unsubscribe()

	// Publish valid message
	err = bus.Publish(ctx, subject, msg)
	require.NoError(t, err)

	// Should receive valid message
	select {
	case receivedMsg := <-validMessages:
		assert.Equal(t, msg.ID, receivedMsg.ID)
		assert.Equal(t, msg.EnvelopeHash, receivedMsg.EnvelopeHash)
	case <-time.After(3 * time.Second):
		t.Fatal("Valid message not received")
	}

	// Now test with tampered message (invalid hash)
	tamperedMsg := *msg
	tamperedMsg.EnvelopeHash = "invalid-hash"

	// Publish tampered message - this should be rejected by the consumer
	// Note: The message will be published but rejected during consumption
	err = bus.Publish(ctx, subject, &tamperedMsg)
	require.NoError(t, err)

	// Should not receive the tampered message (it gets NAK'd)
	select {
	case <-validMessages:
		t.Fatal("Should not have received tampered message")
	case <-invalidMessages:
		// Expected - message was rejected
	case <-time.After(2 * time.Second):
		// Expected - message was rejected and not delivered to handler
	}
}
