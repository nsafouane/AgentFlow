package message

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/agentflow/agentflow/internal/storage/queries"
	"github.com/agentflow/agentflow/pkg/messaging"
)

// MessageQuerier defines the interface for message queries
type MessageQuerier interface {
	CreateMessage(ctx context.Context, arg queries.CreateMessageParams) (queries.Message, error)
	GetMessage(ctx context.Context, arg queries.GetMessageParams) (queries.Message, error)
	ListMessagesByTrace(ctx context.Context, arg queries.ListMessagesByTraceParams) ([]queries.Message, error)
}

// Service provides message storage operations with envelope hash validation
type Service struct {
	db         *pgxpool.Pool
	queries    MessageQuerier
	serializer *messaging.CanonicalSerializer
}

// NewService creates a new message service
func NewService(db *pgxpool.Pool) (*Service, error) {
	serializer, err := messaging.NewCanonicalSerializer()
	if err != nil {
		return nil, fmt.Errorf("failed to create canonical serializer: %w", err)
	}

	return &Service{
		db:         db,
		queries:    queries.New(db),
		serializer: serializer,
	}, nil
}

// CreateMessage stores a message with envelope hash validation
func (s *Service) CreateMessage(ctx context.Context, msg *messaging.Message, tenantID uuid.UUID) error {
	// Validate that envelope_hash is present
	if msg.EnvelopeHash == "" {
		return fmt.Errorf("envelope_hash is required but missing")
	}

	// Validate envelope hash integrity
	if err := s.serializer.ValidateHash(msg); err != nil {
		return fmt.Errorf("envelope hash validation failed: %w", err)
	}

	// Convert message to database format
	dbMsg, err := s.messageToDBParams(msg, tenantID)
	if err != nil {
		return fmt.Errorf("failed to convert message to database format: %w", err)
	}

	// Store in database
	_, err = s.queries.CreateMessage(ctx, dbMsg)
	if err != nil {
		return fmt.Errorf("failed to store message: %w", err)
	}

	return nil
}

// GetMessage retrieves a message and validates its envelope hash
func (s *Service) GetMessage(ctx context.Context, messageID, tenantID uuid.UUID) (*messaging.Message, error) {
	// Retrieve from database
	dbMsg, err := s.queries.GetMessage(ctx, queries.GetMessageParams{
		ID:       pgtype.UUID{Bytes: messageID, Valid: true},
		TenantID: pgtype.UUID{Bytes: tenantID, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve message: %w", err)
	}

	// Convert to messaging format
	msg, err := s.dbMessageToMessage(&dbMsg)
	if err != nil {
		return nil, fmt.Errorf("failed to convert database message: %w", err)
	}

	// Validate envelope hash integrity
	if err := s.serializer.ValidateHash(msg); err != nil {
		return nil, fmt.Errorf("stored message envelope hash validation failed: %w", err)
	}

	return msg, nil
}

// ValidateMessageIntegrity recomputes and validates the envelope hash for a stored message
func (s *Service) ValidateMessageIntegrity(ctx context.Context, messageID, tenantID uuid.UUID) error {
	msg, err := s.GetMessage(ctx, messageID, tenantID)
	if err != nil {
		return fmt.Errorf("failed to retrieve message for validation: %w", err)
	}

	// The GetMessage method already validates the hash, so if we get here, it's valid
	_ = msg
	return nil
}

// ListMessagesByTrace retrieves messages for a trace and validates their integrity
func (s *Service) ListMessagesByTrace(ctx context.Context, tenantID uuid.UUID, traceID string) ([]*messaging.Message, error) {
	dbMessages, err := s.queries.ListMessagesByTrace(ctx, queries.ListMessagesByTraceParams{
		TenantID: pgtype.UUID{Bytes: tenantID, Valid: true},
		TraceID:  pgtype.Text{String: traceID, Valid: true},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve messages by trace: %w", err)
	}

	messages := make([]*messaging.Message, 0, len(dbMessages))
	for _, dbMsg := range dbMessages {
		msg, err := s.dbMessageToMessage(&dbMsg)
		if err != nil {
			return nil, fmt.Errorf("failed to convert database message: %w", err)
		}

		// Validate envelope hash integrity for replay operations
		if err := s.serializer.ValidateHash(msg); err != nil {
			return nil, fmt.Errorf("message %s envelope hash validation failed during replay: %w", msg.ID, err)
		}

		messages = append(messages, msg)
	}

	return messages, nil
}

// RecomputeEnvelopeHash recomputes the envelope hash for a message (for testing/verification)
func (s *Service) RecomputeEnvelopeHash(msg *messaging.Message) (string, error) {
	return s.serializer.ComputeHash(msg)
}

// messageToDBParams converts a messaging.Message to database parameters
func (s *Service) messageToDBParams(msg *messaging.Message, tenantID uuid.UUID) (queries.CreateMessageParams, error) {
	// Parse message ID as UUID
	msgUUID, err := uuid.Parse(msg.ID)
	if err != nil {
		return queries.CreateMessageParams{}, fmt.Errorf("invalid message ID format: %w", err)
	}

	// Marshal JSON fields
	payloadBytes, err := json.Marshal(msg.Payload)
	if err != nil {
		return queries.CreateMessageParams{}, fmt.Errorf("failed to marshal payload: %w", err)
	}

	metadataBytes, err := json.Marshal(msg.Metadata)
	if err != nil {
		return queries.CreateMessageParams{}, fmt.Errorf("failed to marshal metadata: %w", err)
	}

	costBytes, err := json.Marshal(msg.Cost)
	if err != nil {
		return queries.CreateMessageParams{}, fmt.Errorf("failed to marshal cost: %w", err)
	}

	return queries.CreateMessageParams{
		ID:           pgtype.UUID{Bytes: msgUUID, Valid: true},
		TenantID:     pgtype.UUID{Bytes: tenantID, Valid: true},
		TraceID:      pgtype.Text{String: msg.TraceID, Valid: msg.TraceID != ""},
		SpanID:       pgtype.Text{String: msg.SpanID, Valid: msg.SpanID != ""},
		FromAgent:    msg.From,
		ToAgent:      msg.To,
		Type:         string(msg.Type),
		Payload:      payloadBytes,
		Metadata:     metadataBytes,
		Cost:         costBytes,
		Ts:           pgtype.Timestamptz{Time: msg.Timestamp, Valid: true},
		EnvelopeHash: msg.EnvelopeHash,
	}, nil
}

// dbMessageToMessage converts a database Message to messaging.Message
func (s *Service) dbMessageToMessage(dbMsg *queries.Message) (*messaging.Message, error) {
	// Convert UUID to string
	msgID := uuid.UUID(dbMsg.ID.Bytes).String()

	// Unmarshal JSON fields
	var payload interface{}
	if err := json.Unmarshal(dbMsg.Payload, &payload); err != nil {
		return nil, fmt.Errorf("failed to unmarshal payload: %w", err)
	}

	var metadata map[string]interface{}
	if err := json.Unmarshal(dbMsg.Metadata, &metadata); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metadata: %w", err)
	}

	var cost messaging.CostInfo
	if err := json.Unmarshal(dbMsg.Cost, &cost); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cost: %w", err)
	}

	return &messaging.Message{
		ID:           msgID,
		TraceID:      dbMsg.TraceID.String,
		SpanID:       dbMsg.SpanID.String,
		From:         dbMsg.FromAgent,
		To:           dbMsg.ToAgent,
		Type:         messaging.MessageType(dbMsg.Type),
		Payload:      payload,
		Metadata:     metadata,
		Cost:         cost,
		Timestamp:    dbMsg.Ts.Time,
		EnvelopeHash: dbMsg.EnvelopeHash,
	}, nil
}
