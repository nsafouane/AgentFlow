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
	"strings"
	"testing"
)

func TestMaskSecret(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
		{
			name:     "single character",
			input:    "a",
			expected: "*",
		},
		{
			name:     "two characters",
			input:    "ab",
			expected: "**",
		},
		{
			name:     "three characters",
			input:    "abc",
			expected: "***",
		},
		{
			name:     "four characters",
			input:    "abcd",
			expected: "****",
		},
		{
			name:     "five characters",
			input:    "abcde",
			expected: "ab*de",
		},
		{
			name:     "long secret",
			input:    "this-is-a-very-long-secret-value",
			expected: "th****************************ue",
		},
		{
			name:     "typical API key",
			input:    "sk-1234567890abcdef",
			expected: "sk***************ef",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MaskSecret(tt.input)
			if result != tt.expected {
				t.Errorf("MaskSecret(%q) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestValidateKey(t *testing.T) {
	tests := []struct {
		name    string
		key     string
		wantErr bool
	}{
		{
			name:    "valid simple key",
			key:     "api_key",
			wantErr: false,
		},
		{
			name:    "valid key with hyphens",
			key:     "database-password",
			wantErr: false,
		},
		{
			name:    "valid key with numbers",
			key:     "secret123",
			wantErr: false,
		},
		{
			name:    "valid mixed case",
			key:     "MySecret_Key-123",
			wantErr: false,
		},
		{
			name:    "empty key",
			key:     "",
			wantErr: true,
		},
		{
			name:    "key with spaces",
			key:     "api key",
			wantErr: true,
		},
		{
			name:    "key with special characters",
			key:     "api@key",
			wantErr: true,
		},
		{
			name:    "key with dots",
			key:     "api.key",
			wantErr: true,
		},
		{
			name:    "key too long",
			key:     strings.Repeat("a", 256),
			wantErr: true,
		},
		{
			name:    "key at max length",
			key:     strings.Repeat("a", 255),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateKey(tt.key)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateKey(%q) error = %v, wantErr %v", tt.key, err, tt.wantErr)
			}
		})
	}
}
