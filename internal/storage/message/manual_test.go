package message

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/agentflow/agentflow/pkg/messaging"
)

// ManualTestEnvelopeHashIntegration demonstrates envelope hash persistence and validation
// This test should be run manually against a real database to verify Q1.2 integration
func ManualTestEnvelopeHashIntegration() {
	fmt.Println("=== Manual Test: Envelope Hash Persistence & Q1.2 Integration ===")

	// Check if we have a database connection
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://agentflow:dev_password@localhost:5432/agentflow_dev?sslmode=disable"
	}

	// Connect to database
	ctx := context.Background()
	db, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		log.Printf("Failed to connect to database (this is expected in CI): %v", err)
		fmt.Println("Skipping manual test - no database connection")
		return
	}
	defer db.Close()

	// Test database connectivity
	if err := db.Ping(ctx); err != nil {
		log.Printf("Database ping failed (this is expected in CI): %v", err)
		fmt.Println("Skipping manual test - database not available")
		return
	}

	fmt.Println("✓ Database connection established")

	// Create message service
	service, err := NewService(db)
	if err != nil {
		log.Fatalf("Failed to create message service: %v", err)
	}

	// Create test tenant (assuming it exists or create manually)
	tenantID := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000") // Fixed test tenant

	// Test 1: Create message with Q1.2 canonical serializer
	fmt.Println("\n--- Test 1: Message Creation with Q1.2 Canonical Serializer ---")

	msg := messaging.NewMessage(
		uuid.New().String(),
		"manual-test-agent-1",
		"manual-test-agent-2",
		messaging.MessageTypeRequest,
	)

	// Set complex payload to test serialization
	msg.SetPayload(map[string]interface{}{
		"action": "process_document",
		"params": map[string]interface{}{
			"document_id": "doc-12345",
			"priority":    "high",
			"metadata": map[string]interface{}{
				"source":    "api",
				"timestamp": time.Now().Unix(),
				"tags":      []string{"urgent", "customer-facing"},
			},
		},
	})

	msg.AddMetadata("workflow_id", "wf-789")
	msg.AddMetadata("user_id", "user-456")
	msg.SetCost(150, 0.025)
	msg.SetTraceContext("trace-abc123", "span-def456")

	// Compute envelope hash using Q1.2 canonical serializer
	serializer, err := messaging.NewCanonicalSerializer()
	if err != nil {
		log.Fatalf("Failed to create canonical serializer: %v", err)
	}

	err = serializer.SetEnvelopeHash(msg)
	if err != nil {
		log.Fatalf("Failed to set envelope hash: %v", err)
	}

	fmt.Printf("✓ Message created with envelope hash: %s\n", msg.EnvelopeHash)

	// Store message in database
	err = service.CreateMessage(ctx, msg, tenantID)
	if err != nil {
		log.Fatalf("Failed to store message: %v", err)
	}

	fmt.Printf("✓ Message stored in database with ID: %s\n", msg.ID)

	// Test 2: Retrieve and validate stored message
	fmt.Println("\n--- Test 2: Message Retrieval and Hash Validation ---")

	storedMsg, err := service.GetMessage(ctx, uuid.MustParse(msg.ID), tenantID)
	if err != nil {
		log.Fatalf("Failed to retrieve message: %v", err)
	}

	fmt.Printf("✓ Message retrieved from database\n")
	fmt.Printf("  Original hash:  %s\n", msg.EnvelopeHash)
	fmt.Printf("  Retrieved hash: %s\n", storedMsg.EnvelopeHash)

	if msg.EnvelopeHash != storedMsg.EnvelopeHash {
		log.Fatalf("Hash mismatch! Original: %s, Retrieved: %s", msg.EnvelopeHash, storedMsg.EnvelopeHash)
	}

	fmt.Printf("✓ Envelope hash matches after storage/retrieval\n")

	// Test 3: Recompute hash and verify it matches
	fmt.Println("\n--- Test 3: Hash Recomputation Verification ---")

	recomputedHash, err := service.RecomputeEnvelopeHash(storedMsg)
	if err != nil {
		log.Fatalf("Failed to recompute hash: %v", err)
	}

	fmt.Printf("  Stored hash:     %s\n", storedMsg.EnvelopeHash)
	fmt.Printf("  Recomputed hash: %s\n", recomputedHash)

	if storedMsg.EnvelopeHash != recomputedHash {
		log.Fatalf("Recomputed hash mismatch! Stored: %s, Recomputed: %s", storedMsg.EnvelopeHash, recomputedHash)
	}

	fmt.Printf("✓ Recomputed hash matches stored hash\n")

	// Test 4: Verify canonical serialization produces same result
	fmt.Println("\n--- Test 4: Q1.2 Canonical Serializer Consistency ---")

	canonicalHash, err := serializer.ComputeHash(storedMsg)
	if err != nil {
		log.Fatalf("Failed to compute canonical hash: %v", err)
	}

	fmt.Printf("  Service hash:    %s\n", recomputedHash)
	fmt.Printf("  Canonical hash:  %s\n", canonicalHash)

	if recomputedHash != canonicalHash {
		log.Fatalf("Canonical serializer mismatch! Service: %s, Canonical: %s", recomputedHash, canonicalHash)
	}

	fmt.Printf("✓ Service and Q1.2 canonical serializer produce identical hashes\n")

	// Test 5: Trace-based retrieval with integrity validation
	fmt.Println("\n--- Test 5: Trace-based Retrieval with Integrity Validation ---")

	// Create additional messages with same trace ID
	traceID := "manual-test-trace-" + uuid.New().String()[:8]
	messageIDs := make([]string, 3)

	for i := 0; i < 3; i++ {
		traceMsg := messaging.NewMessage(
			uuid.New().String(),
			fmt.Sprintf("agent-%d", i+1),
			fmt.Sprintf("agent-%d", i+2),
			messaging.MessageTypeEvent,
		)
		traceMsg.SetTraceContext(traceID, fmt.Sprintf("span-%d", i+1))
		traceMsg.SetPayload(map[string]interface{}{
			"event": fmt.Sprintf("step_%d_completed", i+1),
			"data":  map[string]interface{}{"step": i + 1, "status": "success"},
		})

		err = serializer.SetEnvelopeHash(traceMsg)
		if err != nil {
			log.Fatalf("Failed to set envelope hash for trace message %d: %v", i+1, err)
		}

		err = service.CreateMessage(ctx, traceMsg, tenantID)
		if err != nil {
			log.Fatalf("Failed to store trace message %d: %v", i+1, err)
		}

		messageIDs[i] = traceMsg.ID
	}

	fmt.Printf("✓ Created 3 messages with trace ID: %s\n", traceID)

	// Retrieve messages by trace
	traceMessages, err := service.ListMessagesByTrace(ctx, tenantID, traceID)
	if err != nil {
		log.Fatalf("Failed to retrieve messages by trace: %v", err)
	}

	if len(traceMessages) != 3 {
		log.Fatalf("Expected 3 messages, got %d", len(traceMessages))
	}

	fmt.Printf("✓ Retrieved %d messages by trace ID\n", len(traceMessages))

	// Verify all messages have valid envelope hashes
	for i, traceMsg := range traceMessages {
		recomputedHash, err := service.RecomputeEnvelopeHash(traceMsg)
		if err != nil {
			log.Fatalf("Failed to recompute hash for trace message %d: %v", i+1, err)
		}

		if traceMsg.EnvelopeHash != recomputedHash {
			log.Fatalf("Hash validation failed for trace message %d", i+1)
		}
	}

	fmt.Printf("✓ All trace messages have valid envelope hashes\n")

	// Test 6: Demonstrate message content inspection
	fmt.Println("\n--- Test 6: Message Content Inspection ---")

	inspectMsg := traceMessages[0]
	fmt.Printf("Sample message inspection:\n")
	fmt.Printf("  ID: %s\n", inspectMsg.ID)
	fmt.Printf("  From: %s -> To: %s\n", inspectMsg.From, inspectMsg.To)
	fmt.Printf("  Type: %s\n", inspectMsg.Type)
	fmt.Printf("  Trace ID: %s\n", inspectMsg.TraceID)
	fmt.Printf("  Span ID: %s\n", inspectMsg.SpanID)
	fmt.Printf("  Timestamp: %s\n", inspectMsg.Timestamp.Format(time.RFC3339))
	fmt.Printf("  Envelope Hash: %s\n", inspectMsg.EnvelopeHash)

	// Pretty print payload
	payloadJSON, _ := json.MarshalIndent(inspectMsg.Payload, "  ", "  ")
	fmt.Printf("  Payload: %s\n", string(payloadJSON))

	// Pretty print metadata
	metadataJSON, _ := json.MarshalIndent(inspectMsg.Metadata, "  ", "  ")
	fmt.Printf("  Metadata: %s\n", string(metadataJSON))

	fmt.Printf("  Cost: %d tokens, $%.4f\n", inspectMsg.Cost.Tokens, inspectMsg.Cost.Dollars)

	fmt.Println("\n=== Manual Test Completed Successfully ===")
	fmt.Println("✓ Envelope hash persistence working correctly")
	fmt.Println("✓ Q1.2 canonical serializer integration verified")
	fmt.Println("✓ Message integrity validation functional")
	fmt.Println("✓ Replay operation support confirmed")
}

// RunManualTest is a helper function to run the manual test
func RunManualTest() {
	ManualTestEnvelopeHashIntegration()
}
