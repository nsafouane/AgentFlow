package security

import (
	"testing"
)

// TestManualMultiTenancyDemo runs the multi-tenancy demonstration
// This test demonstrates the complete multi-tenancy enforcement system
func TestManualMultiTenancyDemo(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping manual multi-tenancy demo in short mode")
	}

	// Run the demonstration
	DemoMultiTenancyEnforcement()
}
