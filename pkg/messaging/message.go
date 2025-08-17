package messaging

import (
	"time"
)

// MessageType represents the type of message
type MessageType string

const (
	MessageTypeRequest  MessageType = "request"
	MessageTypeResponse MessageType = "response"
	MessageTypeEvent    MessageType = "event"
	MessageTypeControl  MessageType = "control"
)

// CostInfo tracks token and dollar costs for message processing
type CostInfo struct {
	Tokens  int     `json:"tokens"`
	Dollars float64 `json:"dollars"`
}

// Message represents a complete AgentFlow message with all required fields
type Message struct {
	ID           string                 `json:"id"`            // ULID
	TraceID      string                 `json:"trace_id"`      // OpenTelemetry trace ID
	SpanID       string                 `json:"span_id"`       // OpenTelemetry span ID
	From         string                 `json:"from"`          // Source agent ID
	To           string                 `json:"to"`            // Target agent ID or topic
	Type         MessageType            `json:"type"`          // request, response, event, control
	Payload      interface{}            `json:"payload"`       // Message-specific data
	Metadata     map[string]interface{} `json:"metadata"`      // Workflow context
	Cost         CostInfo               `json:"cost"`          // Token/dollar tracking
	Timestamp    time.Time              `json:"ts"`            // RFC3339 timestamp
	EnvelopeHash string                 `json:"envelope_hash"` // SHA256 of canonical content
}

// NewMessage creates a new message with required fields
func NewMessage(id, from, to string, msgType MessageType) *Message {
	return &Message{
		ID:        id,
		From:      from,
		To:        to,
		Type:      msgType,
		Metadata:  make(map[string]interface{}),
		Cost:      CostInfo{},
		Timestamp: time.Now().UTC(),
	}
}

// SetTraceContext sets the OpenTelemetry trace context
func (m *Message) SetTraceContext(traceID, spanID string) {
	m.TraceID = traceID
	m.SpanID = spanID
}

// SetPayload sets the message payload
func (m *Message) SetPayload(payload interface{}) {
	m.Payload = payload
}

// AddMetadata adds a metadata key-value pair
func (m *Message) AddMetadata(key string, value interface{}) {
	if m.Metadata == nil {
		m.Metadata = make(map[string]interface{})
	}
	m.Metadata[key] = value
}

// SetCost sets the cost information
func (m *Message) SetCost(tokens int, dollars float64) {
	m.Cost = CostInfo{
		Tokens:  tokens,
		Dollars: dollars,
	}
}
