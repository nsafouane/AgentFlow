package security

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJWTAuthenticator(t *testing.T) {
	config := &AuthConfig{
		JWTSecret:     "test-secret-32-characters-long",
		TokenExpiry:   time.Hour,
		RefreshExpiry: 24 * time.Hour,
		OIDCEnabled:   false,
	}

	auth := NewAuthenticator(config)
	ctx := context.Background()

	t.Run("IssueToken_Success", func(t *testing.T) {
		req := &TokenRequest{
			TenantID:    "tenant123",
			UserID:      "user456",
			Roles:       []string{"admin", "developer"},
			Permissions: []string{"workflows:*", "agents:read"},
		}

		resp, err := auth.IssueToken(ctx, req)
		require.NoError(t, err)
		assert.NotEmpty(t, resp.AccessToken)
		assert.Equal(t, "Bearer", resp.TokenType)
		assert.Equal(t, int64(3600), resp.ExpiresIn)
		assert.NotEmpty(t, resp.RefreshToken)
		assert.Greater(t, resp.IssuedAt, int64(0))
	})

	t.Run("IssueToken_MissingTenantID", func(t *testing.T) {
		req := &TokenRequest{
			UserID: "user456",
		}

		_, err := auth.IssueToken(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "tenant_id is required")
	})

	t.Run("IssueToken_MissingUserID", func(t *testing.T) {
		req := &TokenRequest{
			TenantID: "tenant123",
		}

		_, err := auth.IssueToken(ctx, req)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "user_id is required")
	})

	t.Run("ValidateToken_Success", func(t *testing.T) {
		// Issue a token first
		req := &TokenRequest{
			TenantID:    "tenant123",
			UserID:      "user456",
			Roles:       []string{"admin"},
			Permissions: []string{"workflows:*"},
		}

		resp, err := auth.IssueToken(ctx, req)
		require.NoError(t, err)

		// Validate the token
		claims, err := auth.ValidateToken(ctx, resp.AccessToken)
		require.NoError(t, err)
		assert.Equal(t, "tenant123", claims.TenantID)
		assert.Equal(t, "user456", claims.UserID)
		assert.Equal(t, []string{"admin"}, claims.Roles)
		assert.Equal(t, []string{"workflows:*"}, claims.Permissions)
	})

	t.Run("ValidateToken_InvalidToken", func(t *testing.T) {
		_, err := auth.ValidateToken(ctx, "invalid-token")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid token")
	})

	t.Run("ValidateToken_ExpiredToken", func(t *testing.T) {
		// Create config with very short expiry
		shortConfig := &AuthConfig{
			JWTSecret:   "test-secret-32-characters-long",
			TokenExpiry: time.Millisecond,
		}
		shortAuth := NewAuthenticator(shortConfig)

		req := &TokenRequest{
			TenantID: "tenant123",
			UserID:   "user456",
		}

		resp, err := shortAuth.IssueToken(ctx, req)
		require.NoError(t, err)

		// Wait for token to expire
		time.Sleep(10 * time.Millisecond)

		_, err = shortAuth.ValidateToken(ctx, resp.AccessToken)
		assert.Error(t, err)
	})

	t.Run("RevokeToken_Success", func(t *testing.T) {
		// Issue a token
		req := &TokenRequest{
			TenantID: "tenant123",
			UserID:   "user456",
		}

		resp, err := auth.IssueToken(ctx, req)
		require.NoError(t, err)

		// Validate token works initially
		_, err = auth.ValidateToken(ctx, resp.AccessToken)
		require.NoError(t, err)

		// Revoke token
		err = auth.RevokeToken(ctx, resp.AccessToken)
		require.NoError(t, err)

		// Token should now be invalid
		_, err = auth.ValidateToken(ctx, resp.AccessToken)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "revoked")
	})
}

func TestAuthConfig(t *testing.T) {
	t.Run("DefaultAuthConfig", func(t *testing.T) {
		config := DefaultAuthConfig()
		assert.NotEmpty(t, config.JWTSecret)
		assert.Equal(t, 24*time.Hour, config.TokenExpiry)
		assert.Equal(t, 7*24*time.Hour, config.RefreshExpiry)
		assert.False(t, config.OIDCEnabled)
	})

	t.Run("LoadAuthConfigFromEnv", func(t *testing.T) {
		// Set environment variables
		os.Setenv("AF_JWT_SECRET", "env-secret")
		os.Setenv("AF_TOKEN_EXPIRY", "2h")
		os.Setenv("AF_OIDC_ENABLED", "true")
		os.Setenv("AF_OIDC_ISSUER", "https://example.com")
		defer func() {
			os.Unsetenv("AF_JWT_SECRET")
			os.Unsetenv("AF_TOKEN_EXPIRY")
			os.Unsetenv("AF_OIDC_ENABLED")
			os.Unsetenv("AF_OIDC_ISSUER")
		}()

		config := LoadAuthConfigFromEnv()
		assert.Equal(t, "env-secret", config.JWTSecret)
		assert.Equal(t, 2*time.Hour, config.TokenExpiry)
		assert.True(t, config.OIDCEnabled)
		assert.Equal(t, "https://example.com", config.OIDCIssuer)
	})
}

func TestExtractTokenFromHeader(t *testing.T) {
	t.Run("ValidBearerToken", func(t *testing.T) {
		token, err := ExtractTokenFromHeader("Bearer abc123")
		require.NoError(t, err)
		assert.Equal(t, "abc123", token)
	})

	t.Run("EmptyHeader", func(t *testing.T) {
		_, err := ExtractTokenFromHeader("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "authorization header is required")
	})

	t.Run("InvalidFormat", func(t *testing.T) {
		_, err := ExtractTokenFromHeader("abc123")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid authorization header format")
	})

	t.Run("WrongScheme", func(t *testing.T) {
		_, err := ExtractTokenFromHeader("Basic abc123")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "must use Bearer scheme")
	})
}

func TestAgentFlowClaims(t *testing.T) {
	t.Run("ToPrincipal", func(t *testing.T) {
		claims := &AgentFlowClaims{
			TenantID: "tenant123",
			UserID:   "user456",
			Roles:    []string{"admin", "developer"},
		}

		principal := claims.ToPrincipal()
		assert.Equal(t, "user456", principal.ID)
		assert.Equal(t, "tenant123", principal.TenantID)
		assert.Equal(t, []string{"admin", "developer"}, principal.Roles)
	})
}

func TestHybridAuthenticator(t *testing.T) {
	t.Run("JWTFallback_OIDCDisabled", func(t *testing.T) {
		config := &AuthConfig{
			JWTSecret:   "test-secret-32-characters-long",
			TokenExpiry: time.Hour,
			OIDCEnabled: false,
		}

		auth, err := NewHybridAuthenticator(config)
		require.NoError(t, err)

		// Issue and validate token should work with JWT
		req := &TokenRequest{
			TenantID: "tenant123",
			UserID:   "user456",
		}

		resp, err := auth.IssueToken(context.Background(), req)
		require.NoError(t, err)

		claims, err := auth.ValidateToken(context.Background(), resp.AccessToken)
		require.NoError(t, err)
		assert.Equal(t, "tenant123", claims.TenantID)
		assert.Equal(t, "user456", claims.UserID)
	})

	t.Run("OIDCEnabled_FallbackToJWT", func(t *testing.T) {
		config := &AuthConfig{
			JWTSecret:   "test-secret-32-characters-long",
			TokenExpiry: time.Hour,
			OIDCEnabled: true,
			OIDCIssuer:  "https://invalid-issuer.example.com",
		}

		// Should create authenticator even if OIDC setup fails
		auth, err := NewHybridAuthenticator(config)
		require.NoError(t, err)

		// Should still work with JWT fallback
		req := &TokenRequest{
			TenantID: "tenant123",
			UserID:   "user456",
		}

		resp, err := auth.IssueToken(context.Background(), req)
		require.NoError(t, err)

		claims, err := auth.ValidateToken(context.Background(), resp.AccessToken)
		require.NoError(t, err)
		assert.Equal(t, "tenant123", claims.TenantID)
	})
}
