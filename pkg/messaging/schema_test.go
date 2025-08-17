package messaging

import (
	"testing"
	"time"
)

func TestMessageValidator_ValidMessage(t *testing.T) {
	validator, err := NewMessageValidator()
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	validMsg := &Message{
		ID:           "01HN8ZQJKM9XVQZJKM9XVQZJKM",
		TraceID:      "abcdef1234567890abcdef1234567890",
		SpanID:       "1234567890abcdef",
		From:         "agent-1",
		To:           "agent-2",
		Type:         MessageTypeRequest,
		Payload:      map[string]interface{}{"action": "test"},
		Metadata:     map[string]interface{}{"workflow": "test-wf"},
		Cost:         CostInfo{Tokens: 100, Dollars: 0.01},
		Timestamp:    time.Now().UTC(),
		EnvelopeHash: "a1b2c3d4e5f67890123456789012345678901234567890123456789012345678",
	}

	err = validator.Validate(validMsg)
	if err != nil {
		t.Errorf("Valid message failed validation: %v", err)
	}
}

func TestMessageValidator_MissingRequiredFields(t *testing.T) {
	validator, err := NewMessageValidator()
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	testCases := []struct {
		name    string
		message *Message
	}{
		{
			name: "missing ID",
			message: &Message{
				From:      "agent-1",
				To:        "agent-2",
				Type:      MessageTypeRequest,
				Timestamp: time.Now().UTC(),
			},
		},
		{
			name: "missing From",
			message: &Message{
				ID:        "01HN8ZQJKM9XVQZJKM9XVQZJKM",
				To:        "agent-2",
				Type:      MessageTypeRequest,
				Timestamp: time.Now().UTC(),
			},
		},
		{
			name: "missing To",
			message: &Message{
				ID:        "01HN8ZQJKM9XVQZJKM9XVQZJKM",
				From:      "agent-1",
				Type:      MessageTypeRequest,
				Timestamp: time.Now().UTC(),
			},
		},
		{
			name: "missing Type",
			message: &Message{
				ID:        "01HN8ZQJKM9XVQZJKM9XVQZJKM",
				From:      "agent-1",
				To:        "agent-2",
				Timestamp: time.Now().UTC(),
			},
		},
		// Note: Timestamp test removed because Go's zero-value time.Time
		// still serializes to a valid RFC3339 timestamp
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validator.Validate(tc.message)
			if err == nil {
				t.Errorf("Expected validation to fail for %s", tc.name)
			}
		})
	}
}

func TestMessageValidator_InvalidFieldFormats(t *testing.T) {
	validator, err := NewMessageValidator()
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	testCases := []struct {
		name     string
		jsonData string
	}{
		{
			name: "invalid ULID format",
			jsonData: `{
				"id": "invalid-ulid",
				"from": "agent-1",
				"to": "agent-2",
				"type": "request",
				"ts": "2024-01-01T12:00:00Z"
			}`,
		},
		{
			name: "invalid trace_id format",
			jsonData: `{
				"id": "01HN8ZQJKM9XVQZJKM9XVQZJKM",
				"trace_id": "invalid-trace-id",
				"from": "agent-1",
				"to": "agent-2",
				"type": "request",
				"ts": "2024-01-01T12:00:00Z"
			}`,
		},
		{
			name: "invalid span_id format",
			jsonData: `{
				"id": "01HN8ZQJKM9XVQZJKM9XVQZJKM",
				"span_id": "invalid-span",
				"from": "agent-1",
				"to": "agent-2",
				"type": "request",
				"ts": "2024-01-01T12:00:00Z"
			}`,
		},
		{
			name: "invalid message type",
			jsonData: `{
				"id": "01HN8ZQJKM9XVQZJKM9XVQZJKM",
				"from": "agent-1",
				"to": "agent-2",
				"type": "invalid-type",
				"ts": "2024-01-01T12:00:00Z"
			}`,
		},
		{
			name: "invalid envelope_hash format",
			jsonData: `{
				"id": "01HN8ZQJKM9XVQZJKM9XVQZJKM",
				"from": "agent-1",
				"to": "agent-2",
				"type": "request",
				"ts": "2024-01-01T12:00:00Z",
				"envelope_hash": "invalid-hash"
			}`,
		},
		{
			name: "negative cost tokens",
			jsonData: `{
				"id": "01HN8ZQJKM9XVQZJKM9XVQZJKM",
				"from": "agent-1",
				"to": "agent-2",
				"type": "request",
				"ts": "2024-01-01T12:00:00Z",
				"cost": {
					"tokens": -10,
					"dollars": 0.01
				}
			}`,
		},
		{
			name: "negative cost dollars",
			jsonData: `{
				"id": "01HN8ZQJKM9XVQZJKM9XVQZJKM",
				"from": "agent-1",
				"to": "agent-2",
				"type": "request",
				"ts": "2024-01-01T12:00:00Z",
				"cost": {
					"tokens": 10,
					"dollars": -0.01
				}
			}`,
		},
		{
			name: "missing timestamp",
			jsonData: `{
				"id": "01HN8ZQJKM9XVQZJKM9XVQZJKM",
				"from": "agent-1",
				"to": "agent-2",
				"type": "request"
			}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validator.ValidateJSON([]byte(tc.jsonData))
			if err == nil {
				t.Errorf("Expected validation to fail for %s", tc.name)
			}
		})
	}
}

func TestMessageValidator_ValidFieldFormats(t *testing.T) {
	validator, err := NewMessageValidator()
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	testCases := []struct {
		name     string
		jsonData string
	}{
		{
			name: "minimal valid message",
			jsonData: `{
				"id": "01HN8ZQJKM9XVQZJKM9XVQZJKM",
				"from": "agent-1",
				"to": "agent-2",
				"type": "request",
				"ts": "2024-01-01T12:00:00Z"
			}`,
		},
		{
			name: "complete valid message",
			jsonData: `{
				"id": "01HN8ZQJKM9XVQZJKM9XVQZJKM",
				"trace_id": "abcdef1234567890abcdef1234567890",
				"span_id": "1234567890abcdef",
				"from": "agent-1",
				"to": "agent-2",
				"type": "response",
				"payload": {"result": "success"},
				"metadata": {"workflow": "test"},
				"cost": {
					"tokens": 100,
					"dollars": 0.01
				},
				"ts": "2024-01-01T12:00:00.123456789Z",
				"envelope_hash": "a1b2c3d4e5f67890123456789012345678901234567890123456789012345678"
			}`,
		},
		{
			name: "event message type",
			jsonData: `{
				"id": "01HN8ZQJKM9XVQZJKM9XVQZJKM",
				"from": "agent-1",
				"to": "system.events",
				"type": "event",
				"ts": "2024-01-01T12:00:00Z"
			}`,
		},
		{
			name: "control message type",
			jsonData: `{
				"id": "01HN8ZQJKM9XVQZJKM9XVQZJKM",
				"from": "control-plane",
				"to": "agent-1",
				"type": "control",
				"ts": "2024-01-01T12:00:00Z"
			}`,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			err := validator.ValidateJSON([]byte(tc.jsonData))
			if err != nil {
				t.Errorf("Valid message failed validation for %s: %v", tc.name, err)
			}
		})
	}
}

func TestMessageValidator_BackwardCompatibility(t *testing.T) {
	validator, err := NewMessageValidator()
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	// Test that messages without optional fields still validate
	minimalJSON := `{
		"id": "01HN8ZQJKM9XVQZJKM9XVQZJKM",
		"from": "agent-1",
		"to": "agent-2",
		"type": "request",
		"ts": "2024-01-01T12:00:00Z"
	}`

	err = validator.ValidateJSON([]byte(minimalJSON))
	if err != nil {
		t.Errorf("Minimal message should validate: %v", err)
	}

	// Test that additional properties are rejected (strict schema)
	extendedJSON := `{
		"id": "01HN8ZQJKM9XVQZJKM9XVQZJKM",
		"from": "agent-1",
		"to": "agent-2",
		"type": "request",
		"ts": "2024-01-01T12:00:00Z",
		"unknown_field": "should_fail"
	}`

	err = validator.ValidateJSON([]byte(extendedJSON))
	if err == nil {
		t.Error("Message with unknown fields should fail validation")
	}
}

func TestMessageValidator_CostValidation(t *testing.T) {
	validator, err := NewMessageValidator()
	if err != nil {
		t.Fatalf("Failed to create validator: %v", err)
	}

	// Test valid cost structure
	validCostJSON := `{
		"id": "01HN8ZQJKM9XVQZJKM9XVQZJKM",
		"from": "agent-1",
		"to": "agent-2",
		"type": "request",
		"ts": "2024-01-01T12:00:00Z",
		"cost": {
			"tokens": 100,
			"dollars": 0.01
		}
	}`

	err = validator.ValidateJSON([]byte(validCostJSON))
	if err != nil {
		t.Errorf("Valid cost structure should validate: %v", err)
	}

	// Test incomplete cost structure (missing required fields)
	incompleteCostJSON := `{
		"id": "01HN8ZQJKM9XVQZJKM9XVQZJKM",
		"from": "agent-1",
		"to": "agent-2",
		"type": "request",
		"ts": "2024-01-01T12:00:00Z",
		"cost": {
			"tokens": 100
		}
	}`

	err = validator.ValidateJSON([]byte(incompleteCostJSON))
	if err == nil {
		t.Error("Incomplete cost structure should fail validation")
	}
}
