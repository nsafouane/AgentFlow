package messaging

import (
	"encoding/json"
	"fmt"

	"github.com/xeipuuv/gojsonschema"
)

// MessageSchema defines the JSON schema for message validation
const MessageSchema = `{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "type": "object",
  "required": ["id", "from", "to", "type", "ts"],
  "properties": {
    "id": {
      "type": "string",
      "pattern": "^[0-9A-HJKMNP-TV-Z]{26}$",
      "description": "ULID identifier"
    },
    "trace_id": {
      "anyOf": [
        {"type": "string", "pattern": "^[a-f0-9]{32}$"},
        {"type": "string", "maxLength": 0}
      ],
      "description": "OpenTelemetry trace ID"
    },
    "span_id": {
      "anyOf": [
        {"type": "string", "pattern": "^[a-f0-9]{16}$"},
        {"type": "string", "maxLength": 0}
      ],
      "description": "OpenTelemetry span ID"
    },
    "from": {
      "type": "string",
      "minLength": 1,
      "description": "Source agent ID"
    },
    "to": {
      "type": "string",
      "minLength": 1,
      "description": "Target agent ID or topic"
    },
    "type": {
      "type": "string",
      "enum": ["request", "response", "event", "control"],
      "description": "Message type"
    },
    "payload": {
      "description": "Message-specific data"
    },
    "metadata": {
      "anyOf": [
        {"type": "object", "additionalProperties": true},
        {"type": "null"}
      ],
      "description": "Workflow context"
    },
    "cost": {
      "type": "object",
      "properties": {
        "tokens": {
          "type": "integer",
          "minimum": 0
        },
        "dollars": {
          "type": "number",
          "minimum": 0
        }
      },
      "required": ["tokens", "dollars"],
      "additionalProperties": false
    },
    "ts": {
      "type": "string",
      "format": "date-time",
      "description": "RFC3339 timestamp"
    },
    "envelope_hash": {
      "anyOf": [
        {"type": "string", "pattern": "^[a-f0-9]{64}$"},
        {"type": "string", "maxLength": 0}
      ],
      "description": "SHA256 hash of canonical content"
    }
  },
  "additionalProperties": false
}`

// MessageValidator provides JSON schema validation for messages
type MessageValidator struct {
	schema *gojsonschema.Schema
}

// NewMessageValidator creates a new message validator
func NewMessageValidator() (*MessageValidator, error) {
	schemaLoader := gojsonschema.NewStringLoader(MessageSchema)
	schema, err := gojsonschema.NewSchema(schemaLoader)
	if err != nil {
		return nil, fmt.Errorf("failed to compile message schema: %w", err)
	}

	return &MessageValidator{
		schema: schema,
	}, nil
}

// Validate validates a message against the JSON schema
func (v *MessageValidator) Validate(msg *Message) error {
	// Convert message to JSON for validation
	msgBytes, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message for validation: %w", err)
	}

	documentLoader := gojsonschema.NewBytesLoader(msgBytes)
	result, err := v.schema.Validate(documentLoader)
	if err != nil {
		return fmt.Errorf("validation error: %w", err)
	}

	if !result.Valid() {
		var errors []string
		for _, desc := range result.Errors() {
			errors = append(errors, desc.String())
		}
		return fmt.Errorf("message validation failed: %v", errors)
	}

	return nil
}

// ValidateJSON validates a JSON byte slice against the message schema
func (v *MessageValidator) ValidateJSON(data []byte) error {
	documentLoader := gojsonschema.NewBytesLoader(data)
	result, err := v.schema.Validate(documentLoader)
	if err != nil {
		return fmt.Errorf("validation error: %w", err)
	}

	if !result.Valid() {
		var errors []string
		for _, desc := range result.Errors() {
			errors = append(errors, desc.String())
		}
		return fmt.Errorf("JSON validation failed: %v", errors)
	}

	return nil
}
