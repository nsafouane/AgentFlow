package message

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/agentflow/agentflow/pkg/messaging"
)

func TestService_CreateMessage(t *testing.T) {
	mockQueries := NewMockQueries()
	service := &Service{
		queries: mockQueries,
	}

	// Initialize serializer
	serializer, err := messaging.NewCanonicalSerializer()
	require.NoError(t, err)
	service.serializer = serializer

	tenantID := uuid.New()
	ctx := context.Background()

	t.Run("successful message creation with valid envelope hash", func(t *testing.T) {
		msg := createTestMessage(t)

		// Set envelope hash using canonical serializer
		serializer, err := messaging.NewCanonicalSerializer()
		require.NoError(t, err)
		err = serializer.SetEnvelopeHash(msg)
		require.NoError(t, err)

		err = service.CreateMessage(ctx, msg, tenantID)
		assert.NoError(t, err)

		// Verify message was stored
		storedMsg, err := service.GetMessage(ctx, uuid.MustParse(msg.ID), tenantID)
		require.NoError(t, err)
		assert.Equal(t, msg.ID, storedMsg.ID)
		assert.Equal(t, msg.EnvelopeHash, storedMsg.EnvelopeHash)
	})

	t.Run("reject message with missing envelope hash", func(t *testing.T) {
		msg := createTestMessage(t)
		msg.EnvelopeHash = "" // Missing hash

		err := service.CreateMessage(ctx, msg, tenantID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "envelope_hash is required but missing")
	})

	t.Run("reject message with invalid envelope hash", func(t *testing.T) {
		msg := createTestMessage(t)
		msg.EnvelopeHash = "invalid_hash" // Invalid hash

		err := service.CreateMessage(ctx, msg, tenantID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "envelope hash validation failed")
	})

	t.Run("reject message with tampered content", func(t *testing.T) {
		msg := createTestMessage(t)

		// Set valid envelope hash
		serializer, err := messaging.NewCanonicalSerializer()
		require.NoError(t, err)
		err = serializer.SetEnvelopeHash(msg)
		require.NoError(t, err)

		// Tamper with content after hash is set
		msg.Payload = map[string]interface{}{"tampered": "data"}

		err = service.CreateMessage(ctx, msg, tenantID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "envelope hash validation failed")
	})
}

func TestService_GetMessage(t *testing.T) {
	mockQueries := NewMockQueries()
	service := &Service{
		queries: mockQueries,
	}

	// Initialize serializer
	serializer, err := messaging.NewCanonicalSerializer()
	require.NoError(t, err)
	service.serializer = serializer

	tenantID := uuid.New()
	ctx := context.Background()

	t.Run("retrieve message with valid envelope hash", func(t *testing.T) {
		msg := createTestMessage(t)

		// Set envelope hash and store
		serializer, err := messaging.NewCanonicalSerializer()
		require.NoError(t, err)
		err = serializer.SetEnvelopeHash(msg)
		require.NoError(t, err)

		err = service.CreateMessage(ctx, msg, tenantID)
		require.NoError(t, err)

		// Retrieve and verify
		storedMsg, err := service.GetMessage(ctx, uuid.MustParse(msg.ID), tenantID)
		require.NoError(t, err)
		assert.Equal(t, msg.ID, storedMsg.ID)
		assert.Equal(t, msg.From, storedMsg.From)
		assert.Equal(t, msg.To, storedMsg.To)
		assert.Equal(t, msg.EnvelopeHash, storedMsg.EnvelopeHash)
	})

	t.Run("fail on non-existent message", func(t *testing.T) {
		nonExistentID := uuid.New()
		_, err := service.GetMessage(ctx, nonExistentID, tenantID)
		assert.Error(t, err)
	})
}

func TestService_ValidateMessageIntegrity(t *testing.T) {
	mockQueries := NewMockQueries()
	service := &Service{
		queries: mockQueries,
	}

	// Initialize serializer
	serializer, err := messaging.NewCanonicalSerializer()
	require.NoError(t, err)
	service.serializer = serializer

	tenantID := uuid.New()
	ctx := context.Background()

	t.Run("validate integrity of stored message", func(t *testing.T) {
		msg := createTestMessage(t)

		// Set envelope hash and store
		serializer, err := messaging.NewCanonicalSerializer()
		require.NoError(t, err)
		err = serializer.SetEnvelopeHash(msg)
		require.NoError(t, err)

		err = service.CreateMessage(ctx, msg, tenantID)
		require.NoError(t, err)

		// Validate integrity
		err = service.ValidateMessageIntegrity(ctx, uuid.MustParse(msg.ID), tenantID)
		assert.NoError(t, err)
	})
}

func TestService_ListMessagesByTrace(t *testing.T) {
	mockQueries := NewMockQueries()
	service := &Service{
		queries: mockQueries,
	}

	// Initialize serializer
	serializer, err := messaging.NewCanonicalSerializer()
	require.NoError(t, err)
	service.serializer = serializer

	tenantID := uuid.New()
	ctx := context.Background()
	traceID := "test-trace-123"

	t.Run("retrieve messages by trace with integrity validation", func(t *testing.T) {
		// Create multiple messages with same trace ID
		messages := make([]*messaging.Message, 3)
		serializer, err := messaging.NewCanonicalSerializer()
		require.NoError(t, err)

		for i := 0; i < 3; i++ {
			msg := createTestMessage(t)
			msg.TraceID = traceID
			err = serializer.SetEnvelopeHash(msg)
			require.NoError(t, err)

			err = service.CreateMessage(ctx, msg, tenantID)
			require.NoError(t, err)
			messages[i] = msg
		}

		// Retrieve by trace
		retrievedMessages, err := service.ListMessagesByTrace(ctx, tenantID, traceID)
		require.NoError(t, err)
		assert.Len(t, retrievedMessages, 3)

		// Verify all messages have valid envelope hashes
		for _, msg := range retrievedMessages {
			assert.NotEmpty(t, msg.EnvelopeHash)
			assert.Equal(t, traceID, msg.TraceID)
		}
	})
}

func TestService_RecomputeEnvelopeHash(t *testing.T) {
	mockQueries := NewMockQueries()
	service := &Service{
		queries: mockQueries,
	}

	// Initialize serializer
	serializer, err := messaging.NewCanonicalSerializer()
	require.NoError(t, err)
	service.serializer = serializer

	t.Run("recompute envelope hash matches canonical serializer", func(t *testing.T) {
		msg := createTestMessage(t)

		// Compute hash using service
		serviceHash, err := service.RecomputeEnvelopeHash(msg)
		require.NoError(t, err)

		// Compute hash using canonical serializer directly
		serializer, err := messaging.NewCanonicalSerializer()
		require.NoError(t, err)
		canonicalHash, err := serializer.ComputeHash(msg)
		require.NoError(t, err)

		assert.Equal(t, canonicalHash, serviceHash)
		assert.NotEmpty(t, serviceHash)
		assert.Len(t, serviceHash, 64) // SHA256 hex string length
	})
}

func TestService_EnvelopeHashIntegrationWithQ12(t *testing.T) {
	mockQueries := NewMockQueries()
	service := &Service{
		queries: mockQueries,
	}

	// Initialize serializer
	serializer, err := messaging.NewCanonicalSerializer()
	require.NoError(t, err)
	service.serializer = serializer

	tenantID := uuid.New()
	ctx := context.Background()

	t.Run("integration with Q1.2 canonical serializer", func(t *testing.T) {
		// Create message using Q1.2 messaging package
		msg := messaging.NewMessage(
			uuid.New().String(),
			"agent-1",
			"agent-2",
			messaging.MessageTypeRequest,
		)
		msg.SetPayload(map[string]interface{}{
			"action": "process_document",
			"params": map[string]interface{}{
				"document_id": "doc-123",
				"priority":    "high",
			},
		})
		msg.AddMetadata("workflow_id", "wf-456")
		msg.SetCost(100, 0.01)

		// Set envelope hash using Q1.2 canonical serializer
		serializer, err := messaging.NewCanonicalSerializer()
		require.NoError(t, err)
		err = serializer.SetEnvelopeHash(msg)
		require.NoError(t, err)

		// Store message
		err = service.CreateMessage(ctx, msg, tenantID)
		require.NoError(t, err)

		// Retrieve and verify hash matches
		storedMsg, err := service.GetMessage(ctx, uuid.MustParse(msg.ID), tenantID)
		require.NoError(t, err)
		assert.Equal(t, msg.EnvelopeHash, storedMsg.EnvelopeHash)

		// Verify we can recompute the same hash
		recomputedHash, err := service.RecomputeEnvelopeHash(storedMsg)
		require.NoError(t, err)
		assert.Equal(t, msg.EnvelopeHash, recomputedHash)
	})
}

// Helper functions

func createTestMessage(t *testing.T) *messaging.Message {
	msg := messaging.NewMessage(
		uuid.New().String(),
		"test-agent-1",
		"test-agent-2",
		messaging.MessageTypeRequest,
	)
	msg.SetPayload(map[string]interface{}{
		"test":   "data",
		"number": 42,
	})
	msg.AddMetadata("test_key", "test_value")
	msg.SetCost(10, 0.001)
	msg.SetTraceContext("trace-123", "span-456")
	return msg
}
