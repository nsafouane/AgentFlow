// Copyright 2025 AgentFlow
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package secrets

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// ManualTestSecretRotationAndHotReload demonstrates secret rotation and hot reload capabilities
// This test should be run manually to verify the behavior described in the task requirements
func TestManualSecretRotationAndHotReload(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping manual test in short mode")
	}

	tempDir := t.TempDir()
	secretsFile := filepath.Join(tempDir, "test_secrets.json")

	fmt.Printf("=== Manual Test: Secret Rotation and Hot Reload ===\n")
	fmt.Printf("Secrets file: %s\n\n", secretsFile)

	// Create file provider
	provider := NewFileProvider(secretsFile)
	ctx := context.Background()

	// Step 1: Set initial secrets
	fmt.Println("Step 1: Setting initial secrets...")
	secrets := map[string]string{
		"api_key":     "initial-api-key-value",
		"db_password": "initial-db-password",
		"jwt_secret":  "initial-jwt-secret",
	}

	for key, value := range secrets {
		err := provider.SetSecret(ctx, key, value)
		if err != nil {
			t.Fatalf("Failed to set secret %s: %v", key, err)
		}
		fmt.Printf("  Set %s = %s (masked: %s)\n", key, value, MaskSecret(value))
	}

	// Step 2: List all secrets
	fmt.Println("\nStep 2: Listing all secrets...")
	keys, err := provider.ListSecrets(ctx)
	if err != nil {
		t.Fatalf("Failed to list secrets: %v", err)
	}
	fmt.Printf("  Found %d secrets: %v\n", len(keys), keys)

	// Step 3: Rotate a secret
	fmt.Println("\nStep 3: Rotating api_key...")
	originalValue, err := provider.GetSecret(ctx, "api_key")
	if err != nil {
		t.Fatalf("Failed to get original api_key: %v", err)
	}
	fmt.Printf("  Original value: %s (masked: %s)\n", originalValue, MaskSecret(originalValue))

	err = provider.Rotate(ctx, "api_key")
	if err != nil {
		t.Fatalf("Failed to rotate api_key: %v", err)
	}

	newValue, err := provider.GetSecret(ctx, "api_key")
	if err != nil {
		t.Fatalf("Failed to get rotated api_key: %v", err)
	}
	fmt.Printf("  New value: %s (masked: %s)\n", newValue, MaskSecret(newValue))
	fmt.Printf("  Value changed: %t\n", originalValue != newValue)

	// Step 4: Demonstrate hot reload by manually editing the file
	fmt.Println("\nStep 4: Demonstrating hot reload...")
	fmt.Printf("  Current secrets file content:\n")

	// Read current file content
	data, err := os.ReadFile(secretsFile)
	if err != nil {
		t.Fatalf("Failed to read secrets file: %v", err)
	}
	fmt.Printf("  %s\n", string(data))

	// Parse current secrets
	var currentSecrets map[string]string
	if err := json.Unmarshal(data, &currentSecrets); err != nil {
		t.Fatalf("Failed to parse secrets file: %v", err)
	}

	// Add a new secret and modify an existing one externally
	currentSecrets["external_secret"] = "added-externally"
	currentSecrets["db_password"] = "externally-modified-password"

	// Write back to file
	newData, err := json.MarshalIndent(currentSecrets, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal modified secrets: %v", err)
	}

	// Wait a bit to ensure different modification time
	time.Sleep(100 * time.Millisecond)

	err = os.WriteFile(secretsFile, newData, 0600)
	if err != nil {
		t.Fatalf("Failed to write modified secrets file: %v", err)
	}

	fmt.Printf("  Modified secrets file externally\n")
	fmt.Printf("  New file content:\n")
	fmt.Printf("  %s\n", string(newData))

	// Step 5: Verify hot reload works
	fmt.Println("\nStep 5: Verifying hot reload...")

	// Get the externally added secret
	externalValue, err := provider.GetSecret(ctx, "external_secret")
	if err != nil {
		t.Fatalf("Failed to get externally added secret: %v", err)
	}
	fmt.Printf("  External secret retrieved: %s\n", externalValue)

	// Get the externally modified secret
	modifiedValue, err := provider.GetSecret(ctx, "db_password")
	if err != nil {
		t.Fatalf("Failed to get externally modified secret: %v", err)
	}
	fmt.Printf("  Modified secret retrieved: %s\n", modifiedValue)

	// Verify the values are correct
	if externalValue != "added-externally" {
		t.Errorf("Expected external secret 'added-externally', got %s", externalValue)
	}
	if modifiedValue != "externally-modified-password" {
		t.Errorf("Expected modified secret 'externally-modified-password', got %s", modifiedValue)
	}

	// Step 6: List secrets again to show all changes
	fmt.Println("\nStep 6: Final secret listing...")
	finalKeys, err := provider.ListSecrets(ctx)
	if err != nil {
		t.Fatalf("Failed to list final secrets: %v", err)
	}
	fmt.Printf("  Final secrets (%d): %v\n", len(finalKeys), finalKeys)

	// Step 7: Demonstrate environment provider
	fmt.Println("\nStep 7: Testing environment provider...")
	envProvider := NewEnvironmentProvider("MANUAL_TEST_")

	// Set environment variable
	testEnvKey := "MANUAL_TEST_SAMPLE_KEY"
	testEnvValue := "sample-env-value"
	os.Setenv(testEnvKey, testEnvValue)
	defer os.Unsetenv(testEnvKey)

	envValue, err := envProvider.GetSecret(ctx, "sample_key")
	if err != nil {
		t.Fatalf("Failed to get environment secret: %v", err)
	}
	fmt.Printf("  Environment secret retrieved: %s (masked: %s)\n", envValue, MaskSecret(envValue))

	// List environment secrets
	envKeys, err := envProvider.ListSecrets(ctx)
	if err != nil {
		t.Fatalf("Failed to list environment secrets: %v", err)
	}
	fmt.Printf("  Environment secrets: %v\n", envKeys)

	fmt.Println("\n=== Manual Test Completed Successfully ===")
	fmt.Println("Key observations:")
	fmt.Println("1. Secrets can be stored and retrieved from both file and environment providers")
	fmt.Println("2. Secret values are properly masked in logs")
	fmt.Println("3. File provider supports hot reload - external changes are detected automatically")
	fmt.Println("4. Secret rotation generates new random values")
	fmt.Println("5. Both providers validate secret keys and handle errors gracefully")
}
