package messaging

import (
	"testing"
	"time"
)

func TestMessageHelpers(t *testing.T) {
	id := "01TESTID000000000000000000"
	m := NewMessage(id, "agent-a", "agent-b", MessageTypeRequest)

	if m == nil {
		t.Fatal("NewMessage returned nil")
	}
	if m.ID != id {
		t.Fatalf("expected id %s got %s", id, m.ID)
	}
	if m.From != "agent-a" || m.To != "agent-b" || m.Type != MessageTypeRequest {
		t.Fatalf("unexpected basic fields: %+v", m)
	}
	if m.Metadata == nil {
		t.Fatalf("expected metadata to be initialized")
	}
	if m.Timestamp.IsZero() {
		t.Fatalf("expected timestamp to be set")
	}
	// Trace context helpers
	m.SetTraceContext("trace-1", "span-1")
	if m.TraceID != "trace-1" || m.SpanID != "span-1" {
		t.Fatalf("trace context not set: %v %v", m.TraceID, m.SpanID)
	}

	// Payload setter
	m.SetPayload(map[string]interface{}{"k": "v"})
	if m.Payload == nil {
		t.Fatalf("payload was not set")
	}

	// Set cost
	m.SetCost(123, 4.56)
	if m.Cost.Tokens != 123 || m.Cost.Dollars != 4.56 {
		t.Fatalf("cost not set correctly: %+v", m.Cost)
	}

	// Add metadata
	m.AddMetadata("k2", "v2")
	if val, ok := m.Metadata["k2"]; !ok || val != "v2" {
		t.Fatalf("metadata add failed: %+v", m.Metadata)
	}

	// Basic time sanity: timestamp no later than now + 1s
	if m.Timestamp.After(time.Now().Add(time.Second)) {
		t.Fatalf("timestamp is in the future: %v", m.Timestamp)
	}
}
