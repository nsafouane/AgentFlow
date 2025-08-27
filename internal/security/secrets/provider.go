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
	"errors"
	"fmt"
	"strings"
)

// SecretsProvider defines the interface for managing secrets across different providers
type SecretsProvider interface {
	// GetSecret retrieves a secret value by key
	GetSecret(ctx context.Context, key string) (string, error)

	// SetSecret stores a secret value with the given key
	SetSecret(ctx context.Context, key, value string) error

	// DeleteSecret removes a secret by key
	DeleteSecret(ctx context.Context, key string) error

	// ListSecrets returns a list of available secret keys
	ListSecrets(ctx context.Context) ([]string, error)

	// Rotate generates a new value for an existing secret
	Rotate(ctx context.Context, key string) error
}

// Common errors
var (
	ErrSecretNotFound      = errors.New("secret not found")
	ErrSecretExists        = errors.New("secret already exists")
	ErrInvalidKey          = errors.New("invalid secret key")
	ErrPermissionDenied    = errors.New("permission denied")
	ErrProviderUnavailable = errors.New("secrets provider unavailable")
)

// MaskSecret masks sensitive values for logging and debug output
func MaskSecret(value string) string {
	if len(value) == 0 {
		return ""
	}
	if len(value) <= 4 {
		return strings.Repeat("*", len(value))
	}
	return value[:2] + strings.Repeat("*", len(value)-4) + value[len(value)-2:]
}

// ValidateKey validates secret key format and permissions
func ValidateKey(key string) error {
	if key == "" {
		return fmt.Errorf("%w: key cannot be empty", ErrInvalidKey)
	}

	// Key must be alphanumeric with underscores and hyphens
	for _, r := range key {
		if !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') ||
			(r >= '0' && r <= '9') || r == '_' || r == '-') {
			return fmt.Errorf("%w: key contains invalid characters", ErrInvalidKey)
		}
	}

	// Key length limits
	if len(key) > 255 {
		return fmt.Errorf("%w: key too long (max 255 characters)", ErrInvalidKey)
	}

	return nil
}
