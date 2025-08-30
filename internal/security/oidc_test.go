package security

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOIDCProvider(t *testing.T) {
	t.Run("NewOIDCProvider_Disabled", func(t *testing.T) {
		config := &AuthConfig{
			OIDCEnabled: false,
		}

		_, err := NewOIDCProvider(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "OIDC is not enabled")
	})

	t.Run("NewOIDCProvider_MissingIssuer", func(t *testing.T) {
		config := &AuthConfig{
			OIDCEnabled: true,
		}

		_, err := NewOIDCProvider(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "OIDC issuer is required")
	})

	t.Run("NewOIDCProvider_InvalidIssuer", func(t *testing.T) {
		config := &AuthConfig{
			OIDCEnabled: true,
			OIDCIssuer:  "https://invalid-issuer.example.com",
		}

		_, err := NewOIDCProvider(config)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to discover OIDC provider")
	})

	t.Run("DiscoverProvider_Success", func(t *testing.T) {
		// Create a mock OIDC discovery server
		var serverURL string
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/.well-known/openid_configuration" {
				providerInfo := OIDCProviderInfo{
					Issuer:                 serverURL,
					AuthorizationEndpoint:  serverURL + "/auth",
					TokenEndpoint:          serverURL + "/token",
					JWKSUri:                serverURL + "/jwks",
					SupportedScopes:        []string{"openid", "profile", "email"},
					SupportedResponseTypes: []string{"code", "id_token"},
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(providerInfo)
			} else {
				w.WriteHeader(http.StatusNotFound)
			}
		}))
		defer mockServer.Close()
		serverURL = mockServer.URL

		config := &AuthConfig{
			OIDCEnabled: true,
			OIDCIssuer:  mockServer.URL,
		}

		provider, err := NewOIDCProvider(config)
		require.NoError(t, err)

		info, err := provider.GetProviderInfo(context.Background())
		require.NoError(t, err)
		assert.Equal(t, mockServer.URL, info.Issuer)
		assert.Equal(t, mockServer.URL+"/auth", info.AuthorizationEndpoint)
		assert.Equal(t, mockServer.URL+"/token", info.TokenEndpoint)
		assert.Equal(t, mockServer.URL+"/jwks", info.JWKSUri)
	})

	t.Run("ValidateToken_NotImplemented", func(t *testing.T) {
		// Create a mock OIDC discovery server
		var serverURL string
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path == "/.well-known/openid_configuration" {
				providerInfo := OIDCProviderInfo{
					Issuer:  serverURL,
					JWKSUri: serverURL + "/jwks",
				}
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(providerInfo)
			}
		}))
		defer mockServer.Close()
		serverURL = mockServer.URL

		config := &AuthConfig{
			OIDCEnabled: true,
			OIDCIssuer:  mockServer.URL,
		}

		provider, err := NewOIDCProvider(config)
		require.NoError(t, err)

		// This should fail because JWKS integration is not implemented
		_, err = provider.ValidateToken(context.Background(), "fake-jwt-token")
		assert.Error(t, err)
		// The error could be from JWT parsing or JWKS retrieval - just check that it fails
		assert.Contains(t, err.Error(), "token")
	})
}

func TestDeriveTenantFromEmail(t *testing.T) {
	provider := &oidcProvider{}

	tests := []struct {
		email    string
		expected string
	}{
		{"user@example.com", "example-com"},
		{"admin@company.org", "company-org"},
		{"", "default"},
		{"invalid-email", "default"},
		{"user@sub.domain.com", "sub-domain-com"},
	}

	for _, test := range tests {
		t.Run(test.email, func(t *testing.T) {
			result := provider.deriveTenantFromEmail(test.email)
			assert.Equal(t, test.expected, result)
		})
	}
}

func TestHybridAuthenticator_OIDC(t *testing.T) {
	t.Run("OIDC_Fallback_Integration", func(t *testing.T) {
		// Create a mock OIDC server that returns errors
		mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
		}))
		defer mockServer.Close()

		config := &AuthConfig{
			JWTSecret:   "test-secret-32-characters-long",
			TokenExpiry: time.Hour, // Use longer expiry for tests
			OIDCEnabled: true,
			OIDCIssuer:  mockServer.URL,
		}

		// Should create authenticator even if OIDC fails
		auth, err := NewHybridAuthenticator(config)
		require.NoError(t, err)

		// Issue a JWT token
		tokenReq := &TokenRequest{
			TenantID: "tenant123",
			UserID:   "user456",
		}

		tokenResp, err := auth.IssueToken(context.Background(), tokenReq)
		require.NoError(t, err)

		// Validate should fall back to JWT when OIDC fails
		claims, err := auth.ValidateToken(context.Background(), tokenResp.AccessToken)
		require.NoError(t, err)
		assert.Equal(t, "tenant123", claims.TenantID)
		assert.Equal(t, "user456", claims.UserID)
	})

	t.Run("OIDC_Disabled_JWT_Only", func(t *testing.T) {
		config := &AuthConfig{
			JWTSecret:   "test-secret-32-characters-long",
			TokenExpiry: time.Hour, // Use longer expiry for tests
			OIDCEnabled: false,
		}

		auth, err := NewHybridAuthenticator(config)
		require.NoError(t, err)

		// Should work with JWT only
		tokenReq := &TokenRequest{
			TenantID: "tenant123",
			UserID:   "user456",
		}

		tokenResp, err := auth.IssueToken(context.Background(), tokenReq)
		require.NoError(t, err)

		claims, err := auth.ValidateToken(context.Background(), tokenResp.AccessToken)
		require.NoError(t, err)
		assert.Equal(t, "tenant123", claims.TenantID)
		assert.Equal(t, "user456", claims.UserID)
	})
}
