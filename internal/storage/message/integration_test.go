//go:build integration
// +build integration

package message

import (
	"context"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/agentflow/agentflow/pkg/messaging"
)

func TestIntegration(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		t.Skip("DATABASE_URL not set")
	}

	ctx := context.Background()
	db, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		t.Skip("Database not available")
	}
	defer db.Close()

	service, err := NewService(db)
	require.NoError(t, err)

	tenantID := uuid.New()
	msg := messaging.NewMessage(uuid.New().String(), "a1", "a2", messaging.MessageTypeRequest)

	serializer, err := messaging.NewCanonicalSerializer()
	require.NoError(t, err)
	err = serializer.SetEnvelopeHash(msg)
	require.NoError(t, err)

	err = service.CreateMessage(ctx, msg, tenantID)
	require.NoError(t, err)

	storedMsg, err := service.GetMessage(ctx, uuid.MustParse(msg.ID), tenantID)
	require.NoError(t, err)
	assert.Equal(t, msg.EnvelopeHash, storedMsg.EnvelopeHash)
}
