package agent

import (
	"testing"
)

// TestAgentInterfaces tests that agent interfaces are properly defined
func TestAgentInterfaces(t *testing.T) {
	// Test Status constants
	if StatusSuccess != "success" {
		t.Errorf("Expected StatusSuccess to be 'success', got %s", StatusSuccess)
	}

	if StatusError != "error" {
		t.Errorf("Expected StatusError to be 'error', got %s", StatusError)
	}

	if StatusPending != "pending" {
		t.Errorf("Expected StatusPending to be 'pending', got %s", StatusPending)
	}

	// Test Input and Output structures
	input := Input{
		Data: map[string]interface{}{
			"test": "value",
		},
	}

	if input.Data["test"] != "value" {
		t.Error("Input data not properly set")
	}

	output := Output{
		Data: map[string]interface{}{
			"result": "success",
		},
		Status: StatusSuccess,
	}

	if output.Status != StatusSuccess {
		t.Error("Output status not properly set")
	}

	t.Log("Agent interfaces test completed")
}
