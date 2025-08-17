package messaging

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

// TestManualSampleMessages creates and inspects sample messages for manual verification
func TestManualSampleMessages(t *testing.T) {
	ManualTestSampleMessages()
}

// ManualTestSampleMessages creates and inspects sample messages for manual verification
func ManualTestSampleMessages() {
	fmt.Println("=== Manual Test: Sample Messages ===")

	serializer, err := NewCanonicalSerializer()
	if err != nil {
		fmt.Printf("Failed to create serializer: %v\n", err)
		return
	}

	// Test 1: Basic message
	fmt.Println("\n1. Basic Message:")
	basicMsg := &Message{
		ID:        "01HN8ZQJKM9XVQZJKM9XVQZJKM",
		From:      "agent-1",
		To:        "agent-2",
		Type:      MessageTypeRequest,
		Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		Cost:      CostInfo{Tokens: 50, Dollars: 0.005},
	}

	err = serializer.SetEnvelopeHash(basicMsg)
	if err != nil {
		fmt.Printf("Failed to set envelope hash: %v\n", err)
		return
	}

	basicData, _ := serializer.Serialize(basicMsg)
	fmt.Printf("Serialized: %s\n", string(basicData))
	fmt.Printf("Hash: %s\n", basicMsg.EnvelopeHash)

	// Test 2: Complex message with nested data
	fmt.Println("\n2. Complex Message with Nested Data:")
	complexMsg := &Message{
		ID:      "01HN8ZQJKM9XVQZJKM9XVQZJKN",
		TraceID: "abcdef1234567890abcdef1234567890",
		SpanID:  "1234567890abcdef",
		From:    "orchestrator",
		To:      "workflow.user-support.in",
		Type:    MessageTypeEvent,
		Payload: map[string]interface{}{
			"event_type": "user_query",
			"data": map[string]interface{}{
				"user_id": "user-123",
				"query":   "How do I reset my password?",
				"context": map[string]interface{}{
					"session_id": "sess-456",
					"timestamp":  "2024-01-01T12:00:00Z",
					"metadata": map[string]interface{}{
						"source":     "web",
						"user_agent": "Mozilla/5.0...",
					},
				},
			},
		},
		Metadata: map[string]interface{}{
			"workflow_id": "wf-user-support-789",
			"step":        "initial-query",
			"priority":    "normal",
			"retry_count": 0,
			"max_retries": 3,
			"timeout_ms":  30000,
		},
		Cost:      CostInfo{Tokens: 150, Dollars: 0.015},
		Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 123456789, time.UTC),
	}

	err = serializer.SetEnvelopeHash(complexMsg)
	if err != nil {
		fmt.Printf("Failed to set envelope hash: %v\n", err)
		return
	}

	complexData, _ := serializer.Serialize(complexMsg)
	fmt.Printf("Serialized: %s\n", string(complexData))
	fmt.Printf("Hash: %s\n", complexMsg.EnvelopeHash)

	// Test 3: Backward compatibility - deserialize and re-serialize
	fmt.Println("\n3. Backward Compatibility Test:")

	// Simulate an "old" message format (without some newer fields)
	oldMessageJSON := `{
		"id": "01HN8ZQJKM9XVQZJKM9XVQZJK0",
		"from": "legacy-agent",
		"to": "new-agent",
		"type": "request",
		"payload": {"action": "legacy_action"},
		"metadata": {},
		"cost": {"tokens": 25, "dollars": 0.0025},
		"ts": "2024-01-01T12:00:00Z",
		"trace_id": "",
		"span_id": "",
		"envelope_hash": ""
	}`

	fmt.Printf("Old format JSON: %s\n", oldMessageJSON)

	// Deserialize old format
	oldMsg, err := serializer.Deserialize([]byte(oldMessageJSON))
	if err != nil {
		fmt.Printf("Failed to deserialize old message: %v\n", err)
		return
	}

	// Set envelope hash for the deserialized message
	err = serializer.SetEnvelopeHash(oldMsg)
	if err != nil {
		fmt.Printf("Failed to set envelope hash for old message: %v\n", err)
		return
	}

	// Re-serialize with new format
	newData, _ := serializer.Serialize(oldMsg)
	fmt.Printf("Re-serialized: %s\n", string(newData))
	fmt.Printf("New hash: %s\n", oldMsg.EnvelopeHash)

	// Test 4: Field ordering independence
	fmt.Println("\n4. Field Ordering Independence Test:")

	// Create two identical messages with different internal map ordering
	msg1 := &Message{
		ID:        "01HN8ZQJKM9XVQZJKM9XVQZJKP",
		From:      "agent-a",
		To:        "agent-b",
		Type:      MessageTypeResponse,
		Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		Payload: map[string]interface{}{
			"result": "success",
			"data":   "test-data",
			"code":   200,
		},
		Metadata: map[string]interface{}{
			"workflow": "test-wf",
			"step":     "final",
			"version":  "1.0",
		},
		Cost: CostInfo{Tokens: 75, Dollars: 0.0075},
	}

	msg2 := &Message{
		ID:        "01HN8ZQJKM9XVQZJKM9XVQZJKP",
		From:      "agent-a",
		To:        "agent-b",
		Type:      MessageTypeResponse,
		Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		Payload: map[string]interface{}{
			"code":   200,
			"result": "success",
			"data":   "test-data",
		},
		Metadata: map[string]interface{}{
			"version":  "1.0",
			"workflow": "test-wf",
			"step":     "final",
		},
		Cost: CostInfo{Tokens: 75, Dollars: 0.0075},
	}

	hash1, _ := serializer.ComputeHash(msg1)
	hash2, _ := serializer.ComputeHash(msg2)

	fmt.Printf("Message 1 hash: %s\n", hash1)
	fmt.Printf("Message 2 hash: %s\n", hash2)
	fmt.Printf("Hashes match: %t\n", hash1 == hash2)

	data1, _ := serializer.Serialize(msg1)
	data2, _ := serializer.Serialize(msg2)
	fmt.Printf("Serializations match: %t\n", string(data1) == string(data2))

	// Test 5: Schema evolution scenario
	fmt.Println("\n5. Schema Evolution Scenario:")

	// Simulate adding a new optional field to the message
	currentMsg := &Message{
		ID:        "01HN8ZQJKM9XVQZJKM9XVQZJKQ",
		From:      "new-agent",
		To:        "another-agent",
		Type:      MessageTypeEvent,
		Timestamp: time.Date(2024, 1, 1, 12, 0, 0, 0, time.UTC),
		Cost:      CostInfo{Tokens: 100, Dollars: 0.01},
	}

	// Add some metadata that might represent new fields
	currentMsg.AddMetadata("schema_version", "1.1")
	currentMsg.AddMetadata("new_feature_flag", true)
	currentMsg.AddMetadata("compatibility_mode", "backward")

	err = serializer.SetEnvelopeHash(currentMsg)
	if err != nil {
		fmt.Printf("Failed to set envelope hash: %v\n", err)
		return
	}

	currentData, _ := serializer.Serialize(currentMsg)
	fmt.Printf("Current message: %s\n", string(currentData))

	// Verify it can be deserialized and validated
	deserializedCurrent, err := serializer.Deserialize(currentData)
	if err != nil {
		fmt.Printf("Failed to deserialize current message: %v\n", err)
		return
	}

	err = serializer.ValidateHash(deserializedCurrent)
	if err != nil {
		fmt.Printf("Hash validation failed: %v\n", err)
		return
	}

	fmt.Printf("Schema evolution test passed: message with new metadata fields works correctly\n")

	fmt.Println("\n=== Manual Test Complete ===")
}

// PrettyPrintMessage formats a message for human-readable output
func PrettyPrintMessage(msg *Message) string {
	data, err := json.MarshalIndent(msg, "", "  ")
	if err != nil {
		return fmt.Sprintf("Error formatting message: %v", err)
	}
	return string(data)
}
