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
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/agentflow/agentflow/internal/logging"
)

// EnvironmentProvider implements SecretsProvider using environment variables
type EnvironmentProvider struct {
	prefix string // Prefix for environment variables (e.g., "AF_SECRET_")
	logger logging.Logger
}

// NewEnvironmentProvider creates a new environment-based secrets provider
func NewEnvironmentProvider(prefix string) *EnvironmentProvider {
	if prefix == "" {
		prefix = "AF_SECRET_"
	}

	return &EnvironmentProvider{
		prefix: prefix,
		logger: logging.NewLogger().WithFields(logging.String("component", "secrets.environment")),
	}
}

// GetSecret retrieves a secret from environment variables
func (p *EnvironmentProvider) GetSecret(ctx context.Context, key string) (string, error) {
	if err := ValidateKey(key); err != nil {
		p.logger.WithTrace(ctx).Error("Invalid secret key", err,
			logging.String("key", MaskSecret(key)))
		return "", err
	}

	envKey := p.prefix + strings.ToUpper(key)
	value := os.Getenv(envKey)

	if value == "" {
		p.logger.WithTrace(ctx).Warn("Secret not found in environment",
			logging.String("key", key),
			logging.String("env_key", envKey))
		return "", fmt.Errorf("%w: %s", ErrSecretNotFound, key)
	}

	p.logger.WithTrace(ctx).Info("Secret retrieved from environment",
		logging.String("key", key),
		logging.String("env_key", envKey),
		logging.Int("value_length", len(value)))

	return value, nil
}

// SetSecret sets an environment variable (limited functionality)
func (p *EnvironmentProvider) SetSecret(ctx context.Context, key, value string) error {
	if err := ValidateKey(key); err != nil {
		p.logger.WithTrace(ctx).Error("Invalid secret key for set operation", err,
			logging.String("key", MaskSecret(key)))
		return err
	}

	envKey := p.prefix + strings.ToUpper(key)

	// Set environment variable for current process
	if err := os.Setenv(envKey, value); err != nil {
		p.logger.WithTrace(ctx).Error("Failed to set environment variable", err,
			logging.String("key", key),
			logging.String("env_key", envKey))
		return fmt.Errorf("failed to set environment variable: %w", err)
	}

	p.logger.WithTrace(ctx).Info("Secret set in environment",
		logging.String("key", key),
		logging.String("env_key", envKey),
		logging.Int("value_length", len(value)),
		logging.String("timestamp", time.Now().UTC().Format(time.RFC3339)))

	return nil
}

// DeleteSecret removes an environment variable
func (p *EnvironmentProvider) DeleteSecret(ctx context.Context, key string) error {
	if err := ValidateKey(key); err != nil {
		p.logger.WithTrace(ctx).Error("Invalid secret key for delete operation", err,
			logging.String("key", MaskSecret(key)))
		return err
	}

	envKey := p.prefix + strings.ToUpper(key)

	// Check if secret exists
	if os.Getenv(envKey) == "" {
		p.logger.WithTrace(ctx).Warn("Attempted to delete non-existent secret",
			logging.String("key", key),
			logging.String("env_key", envKey))
		return fmt.Errorf("%w: %s", ErrSecretNotFound, key)
	}

	// Unset environment variable
	if err := os.Unsetenv(envKey); err != nil {
		p.logger.WithTrace(ctx).Error("Failed to unset environment variable", err,
			logging.String("key", key),
			logging.String("env_key", envKey))
		return fmt.Errorf("failed to unset environment variable: %w", err)
	}

	p.logger.WithTrace(ctx).Info("Secret deleted from environment",
		logging.String("key", key),
		logging.String("env_key", envKey),
		logging.String("timestamp", time.Now().UTC().Format(time.RFC3339)))

	return nil
}

// ListSecrets returns all environment variables with the configured prefix
func (p *EnvironmentProvider) ListSecrets(ctx context.Context) ([]string, error) {
	var secrets []string

	for _, env := range os.Environ() {
		if strings.HasPrefix(env, p.prefix) {
			// Extract key part (remove prefix and value)
			parts := strings.SplitN(env, "=", 2)
			if len(parts) >= 1 {
				key := strings.TrimPrefix(parts[0], p.prefix)
				key = strings.ToLower(key)
				secrets = append(secrets, key)
			}
		}
	}

	p.logger.WithTrace(ctx).Info("Listed secrets from environment",
		logging.Int("count", len(secrets)),
		logging.String("prefix", p.prefix))

	return secrets, nil
}

// Rotate generates a new value for an existing secret (not supported for environment)
func (p *EnvironmentProvider) Rotate(ctx context.Context, key string) error {
	p.logger.WithTrace(ctx).Warn("Secret rotation not supported for environment provider",
		logging.String("key", key))

	return fmt.Errorf("secret rotation not supported for environment provider")
}
