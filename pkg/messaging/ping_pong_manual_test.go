//go:build manual

package messaging

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/agentflow/agentflow/internal/logging"
)

// ManualPingPongTest demonstrates structured logging during a ping-pong message scenario
// Run this test manually with: go test -tags=manual -v ./pkg/messaging -run TestManualPingPong
// Set AF_TEST_MESSAGING_NOISY=1 to see additional debug output
func TestManualPingPong(t *testing.T) {
	// Enable tracing for this manual test
	t.Setenv("AF_TRACING_ENABLED", "true")
	t.Setenv("AF_TEST_MESSAGING_NOISY", "1")

	ctx := context.Background()

	// Start NATS container
	natsContainer, err := StartNATSContainer(ctx)
	if err != nil {
		t.Fatalf("Failed to start NATS container: %v", err)
	}
	defer natsContainer.Stop(ctx)

	// Create bus with default logger (outputs to stdout)
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
	if err != nil {
		t.Fatalf("Failed to create NATS bus: %v", err)
	}
	defer bus.Close()

	// Create a logger for the test
	logger := logging.NewLogger()
	logger.Info("Starting ping-pong manual test",
		logging.String("test", "ping_pong_scenario"))

	// Set up Agent A (ping sender)
	agentASubject := "agents.agent-a.in"
	agentBSubject := "agents.agent-b.in"

	// Agent A will receive pong messages
	agentAReceived := make(chan *Message, 10)
	agentAHandler := func(ctx context.Context, msg *Message) error {
		msgLogger := logger.WithTrace(ctx).WithMessage(msg.ID).WithAgent("agent-a")
		msgLogger.Info("Agent A received pong message",
			logging.String("from", msg.From),
			logging.String("payload_type", fmt.Sprintf("%T", msg.Payload)))

		agentAReceived <- msg
		return nil
	}

	// Agent B will receive ping messages and respond with pong
	agentBReceived := make(chan *Message, 10)
	agentBHandler := func(ctx context.Context, msg *Message) error {
		msgLogger := logger.WithTrace(ctx).WithMessage(msg.ID).WithAgent("agent-b")
		msgLogger.Info("Agent B received ping message",
			logging.String("from", msg.From),
			logging.String("payload_type", fmt.Sprintf("%T", msg.Payload)))

		// Create pong response
		pongMsg := NewMessage(
			fmt.Sprintf("pong-%d", time.Now().UnixNano()),
			"agent-b",
			"agent-a",
			MessageTypeResponse)

		pongMsg.SetPayload(map[string]interface{}{
			"type":             "pong",
			"original_ping_id": msg.ID,
			"timestamp":        time.Now().UTC(),
		})

		// Set trace context from incoming message
		pongMsg.SetTraceContext(msg.TraceID, msg.SpanID)

		// Set envelope hash
		natsBus := bus.(*natsBus)
		if err := natsBus.serializer.SetEnvelopeHash(pongMsg); err != nil {
			msgLogger.Error("Failed to set envelope hash for pong", err)
			return err
		}

		msgLogger.Info("Agent B sending pong response",
			logging.String("pong_id", pongMsg.ID),
			logging.String("original_ping", msg.ID))

		// Send pong back to Agent A
		if err := bus.Publish(ctx, agentASubject, pongMsg); err != nil {
			msgLogger.Error("Failed to send pong message", err)
			return err
		}

		agentBReceived <- msg
		return nil
	}

	// Subscribe both agents
	subA, err := bus.Subscribe(ctx, agentASubject, agentAHandler)
	if err != nil {
		t.Fatalf("Failed to subscribe Agent A: %v", err)
	}
	defer subA.Unsubscribe()

	subB, err := bus.Subscribe(ctx, agentBSubject, agentBHandler)
	if err != nil {
		t.Fatalf("Failed to subscribe Agent B: %v", err)
	}
	defer subB.Unsubscribe()

	// Give subscriptions time to be ready
	time.Sleep(100 * time.Millisecond)

	logger.Info("Subscriptions ready, starting ping-pong sequence")

	// Send ping messages from Agent A to Agent B
	numPings := 3
	for i := 0; i < numPings; i++ {
		pingMsg := NewMessage(
			fmt.Sprintf("ping-%d", i+1),
			"agent-a",
			"agent-b",
			MessageTypeRequest)

		pingMsg.SetPayload(map[string]interface{}{
			"type":      "ping",
			"sequence":  i + 1,
			"timestamp": time.Now().UTC(),
			"message":   fmt.Sprintf("Hello from Agent A - ping #%d", i+1),
		})

		// Set envelope hash
		natsBus := bus.(*natsBus)
		if err := natsBus.serializer.SetEnvelopeHash(pingMsg); err != nil {
			t.Fatalf("Failed to set envelope hash for ping: %v", err)
		}

		pingLogger := logger.WithMessage(pingMsg.ID).WithAgent("agent-a")
		pingLogger.Info("Sending ping message",
			logging.Int("sequence", i+1),
			logging.String("target", "agent-b"))

		// Send ping to Agent B
		if err := bus.Publish(ctx, agentBSubject, pingMsg); err != nil {
			t.Fatalf("Failed to send ping message: %v", err)
		}

		// Small delay between pings
		time.Sleep(100 * time.Millisecond)
	}

	// Wait for all ping-pong exchanges to complete
	logger.Info("Waiting for ping-pong exchanges to complete")

	// Collect all received messages
	timeout := time.After(10 * time.Second)
	receivedPings := 0
	receivedPongs := 0

	for receivedPings < numPings || receivedPongs < numPings {
		select {
		case <-agentBReceived:
			receivedPings++
			logger.Info("Ping received by Agent B", logging.Int("total_pings", receivedPings))
		case <-agentAReceived:
			receivedPongs++
			logger.Info("Pong received by Agent A", logging.Int("total_pongs", receivedPongs))
		case <-timeout:
			t.Fatalf("Timeout waiting for ping-pong completion. Pings: %d/%d, Pongs: %d/%d",
				receivedPings, numPings, receivedPongs, numPings)
		}
	}

	logger.Info("Ping-pong test completed successfully",
		logging.Int("total_exchanges", numPings),
		logging.String("status", "success"))

	// Print summary
	fmt.Printf("\n=== PING-PONG TEST SUMMARY ===\n")
	fmt.Printf("Total ping messages sent: %d\n", numPings)
	fmt.Printf("Total ping messages received: %d\n", receivedPings)
	fmt.Printf("Total pong messages received: %d\n", receivedPongs)
	fmt.Printf("All messages processed successfully!\n")
	fmt.Printf("\n=== LOG ANALYSIS ===\n")
	fmt.Printf("Review the log output above to verify:\n")
	fmt.Printf("1. All log entries are in JSON format\n")
	fmt.Printf("2. Each log entry contains correlation fields (trace_id, span_id, message_id)\n")
	fmt.Printf("3. Agent-specific logs include agent_id field\n")
	fmt.Printf("4. Message processing logs include message metadata\n")
	fmt.Printf("5. Error logs (if any) include proper error context\n")
	fmt.Printf("\nTo analyze logs programmatically, pipe output through jq:\n")
	fmt.Printf("go test -tags=manual -v ./pkg/messaging -run TestManualPingPong 2>&1 | grep '^{' | jq .\n")
}

// TestManualLogTailing demonstrates how to tail logs during message processing
// This test is designed to be run manually to observe log output
func TestManualLogTailing(t *testing.T) {
	// This test is meant to be run manually with log tailing
	// Example usage:
	// Terminal 1: go test -tags=manual -v ./pkg/messaging -run TestManualLogTailing
	// Terminal 2: tail -f /tmp/agentflow-test.log | jq .

	t.Skip("This is a manual test for log tailing demonstration")

	// Create a temporary log file
	logFile, err := os.CreateTemp("", "agentflow-test-*.log")
	if err != nil {
		t.Fatalf("Failed to create temp log file: %v", err)
	}
	defer os.Remove(logFile.Name())

	fmt.Printf("Log file created: %s\n", logFile.Name())
	fmt.Printf("In another terminal, run: tail -f %s | jq .\n", logFile.Name())
	fmt.Printf("Press Enter to continue...")
	fmt.Scanln()

	// Create logger that writes to file
	logger := logging.NewLoggerWithOutput(logFile)

	// Simulate some message processing
	for i := 0; i < 10; i++ {
		msgLogger := logger.WithMessage(fmt.Sprintf("msg-%d", i)).WithAgent("test-agent")
		msgLogger.Info("Processing test message",
			logging.Int("sequence", i),
			logging.String("operation", "test_processing"))

		time.Sleep(500 * time.Millisecond)
	}

	fmt.Printf("Test completed. Check the log file: %s\n", logFile.Name())
}
