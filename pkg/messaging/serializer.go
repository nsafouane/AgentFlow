package messaging

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"time"
)

// CanonicalSerializer provides deterministic message serialization and hashing
type CanonicalSerializer struct {
	validator *MessageValidator
}

// NewCanonicalSerializer creates a new canonical serializer
func NewCanonicalSerializer() (*CanonicalSerializer, error) {
	validator, err := NewMessageValidator()
	if err != nil {
		return nil, fmt.Errorf("failed to create validator: %w", err)
	}

	return &CanonicalSerializer{
		validator: validator,
	}, nil
}

// Serialize converts a message to canonical JSON bytes with deterministic ordering
func (s *CanonicalSerializer) Serialize(msg *Message) ([]byte, error) {
	// Create canonical representation
	canonical := s.toCanonicalMap(msg)

	// Marshal with deterministic ordering
	data, err := s.marshalCanonical(canonical)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal canonical message: %w", err)
	}

	return data, nil
}

// Deserialize converts JSON bytes back to a Message
func (s *CanonicalSerializer) Deserialize(data []byte) (*Message, error) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal message: %w", err)
	}

	// Validate the deserialized message
	if err := s.validator.Validate(&msg); err != nil {
		return nil, fmt.Errorf("deserialized message validation failed: %w", err)
	}

	return &msg, nil
}

// ComputeHash computes the SHA256 hash of the canonical message content
func (s *CanonicalSerializer) ComputeHash(msg *Message) (string, error) {
	// Create a copy without the envelope_hash field for hashing
	msgCopy := *msg
	msgCopy.EnvelopeHash = ""

	// Serialize to canonical form
	data, err := s.Serialize(&msgCopy)
	if err != nil {
		return "", fmt.Errorf("failed to serialize message for hashing: %w", err)
	}

	// Compute SHA256 hash
	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash), nil
}

// ValidateHash verifies that the message's envelope_hash matches its content
func (s *CanonicalSerializer) ValidateHash(msg *Message) error {
	expectedHash, err := s.ComputeHash(msg)
	if err != nil {
		return fmt.Errorf("failed to compute expected hash: %w", err)
	}

	if msg.EnvelopeHash != expectedHash {
		return fmt.Errorf("hash mismatch: expected %s, got %s", expectedHash, msg.EnvelopeHash)
	}

	return nil
}

// SetEnvelopeHash computes and sets the envelope hash for a message
func (s *CanonicalSerializer) SetEnvelopeHash(msg *Message) error {
	hash, err := s.ComputeHash(msg)
	if err != nil {
		return fmt.Errorf("failed to compute envelope hash: %w", err)
	}

	msg.EnvelopeHash = hash
	return nil
}

// toCanonicalMap converts a message to a canonical map representation
func (s *CanonicalSerializer) toCanonicalMap(msg *Message) map[string]interface{} {
	canonical := make(map[string]interface{})

	// Add fields in deterministic order
	canonical["id"] = msg.ID
	canonical["trace_id"] = msg.TraceID
	canonical["span_id"] = msg.SpanID
	canonical["from"] = msg.From
	canonical["to"] = msg.To
	canonical["type"] = string(msg.Type)
	canonical["payload"] = s.canonicalizeValue(msg.Payload)
	canonical["metadata"] = s.canonicalizeValue(msg.Metadata)
	canonical["cost"] = map[string]interface{}{
		"tokens":  msg.Cost.Tokens,
		"dollars": msg.Cost.Dollars,
	}
	canonical["ts"] = msg.Timestamp.Format(time.RFC3339Nano)

	// Only include envelope_hash if it's not empty
	if msg.EnvelopeHash != "" {
		canonical["envelope_hash"] = msg.EnvelopeHash
	}

	return canonical
}

// canonicalizeValue recursively canonicalizes any value for deterministic serialization
func (s *CanonicalSerializer) canonicalizeValue(v interface{}) interface{} {
	if v == nil {
		return nil
	}

	val := reflect.ValueOf(v)
	switch val.Kind() {
	case reflect.Map:
		result := make(map[string]interface{})
		for _, key := range val.MapKeys() {
			keyStr := fmt.Sprintf("%v", key.Interface())
			result[keyStr] = s.canonicalizeValue(val.MapIndex(key).Interface())
		}
		return result

	case reflect.Slice, reflect.Array:
		result := make([]interface{}, val.Len())
		for i := 0; i < val.Len(); i++ {
			result[i] = s.canonicalizeValue(val.Index(i).Interface())
		}
		return result

	case reflect.Struct:
		// Convert struct to map for canonical representation
		result := make(map[string]interface{})
		typ := val.Type()
		for i := 0; i < val.NumField(); i++ {
			field := typ.Field(i)
			if field.IsExported() {
				jsonTag := field.Tag.Get("json")
				if jsonTag != "" && jsonTag != "-" {
					// Use JSON tag name if available
					tagName := jsonTag
					if idx := len(jsonTag); idx > 0 {
						if commaIdx := 0; commaIdx < len(jsonTag) {
							for j, r := range jsonTag {
								if r == ',' {
									commaIdx = j
									break
								}
							}
							if commaIdx > 0 {
								tagName = jsonTag[:commaIdx]
							}
						}
					}
					result[tagName] = s.canonicalizeValue(val.Field(i).Interface())
				} else {
					result[field.Name] = s.canonicalizeValue(val.Field(i).Interface())
				}
			}
		}
		return result

	default:
		return v
	}
}

// marshalCanonical marshals a map with deterministic key ordering
func (s *CanonicalSerializer) marshalCanonical(data map[string]interface{}) ([]byte, error) {
	// Sort keys for deterministic output
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// Build JSON manually to ensure key ordering
	result := "{"
	for i, key := range keys {
		if i > 0 {
			result += ","
		}

		keyBytes, err := json.Marshal(key)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal key %s: %w", key, err)
		}

		valueBytes, err := s.marshalCanonicalValue(data[key])
		if err != nil {
			return nil, fmt.Errorf("failed to marshal value for key %s: %w", key, err)
		}

		result += string(keyBytes) + ":" + string(valueBytes)
	}
	result += "}"

	return []byte(result), nil
}

// marshalCanonicalValue marshals any value with canonical ordering
func (s *CanonicalSerializer) marshalCanonicalValue(v interface{}) ([]byte, error) {
	if v == nil {
		return []byte("null"), nil
	}

	switch val := v.(type) {
	case map[string]interface{}:
		return s.marshalCanonical(val)
	case []interface{}:
		result := "["
		for i, item := range val {
			if i > 0 {
				result += ","
			}
			itemBytes, err := s.marshalCanonicalValue(item)
			if err != nil {
				return nil, fmt.Errorf("failed to marshal array item %d: %w", i, err)
			}
			result += string(itemBytes)
		}
		result += "]"
		return []byte(result), nil
	default:
		return json.Marshal(v)
	}
}
