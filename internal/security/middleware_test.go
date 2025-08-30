package security

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/agentflow/agentflow/internal/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthMiddleware(t *testing.T) {
	config := &AuthConfig{
		JWTSecret:   "test-secret-32-characters-long",
		TokenExpiry: time.Hour, // Use longer expiry for tests
		OIDCEnabled: false,
	}

	auth := NewAuthenticator(config)
	logger := logging.NewLogger()
	middleware := NewAuthMiddleware(auth, logger, config)

	// Create a test handler that checks for authentication
	testHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims := GetClaimsFromContext(r.Context())
		if claims != nil {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("authenticated"))
		} else {
			w.WriteHeader(http.StatusUnauthorized)
			w.Write([]byte("not authenticated"))
		}
	})

	t.Run("PublicEndpoint_NoAuth", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		// Use a simple handler for public endpoints that always returns OK
		publicHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("public"))
		})

		handler := middleware.Middleware()(publicHandler)
		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("ProtectedEndpoint_NoToken", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/workflows", nil)
		w := httptest.NewRecorder()

		handler := middleware.Middleware()(testHandler)
		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "missing_authorization_header")
	})

	t.Run("ProtectedEndpoint_InvalidToken", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/workflows", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")
		w := httptest.NewRecorder()

		handler := middleware.Middleware()(testHandler)
		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "invalid_token")
	})

	t.Run("ProtectedEndpoint_ValidToken", func(t *testing.T) {
		// Issue a valid token
		tokenReq := &TokenRequest{
			TenantID: "tenant123",
			UserID:   "user456",
			Roles:    []string{"admin"},
		}

		tokenResp, err := auth.IssueToken(context.Background(), tokenReq)
		require.NoError(t, err)

		req := httptest.NewRequest("GET", "/api/v1/workflows", nil)
		req.Header.Set("Authorization", "Bearer "+tokenResp.AccessToken)
		w := httptest.NewRecorder()

		handler := middleware.Middleware()(testHandler)
		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "authenticated", w.Body.String())
	})

	t.Run("RequireRole_Success", func(t *testing.T) {
		// Issue a token with admin role
		tokenReq := &TokenRequest{
			TenantID: "tenant123",
			UserID:   "user456",
			Roles:    []string{"admin"},
		}

		tokenResp, err := auth.IssueToken(context.Background(), tokenReq)
		require.NoError(t, err)

		req := httptest.NewRequest("GET", "/api/v1/admin", nil)
		req.Header.Set("Authorization", "Bearer "+tokenResp.AccessToken)
		w := httptest.NewRecorder()

		// Apply both auth and role middleware
		handler := middleware.Middleware()(middleware.RequireRole("admin")(testHandler))
		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("RequireRole_Failure", func(t *testing.T) {
		// Issue a token with viewer role
		tokenReq := &TokenRequest{
			TenantID: "tenant123",
			UserID:   "user456",
			Roles:    []string{"viewer"},
		}

		tokenResp, err := auth.IssueToken(context.Background(), tokenReq)
		require.NoError(t, err)

		req := httptest.NewRequest("GET", "/api/v1/admin", nil)
		req.Header.Set("Authorization", "Bearer "+tokenResp.AccessToken)
		w := httptest.NewRecorder()

		// Apply both auth and role middleware
		handler := middleware.Middleware()(middleware.RequireRole("admin")(testHandler))
		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.Contains(t, w.Body.String(), "insufficient_permissions")
	})

	t.Run("RequirePermission_Success", func(t *testing.T) {
		// Issue a token with specific permission
		tokenReq := &TokenRequest{
			TenantID:    "tenant123",
			UserID:      "user456",
			Permissions: []string{"workflows:read"},
		}

		tokenResp, err := auth.IssueToken(context.Background(), tokenReq)
		require.NoError(t, err)

		req := httptest.NewRequest("GET", "/api/v1/workflows", nil)
		req.Header.Set("Authorization", "Bearer "+tokenResp.AccessToken)
		w := httptest.NewRecorder()

		// Apply both auth and permission middleware
		handler := middleware.Middleware()(middleware.RequirePermission("workflows", "read")(testHandler))
		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("RequirePermission_WildcardSuccess", func(t *testing.T) {
		// Issue a token with wildcard permission
		tokenReq := &TokenRequest{
			TenantID:    "tenant123",
			UserID:      "user456",
			Permissions: []string{"workflows:*"},
		}

		tokenResp, err := auth.IssueToken(context.Background(), tokenReq)
		require.NoError(t, err)

		req := httptest.NewRequest("POST", "/api/v1/workflows", nil)
		req.Header.Set("Authorization", "Bearer "+tokenResp.AccessToken)
		w := httptest.NewRecorder()

		// Apply both auth and permission middleware
		handler := middleware.Middleware()(middleware.RequirePermission("workflows", "write")(testHandler))
		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("RequirePermission_Failure", func(t *testing.T) {
		// Issue a token without required permission
		tokenReq := &TokenRequest{
			TenantID:    "tenant123",
			UserID:      "user456",
			Permissions: []string{"agents:read"},
		}

		tokenResp, err := auth.IssueToken(context.Background(), tokenReq)
		require.NoError(t, err)

		req := httptest.NewRequest("GET", "/api/v1/workflows", nil)
		req.Header.Set("Authorization", "Bearer "+tokenResp.AccessToken)
		w := httptest.NewRecorder()

		// Apply both auth and permission middleware
		handler := middleware.Middleware()(middleware.RequirePermission("workflows", "read")(testHandler))
		handler.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.Contains(t, w.Body.String(), "workflows:read")
	})
}

func TestContextHelpers(t *testing.T) {
	claims := &AgentFlowClaims{
		TenantID:    "tenant123",
		UserID:      "user456",
		Roles:       []string{"admin", "developer"},
		Permissions: []string{"workflows:*", "agents:read"},
	}

	ctx := context.Background()
	ctx = context.WithValue(ctx, "auth_claims", claims)
	ctx = context.WithValue(ctx, "tenant_id", claims.TenantID)
	ctx = context.WithValue(ctx, "user_id", claims.UserID)
	ctx = context.WithValue(ctx, "user_roles", claims.Roles)
	ctx = context.WithValue(ctx, "user_permissions", claims.Permissions)

	t.Run("GetClaimsFromContext", func(t *testing.T) {
		result := GetClaimsFromContext(ctx)
		require.NotNil(t, result)
		assert.Equal(t, claims.TenantID, result.TenantID)
		assert.Equal(t, claims.UserID, result.UserID)
	})

	t.Run("GetTenantIDFromContext", func(t *testing.T) {
		result := GetTenantIDFromContext(ctx)
		assert.Equal(t, "tenant123", result)
	})

	t.Run("GetUserIDFromContext", func(t *testing.T) {
		result := GetUserIDFromContext(ctx)
		assert.Equal(t, "user456", result)
	})

	t.Run("GetUserRolesFromContext", func(t *testing.T) {
		result := GetUserRolesFromContext(ctx)
		assert.Equal(t, []string{"admin", "developer"}, result)
	})

	t.Run("GetUserPermissionsFromContext", func(t *testing.T) {
		result := GetUserPermissionsFromContext(ctx)
		assert.Equal(t, []string{"workflows:*", "agents:read"}, result)
	})

	t.Run("HasRole", func(t *testing.T) {
		assert.True(t, HasRole(ctx, "admin"))
		assert.True(t, HasRole(ctx, "developer"))
		assert.False(t, HasRole(ctx, "viewer"))
	})

	t.Run("HasPermission", func(t *testing.T) {
		assert.True(t, HasPermission(ctx, "workflows", "read"))
		assert.True(t, HasPermission(ctx, "workflows", "write"))
		assert.True(t, HasPermission(ctx, "agents", "read"))
		assert.False(t, HasPermission(ctx, "agents", "write"))
		assert.False(t, HasPermission(ctx, "tools", "read"))
	})

	t.Run("EmptyContext", func(t *testing.T) {
		emptyCtx := context.Background()

		assert.Nil(t, GetClaimsFromContext(emptyCtx))
		assert.Empty(t, GetTenantIDFromContext(emptyCtx))
		assert.Empty(t, GetUserIDFromContext(emptyCtx))
		assert.Empty(t, GetUserRolesFromContext(emptyCtx))
		assert.Empty(t, GetUserPermissionsFromContext(emptyCtx))
		assert.False(t, HasRole(emptyCtx, "admin"))
		assert.False(t, HasPermission(emptyCtx, "workflows", "read"))
	})
}
