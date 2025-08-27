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
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestFileProvider_GetSecret(t *testing.T) {
	tempDir := t.TempDir()
	secretsFile := filepath.Join(tempDir, "secrets.json")

	provider := NewFileProvider(secretsFile)
	ctx := context.Background()

	t.Run("secret not found in empty file", func(t *testing.T) {
		_, err := provider.GetSecret(ctx, "nonexistent")
		if err == nil {
			t.Error("Expected error for non-existent secret")
		}
		if !strings.Contains(err.Error(), "secret not found") {
			t.Errorf("Expected 'secret not found' error, got: %v", err)
		}
	})

	t.Run("secret found", func(t *testing.T) {
		// Set up secret first
		err := provider.SetSecret(ctx, "test_key", "test_value")
		if err != nil {
			t.Fatalf("Failed to set secret: %v", err)
		}

		value, err := provider.GetSecret(ctx, "test_key")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if value != "test_value" {
			t.Errorf("Expected 'test_value', got %q", value)
		}
	})

	t.Run("invalid key", func(t *testing.T) {
		_, err := provider.GetSecret(ctx, "invalid@key")
		if err == nil {
			t.Error("Expected error for invalid key")
		}
	})
}

func TestFileProvider_SetSecret(t *testing.T) {
	tempDir := t.TempDir()
	secretsFile := filepath.Join(tempDir, "secrets.json")

	provider := NewFileProvider(secretsFile)
	ctx := context.Background()

	t.Run("set new secret", func(t *testing.T) {
		err := provider.SetSecret(ctx, "new_key", "new_value")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		// Verify file was created and contains the secret
		data, err := os.ReadFile(secretsFile)
		if err != nil {
			t.Fatalf("Failed to read secrets file: %v", err)
		}

		var secrets map[string]string
		if err := json.Unmarshal(data, &secrets); err != nil {
			t.Fatalf("Failed to parse secrets file: %v", err)
		}

		if secrets["new_key"] != "new_value" {
			t.Errorf("Expected 'new_value', got %q", secrets["new_key"])
		}
	})

	t.Run("update existing secret", func(t *testing.T) {
		err := provider.SetSecret(ctx, "new_key", "updated_value")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		value, err := provider.GetSecret(ctx, "new_key")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if value != "updated_value" {
			t.Errorf("Expected 'updated_value', got %q", value)
		}
	})
}

func TestFileProvider_DeleteSecret(t *testing.T) {
	tempDir := t.TempDir()
	secretsFile := filepath.Join(tempDir, "secrets.json")

	provider := NewFileProvider(secretsFile)
	ctx := context.Background()

	t.Run("delete non-existent secret", func(t *testing.T) {
		err := provider.DeleteSecret(ctx, "nonexistent")
		if err == nil {
			t.Error("Expected error for non-existent secret")
		}
		if !strings.Contains(err.Error(), "secret not found") {
			t.Errorf("Expected 'secret not found' error, got: %v", err)
		}
	})

	t.Run("delete existing secret", func(t *testing.T) {
		// Set up secret first
		err := provider.SetSecret(ctx, "delete_me", "delete_value")
		if err != nil {
			t.Fatalf("Failed to set secret: %v", err)
		}

		// Delete the secret
		err = provider.DeleteSecret(ctx, "delete_me")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		// Verify it's gone
		_, err = provider.GetSecret(ctx, "delete_me")
		if err == nil {
			t.Error("Expected error for deleted secret")
		}
	})
}

func TestFileProvider_ListSecrets(t *testing.T) {
	tempDir := t.TempDir()
	secretsFile := filepath.Join(tempDir, "secrets.json")

	provider := NewFileProvider(secretsFile)
	ctx := context.Background()

	t.Run("list empty secrets", func(t *testing.T) {
		keys, err := provider.ListSecrets(ctx)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if len(keys) != 0 {
			t.Errorf("Expected 0 secrets, got %d", len(keys))
		}
	})

	t.Run("list multiple secrets", func(t *testing.T) {
		// Set up multiple secrets
		secrets := map[string]string{
			"secret1": "value1",
			"secret2": "value2",
			"secret3": "value3",
		}

		for key, value := range secrets {
			err := provider.SetSecret(ctx, key, value)
			if err != nil {
				t.Fatalf("Failed to set secret %s: %v", key, err)
			}
		}

		keys, err := provider.ListSecrets(ctx)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		if len(keys) != 3 {
			t.Errorf("Expected 3 secrets, got %d", len(keys))
		}

		// Check that all expected keys are present
		for expectedKey := range secrets {
			found := false
			for _, key := range keys {
				if key == expectedKey {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Expected key %q not found in results: %v", expectedKey, keys)
			}
		}
	})
}

func TestFileProvider_Rotate(t *testing.T) {
	tempDir := t.TempDir()
	secretsFile := filepath.Join(tempDir, "secrets.json")

	provider := NewFileProvider(secretsFile)
	ctx := context.Background()

	t.Run("rotate non-existent secret", func(t *testing.T) {
		err := provider.Rotate(ctx, "nonexistent")
		if err == nil {
			t.Error("Expected error for non-existent secret")
		}
		if !strings.Contains(err.Error(), "secret not found") {
			t.Errorf("Expected 'secret not found' error, got: %v", err)
		}
	})

	t.Run("rotate existing secret", func(t *testing.T) {
		// Set up secret first
		originalValue := "original_value"
		err := provider.SetSecret(ctx, "rotate_me", originalValue)
		if err != nil {
			t.Fatalf("Failed to set secret: %v", err)
		}

		// Rotate the secret
		err = provider.Rotate(ctx, "rotate_me")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		// Verify the value changed
		newValue, err := provider.GetSecret(ctx, "rotate_me")
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}

		if newValue == originalValue {
			t.Error("Secret value should have changed after rotation")
		}

		if len(newValue) != 64 { // 32 bytes = 64 hex characters
			t.Errorf("Expected rotated value to be 64 characters, got %d", len(newValue))
		}

		// Verify it's valid hex
		for _, r := range newValue {
			if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) {
				t.Errorf("Rotated value contains non-hex character: %c", r)
				break
			}
		}
	})
}

func TestFileProvider_HotReload(t *testing.T) {
	tempDir := t.TempDir()
	secretsFile := filepath.Join(tempDir, "secrets.json")

	provider := NewFileProvider(secretsFile)
	ctx := context.Background()

	// Set initial secret
	err := provider.SetSecret(ctx, "test_key", "initial_value")
	if err != nil {
		t.Fatalf("Failed to set initial secret: %v", err)
	}

	// Manually modify the file to simulate external change
	secrets := map[string]string{
		"test_key":   "external_value",
		"new_secret": "new_value",
	}

	data, err := json.MarshalIndent(secrets, "", "  ")
	if err != nil {
		t.Fatalf("Failed to marshal secrets: %v", err)
	}

	// Wait a bit to ensure different modification time
	time.Sleep(10 * time.Millisecond)

	err = os.WriteFile(secretsFile, data, 0600)
	if err != nil {
		t.Fatalf("Failed to write secrets file: %v", err)
	}

	// Get secret should reload and return the new value
	value, err := provider.GetSecret(ctx, "test_key")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if value != "external_value" {
		t.Errorf("Expected 'external_value' (hot reloaded), got %q", value)
	}

	// New secret should also be available
	value, err = provider.GetSecret(ctx, "new_secret")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if value != "new_value" {
		t.Errorf("Expected 'new_value', got %q", value)
	}
}

func TestFileProvider_DefaultPath(t *testing.T) {
	provider := NewFileProvider("")

	if provider.configPath != "secrets.json" {
		t.Errorf("Expected default path 'secrets.json', got %q", provider.configPath)
	}
}

func TestFileProvider_FilePermissions(t *testing.T) {
	tempDir := t.TempDir()
	secretsFile := filepath.Join(tempDir, "secrets.json")

	provider := NewFileProvider(secretsFile)
	ctx := context.Background()

	// Set a secret to create the file
	err := provider.SetSecret(ctx, "test_key", "test_value")
	if err != nil {
		t.Fatalf("Failed to set secret: %v", err)
	}

	// Check file permissions
	info, err := os.Stat(secretsFile)
	if err != nil {
		t.Fatalf("Failed to stat secrets file: %v", err)
	}

	mode := info.Mode()
	if mode.Perm() != 0600 {
		t.Errorf("Expected file permissions 0600, got %o", mode.Perm())
	}
}
