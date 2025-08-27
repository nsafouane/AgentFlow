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
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/agentflow/agentflow/internal/logging"
)

// FileProvider implements SecretsProvider using a JSON file
type FileProvider struct {
	configPath string
	secrets    map[string]string
	lastMod    time.Time
	mu         sync.RWMutex
	logger     logging.Logger
}

// NewFileProvider creates a new file-based secrets provider
func NewFileProvider(configPath string) *FileProvider {
	if configPath == "" {
		configPath = "secrets.json"
	}

	provider := &FileProvider{
		configPath: configPath,
		secrets:    make(map[string]string),
		logger:     logging.NewLogger().WithFields(logging.String("component", "secrets.file")),
	}

	// Load initial secrets
	if err := provider.loadSecrets(); err != nil {
		provider.logger.Warn("Failed to load initial secrets file",
			logging.String("path", configPath),
			logging.Any("error", err))
	}

	return provider
}

// loadSecrets loads secrets from the file and checks for modifications
func (p *FileProvider) loadSecrets() error {
	p.mu.Lock()
	defer p.mu.Unlock()
	return p.loadSecretsLocked()
}

// loadSecretsLocked loads secrets from the file (must be called with lock held)
func (p *FileProvider) loadSecretsLocked() error {
	// Check if file exists
	info, err := os.Stat(p.configPath)
	if os.IsNotExist(err) {
		// Create empty secrets file
		p.secrets = make(map[string]string)
		return p.saveSecretsLocked()
	}
	if err != nil {
		return fmt.Errorf("failed to stat secrets file: %w", err)
	}

	// Check if file has been modified
	if !info.ModTime().After(p.lastMod) {
		return nil // No changes
	}

	// Read and parse file
	data, err := os.ReadFile(p.configPath)
	if err != nil {
		return fmt.Errorf("failed to read secrets file: %w", err)
	}

	var secrets map[string]string
	if len(data) > 0 {
		if err := json.Unmarshal(data, &secrets); err != nil {
			return fmt.Errorf("failed to parse secrets file: %w", err)
		}
	} else {
		secrets = make(map[string]string)
	}

	p.secrets = secrets
	p.lastMod = info.ModTime()

	p.logger.Info("Loaded secrets from file",
		logging.String("path", p.configPath),
		logging.Int("count", len(secrets)),
		logging.Any("last_modified", p.lastMod))

	return nil
}

// saveSecretsLocked saves secrets to file (must be called with lock held)
func (p *FileProvider) saveSecretsLocked() error {
	// Ensure directory exists
	dir := filepath.Dir(p.configPath)
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("failed to create secrets directory: %w", err)
	}

	// Marshal secrets to JSON
	data, err := json.MarshalIndent(p.secrets, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal secrets: %w", err)
	}

	// Write to temporary file first
	tempPath := p.configPath + ".tmp"
	if err := os.WriteFile(tempPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write temporary secrets file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tempPath, p.configPath); err != nil {
		os.Remove(tempPath) // Clean up on failure
		return fmt.Errorf("failed to rename secrets file: %w", err)
	}

	// Update modification time
	if info, err := os.Stat(p.configPath); err == nil {
		p.lastMod = info.ModTime()
	}

	return nil
}

// GetSecret retrieves a secret from the file
func (p *FileProvider) GetSecret(ctx context.Context, key string) (string, error) {
	if err := ValidateKey(key); err != nil {
		p.logger.WithTrace(ctx).Error("Invalid secret key", err,
			logging.String("key", MaskSecret(key)))
		return "", err
	}

	// Reload secrets to check for updates
	if err := p.loadSecrets(); err != nil {
		p.logger.WithTrace(ctx).Error("Failed to reload secrets", err)
		// Continue with cached secrets
	}

	p.mu.RLock()
	value, exists := p.secrets[key]
	p.mu.RUnlock()

	if !exists {
		p.logger.WithTrace(ctx).Warn("Secret not found in file",
			logging.String("key", key),
			logging.String("path", p.configPath))
		return "", fmt.Errorf("%w: %s", ErrSecretNotFound, key)
	}

	p.logger.WithTrace(ctx).Info("Secret retrieved from file",
		logging.String("key", key),
		logging.String("path", p.configPath),
		logging.Int("value_length", len(value)))

	return value, nil
}

// SetSecret stores a secret in the file
func (p *FileProvider) SetSecret(ctx context.Context, key, value string) error {
	if err := ValidateKey(key); err != nil {
		p.logger.WithTrace(ctx).Error("Invalid secret key for set operation", err,
			logging.String("key", MaskSecret(key)))
		return err
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Reload to get latest state first (with lock held)
	if err := p.loadSecretsLocked(); err != nil {
		p.logger.WithTrace(ctx).Error("Failed to reload secrets before set", err)
	}

	p.secrets[key] = value

	if err := p.saveSecretsLocked(); err != nil {
		p.logger.WithTrace(ctx).Error("Failed to save secret to file", err,
			logging.String("key", key))
		return fmt.Errorf("failed to save secret: %w", err)
	}

	p.logger.WithTrace(ctx).Info("Secret set in file",
		logging.String("key", key),
		logging.String("path", p.configPath),
		logging.Int("value_length", len(value)),
		logging.Any("timestamp", time.Now().UTC()))

	return nil
}

// DeleteSecret removes a secret from the file
func (p *FileProvider) DeleteSecret(ctx context.Context, key string) error {
	if err := ValidateKey(key); err != nil {
		p.logger.WithTrace(ctx).Error("Invalid secret key for delete operation", err,
			logging.String("key", MaskSecret(key)))
		return err
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Reload to get latest state first (with lock held)
	if err := p.loadSecretsLocked(); err != nil {
		p.logger.WithTrace(ctx).Error("Failed to reload secrets before delete", err)
	}

	if _, exists := p.secrets[key]; !exists {
		p.logger.WithTrace(ctx).Warn("Attempted to delete non-existent secret",
			logging.String("key", key),
			logging.String("path", p.configPath))
		return fmt.Errorf("%w: %s", ErrSecretNotFound, key)
	}

	delete(p.secrets, key)

	if err := p.saveSecretsLocked(); err != nil {
		p.logger.WithTrace(ctx).Error("Failed to save secrets after delete", err,
			logging.String("key", key))
		return fmt.Errorf("failed to save secrets: %w", err)
	}

	p.logger.WithTrace(ctx).Info("Secret deleted from file",
		logging.String("key", key),
		logging.String("path", p.configPath),
		logging.Any("timestamp", time.Now().UTC()))

	return nil
}

// ListSecrets returns all secret keys from the file
func (p *FileProvider) ListSecrets(ctx context.Context) ([]string, error) {
	// Reload secrets to check for updates
	if err := p.loadSecrets(); err != nil {
		p.logger.WithTrace(ctx).Error("Failed to reload secrets for list", err)
		// Continue with cached secrets
	}

	p.mu.RLock()
	keys := make([]string, 0, len(p.secrets))
	for key := range p.secrets {
		keys = append(keys, key)
	}
	p.mu.RUnlock()

	p.logger.WithTrace(ctx).Info("Listed secrets from file",
		logging.Int("count", len(keys)),
		logging.String("path", p.configPath))

	return keys, nil
}

// Rotate generates a new random value for an existing secret
func (p *FileProvider) Rotate(ctx context.Context, key string) error {
	if err := ValidateKey(key); err != nil {
		p.logger.WithTrace(ctx).Error("Invalid secret key for rotation", err,
			logging.String("key", MaskSecret(key)))
		return err
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	// Reload to get latest state first (with lock held)
	if err := p.loadSecretsLocked(); err != nil {
		p.logger.WithTrace(ctx).Error("Failed to reload secrets before rotation", err)
	}

	if _, exists := p.secrets[key]; !exists {
		p.logger.WithTrace(ctx).Warn("Attempted to rotate non-existent secret",
			logging.String("key", key),
			logging.String("path", p.configPath))
		return fmt.Errorf("%w: %s", ErrSecretNotFound, key)
	}

	// Generate new random value (32 bytes = 64 hex characters)
	randomBytes := make([]byte, 32)
	if _, err := rand.Read(randomBytes); err != nil {
		p.logger.WithTrace(ctx).Error("Failed to generate random value for rotation", err,
			logging.String("key", key))
		return fmt.Errorf("failed to generate random value: %w", err)
	}

	newValue := hex.EncodeToString(randomBytes)
	p.secrets[key] = newValue

	if err := p.saveSecretsLocked(); err != nil {
		p.logger.WithTrace(ctx).Error("Failed to save rotated secret", err,
			logging.String("key", key))
		return fmt.Errorf("failed to save rotated secret: %w", err)
	}

	p.logger.WithTrace(ctx).Info("Secret rotated in file",
		logging.String("key", key),
		logging.String("path", p.configPath),
		logging.Int("new_value_length", len(newValue)),
		logging.Any("timestamp", time.Now().UTC()))

	return nil
}
