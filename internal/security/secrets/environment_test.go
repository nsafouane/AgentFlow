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
	"os"
	"strings"
	"testing"
)

func TestEnvironmentProvider_GetSecret(t *testing.T) {
	provider := NewEnvironmentProvider("TEST_SECRET_")
	ctx := context.Background()

	// Set up test environment variable
	testKey := "api_key"
	testValue := "test-secret-value"
	envKey := "TEST_SECRET_API_KEY"

	// Clean up after test
	defer os.Unsetenv(envKey)

	t.Run("secret not found", func(t *testing.T) {
		_, err := provider.GetSecret(ctx, testKey)
		if err == nil {
			t.Error("Expected error for non-existent secret")
		}
		if !strings.Contains(err.Error(), "secret not found") {
			t.Errorf("Expected 'secret not found' error, got: %v", err)
		}
	})

	t.Run("secret found", func(t *testing.T) {
		os.Setenv(envKey, testValue)

		value, err := provider.GetSecret(ctx, testKey)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if value != testValue {
			t.Errorf("Expected %q, got %q", testValue, value)
		}
	})

	t.Run("invalid key", func(t *testing.T) {
		_, err := provider.GetSecret(ctx, "invalid@key")
		if err == nil {
			t.Error("Expected error for invalid key")
		}
		if !strings.Contains(err.Error(), "invalid secret key") {
			t.Errorf("Expected 'invalid secret key' error, got: %v", err)
		}
	})
}

func TestEnvironmentProvider_SetSecret(t *testing.T) {
	provider := NewEnvironmentProvider("TEST_SECRET_")
	ctx := context.Background()

	testKey := "new_key"
	testValue := "new-secret-value"
	envKey := "TEST_SECRET_NEW_KEY"

	// Clean up after test
	defer os.Unsetenv(envKey)

	t.Run("set new secret", func(t *testing.T) {
		err := provider.SetSecret(ctx, testKey, testValue)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		// Verify it was set
		value := os.Getenv(envKey)
		if value != testValue {
			t.Errorf("Expected %q, got %q", testValue, value)
		}
	})

	t.Run("invalid key", func(t *testing.T) {
		err := provider.SetSecret(ctx, "invalid@key", testValue)
		if err == nil {
			t.Error("Expected error for invalid key")
		}
	})
}

func TestEnvironmentProvider_DeleteSecret(t *testing.T) {
	provider := NewEnvironmentProvider("TEST_SECRET_")
	ctx := context.Background()

	testKey := "delete_key"
	testValue := "delete-secret-value"
	envKey := "TEST_SECRET_DELETE_KEY"

	t.Run("delete non-existent secret", func(t *testing.T) {
		err := provider.DeleteSecret(ctx, testKey)
		if err == nil {
			t.Error("Expected error for non-existent secret")
		}
		if !strings.Contains(err.Error(), "secret not found") {
			t.Errorf("Expected 'secret not found' error, got: %v", err)
		}
	})

	t.Run("delete existing secret", func(t *testing.T) {
		// Set up secret first
		os.Setenv(envKey, testValue)

		err := provider.DeleteSecret(ctx, testKey)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		// Verify it was deleted
		value := os.Getenv(envKey)
		if value != "" {
			t.Errorf("Expected empty value, got %q", value)
		}
	})
}

func TestEnvironmentProvider_ListSecrets(t *testing.T) {
	provider := NewEnvironmentProvider("TEST_LIST_")
	ctx := context.Background()

	// Set up test secrets
	secrets := map[string]string{
		"TEST_LIST_SECRET1": "value1",
		"TEST_LIST_SECRET2": "value2",
		"TEST_LIST_SECRET3": "value3",
	}

	// Clean up after test
	defer func() {
		for key := range secrets {
			os.Unsetenv(key)
		}
	}()

	// Set environment variables
	for key, value := range secrets {
		os.Setenv(key, value)
	}

	keys, err := provider.ListSecrets(ctx)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(keys) != 3 {
		t.Errorf("Expected 3 secrets, got %d", len(keys))
	}

	// Check that all expected keys are present (converted to lowercase)
	expectedKeys := []string{"secret1", "secret2", "secret3"}
	for _, expected := range expectedKeys {
		found := false
		for _, key := range keys {
			if key == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected key %q not found in results: %v", expected, keys)
		}
	}
}

func TestEnvironmentProvider_Rotate(t *testing.T) {
	provider := NewEnvironmentProvider("TEST_SECRET_")
	ctx := context.Background()

	err := provider.Rotate(ctx, "any_key")
	if err == nil {
		t.Error("Expected error for unsupported rotation")
	}
	if !strings.Contains(err.Error(), "not supported") {
		t.Errorf("Expected 'not supported' error, got: %v", err)
	}
}

func TestEnvironmentProvider_DefaultPrefix(t *testing.T) {
	provider := NewEnvironmentProvider("")

	if provider.prefix != "AF_SECRET_" {
		t.Errorf("Expected default prefix 'AF_SECRET_', got %q", provider.prefix)
	}
}
