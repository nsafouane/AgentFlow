package messaging

import (
	"testing"
	"time"
)

func TestCanonicalSerializer_Determinism(t *testing.T) {
	serializer, err := NewCanonicalSerializer()
	if err != nil {
		t.Fatalf("Failed to create serializer: %v", err)
	}

	// Create a test message with complex nested data
	msg := &Message{
		ID:      "01HN8ZQJKM9XVQZJKM9XVQZJKM",
		TraceID: "abcdef1234567890abcdef1234567890",
		SpanID:  "1234567890abcdef",
		From:    "agent-1",
		To:      "agent-2",
		Type:    MessageTypeRequest,
		Payload: map[string]interface{}{
			"action": "process",
			"data": map[string]interface{}{
				"items": []interface{}{
					map[string]interface{}{
						"id":   "item-1",
						"name": "Test Item",
					},
					map[string]interface{}{
						"id":   "item-2",
						"name": "Another Item",
					},
				},
				"metadata": map[string]interface{}{
					"version": "1.0",
					"source":  "test",
				},
			},
		},
		Metadata: map[string]interface{}{
			"workflow_id": "wf-123",
			"step":        "process-data",
			"retry_count": 0,
		},
		Cost: CostInfo{
			Tokens:  100,
			Dollars: 0.01,
		},
		Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
	}

	// Set the envelope hash
	err = serializer.SetEnvelopeHash(msg)
	if err != nil {
		t.Fatalf("Failed to set envelope hash: %v", err)
	}

	// Serialize the message multiple times
	const iterations = 10
	hashes := make([]string, iterations)
	serializedData := make([][]byte, iterations)

	for i := 0; i < iterations; i++ {
		data, err := serializer.Serialize(msg)
		if err != nil {
			t.Fatalf("Serialization %d failed: %v", i, err)
		}
		serializedData[i] = data

		// Compute hash for this serialization
		hash, err := serializer.ComputeHash(msg)
		if err != nil {
			t.Fatalf("Hash computation %d failed: %v", i, err)
		}
		hashes[i] = hash
	}

	// Verify all serializations are identical
	firstSerialization := string(serializedData[0])
	for i := 1; i < iterations; i++ {
		if string(serializedData[i]) != firstSerialization {
			t.Errorf("Serialization %d differs from first serialization", i)
			t.Logf("First: %s", firstSerialization)
			t.Logf("Current: %s", string(serializedData[i]))
		}
	}

	// Verify all hashes are identical
	firstHash := hashes[0]
	for i := 1; i < iterations; i++ {
		if hashes[i] != firstHash {
			t.Errorf("Hash %d differs from first hash: %s != %s", i, hashes[i], firstHash)
		}
	}
}

func TestCanonicalSerializer_FieldOrderingIndependence(t *testing.T) {
	serializer, err := NewCanonicalSerializer()
	if err != nil {
		t.Fatalf("Failed to create serializer: %v", err)
	}

	// Create two messages with the same content but different field ordering in maps
	baseTime := time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC)

	msg1 := &Message{
		ID:        "01HN8ZQJKM9XVQZJKM9XVQZJKM",
		TraceID:   "abcdef1234567890abcdef1234567890",
		SpanID:    "1234567890abcdef",
		From:      "agent-1",
		To:        "agent-2",
		Type:      MessageTypeRequest,
		Timestamp: baseTime,
		Payload: map[string]interface{}{
			"a": "value-a",
			"b": "value-b",
			"c": "value-c",
		},
		Metadata: map[string]interface{}{
			"x": "value-x",
			"y": "value-y",
			"z": "value-z",
		},
		Cost: CostInfo{Tokens: 100, Dollars: 0.01},
	}

	msg2 := &Message{
		ID:        "01HN8ZQJKM9XVQZJKM9XVQZJKM",
		TraceID:   "abcdef1234567890abcdef1234567890",
		SpanID:    "1234567890abcdef",
		From:      "agent-1",
		To:        "agent-2",
		Type:      MessageTypeRequest,
		Timestamp: baseTime,
		Payload: map[string]interface{}{
			"c": "value-c",
			"a": "value-a",
			"b": "value-b",
		},
		Metadata: map[string]interface{}{
			"z": "value-z",
			"x": "value-x",
			"y": "value-y",
		},
		Cost: CostInfo{Tokens: 100, Dollars: 0.01},
	}

	// Compute hashes for both messages
	hash1, err := serializer.ComputeHash(msg1)
	if err != nil {
		t.Fatalf("Failed to compute hash for msg1: %v", err)
	}

	hash2, err := serializer.ComputeHash(msg2)
	if err != nil {
		t.Fatalf("Failed to compute hash for msg2: %v", err)
	}

	// Hashes should be identical despite different field ordering
	if hash1 != hash2 {
		t.Errorf("Hashes differ despite identical content: %s != %s", hash1, hash2)
	}

	// Serializations should also be identical
	data1, err := serializer.Serialize(msg1)
	if err != nil {
		t.Fatalf("Failed to serialize msg1: %v", err)
	}

	data2, err := serializer.Serialize(msg2)
	if err != nil {
		t.Fatalf("Failed to serialize msg2: %v", err)
	}

	if string(data1) != string(data2) {
		t.Errorf("Serializations differ despite identical content")
		t.Logf("Msg1: %s", string(data1))
		t.Logf("Msg2: %s", string(data2))
	}
}

func TestCanonicalSerializer_HashValidation(t *testing.T) {
	serializer, err := NewCanonicalSerializer()
	if err != nil {
		t.Fatalf("Failed to create serializer: %v", err)
	}

	msg := &Message{
		ID:        "01HN8ZQJKM9XVQZJKM9XVQZJKM",
		From:      "agent-1",
		To:        "agent-2",
		Type:      MessageTypeRequest,
		Timestamp: time.Now().UTC(),
		Cost:      CostInfo{Tokens: 50, Dollars: 0.005},
	}

	// Set the correct envelope hash
	err = serializer.SetEnvelopeHash(msg)
	if err != nil {
		t.Fatalf("Failed to set envelope hash: %v", err)
	}

	// Validation should pass
	err = serializer.ValidateHash(msg)
	if err != nil {
		t.Errorf("Hash validation failed for valid message: %v", err)
	}

	// Tamper with the message content
	originalFrom := msg.From
	msg.From = "tampered-agent"

	// Validation should fail
	err = serializer.ValidateHash(msg)
	if err == nil {
		t.Error("Hash validation should have failed for tampered message")
	}

	// Restore original content
	msg.From = originalFrom

	// Tamper with the hash itself
	msg.EnvelopeHash = "invalid-hash"

	// Validation should fail
	err = serializer.ValidateHash(msg)
	if err == nil {
		t.Error("Hash validation should have failed for invalid hash")
	}
}

func TestCanonicalSerializer_SerializeDeserialize(t *testing.T) {
	serializer, err := NewCanonicalSerializer()
	if err != nil {
		t.Fatalf("Failed to create serializer: %v", err)
	}

	original := &Message{
		ID:      "01HN8ZQJKM9XVQZJKM9XVQZJKM",
		TraceID: "abcdef1234567890abcdef1234567890",
		SpanID:  "1234567890abcdef",
		From:    "agent-1",
		To:      "agent-2",
		Type:    MessageTypeEvent,
		Payload: map[string]interface{}{
			"event": "user_action",
			"data":  "test data",
		},
		Metadata: map[string]interface{}{
			"source": "ui",
		},
		Cost:      CostInfo{Tokens: 25, Dollars: 0.0025},
		Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
	}

	// Set envelope hash
	err = serializer.SetEnvelopeHash(original)
	if err != nil {
		t.Fatalf("Failed to set envelope hash: %v", err)
	}

	// Serialize
	data, err := serializer.Serialize(original)
	if err != nil {
		t.Fatalf("Serialization failed: %v", err)
	}

	// Deserialize
	deserialized, err := serializer.Deserialize(data)
	if err != nil {
		t.Fatalf("Deserialization failed: %v", err)
	}

	// Compare key fields
	if deserialized.ID != original.ID {
		t.Errorf("ID mismatch: %s != %s", deserialized.ID, original.ID)
	}
	if deserialized.From != original.From {
		t.Errorf("From mismatch: %s != %s", deserialized.From, original.From)
	}
	if deserialized.To != original.To {
		t.Errorf("To mismatch: %s != %s", deserialized.To, original.To)
	}
	if deserialized.Type != original.Type {
		t.Errorf("Type mismatch: %s != %s", deserialized.Type, original.Type)
	}
	if deserialized.EnvelopeHash != original.EnvelopeHash {
		t.Errorf("EnvelopeHash mismatch: %s != %s", deserialized.EnvelopeHash, original.EnvelopeHash)
	}

	// Validate the deserialized message hash
	err = serializer.ValidateHash(deserialized)
	if err != nil {
		t.Errorf("Deserialized message hash validation failed: %v", err)
	}
}
