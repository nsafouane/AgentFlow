package security

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/agentflow/agentflow/internal/logging"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAuthHandlers(t *testing.T) {
	config := &AuthConfig{
		JWTSecret:   "test-secret-32-characters-long",
		TokenExpiry: time.Hour, // Use longer expiry for tests
		OIDCEnabled: false,
	}

	auth := NewAuthenticator(config)
	logger := logging.NewLogger()
	handlers := NewAuthHandlers(auth, logger)

	t.Run("HandleTokenIssue_Success", func(t *testing.T) {
		reqBody := TokenIssueRequest{
			TenantID:    "tenant123",
			UserID:      "user456",
			Roles:       []string{"admin"},
			Permissions: []string{"workflows:*"},
			ExpiresIn:   "1h",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/v1/auth/token", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		handlers.HandleTokenIssue(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.True(t, response["success"].(bool))
		data := response["data"].(map[string]interface{})
		assert.NotEmpty(t, data["access_token"])
		assert.Equal(t, "Bearer", data["token_type"])
		assert.NotEmpty(t, data["refresh_token"])
	})

	t.Run("HandleTokenIssue_InvalidMethod", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/auth/token", nil)
		w := httptest.NewRecorder()

		handlers.HandleTokenIssue(w, req)

		assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
	})

	t.Run("HandleTokenIssue_InvalidJSON", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/auth/token", bytes.NewReader([]byte("invalid json")))
		w := httptest.NewRecorder()

		handlers.HandleTokenIssue(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "invalid_request")
	})

	t.Run("HandleTokenIssue_MissingTenantID", func(t *testing.T) {
		reqBody := TokenIssueRequest{
			UserID: "user456",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/v1/auth/token", bytes.NewReader(body))
		w := httptest.NewRecorder()

		handlers.HandleTokenIssue(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "missing_tenant_id")
	})

	t.Run("HandleTokenIssue_MissingUserID", func(t *testing.T) {
		reqBody := TokenIssueRequest{
			TenantID: "tenant123",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/v1/auth/token", bytes.NewReader(body))
		w := httptest.NewRecorder()

		handlers.HandleTokenIssue(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "missing_user_id")
	})

	t.Run("HandleTokenIssue_InvalidExpiresIn", func(t *testing.T) {
		reqBody := TokenIssueRequest{
			TenantID:  "tenant123",
			UserID:    "user456",
			ExpiresIn: "invalid-duration",
		}

		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest("POST", "/api/v1/auth/token", bytes.NewReader(body))
		w := httptest.NewRecorder()

		handlers.HandleTokenIssue(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
		assert.Contains(t, w.Body.String(), "invalid_expires_in")
	})

	t.Run("HandleTokenValidate_Success", func(t *testing.T) {
		// First issue a token
		tokenReq := &TokenRequest{
			TenantID: "tenant123",
			UserID:   "user456",
			Roles:    []string{"admin"},
		}

		tokenResp, err := auth.IssueToken(context.Background(), tokenReq)
		require.NoError(t, err)

		// Now validate it
		req := httptest.NewRequest("POST", "/api/v1/auth/validate", nil)
		req.Header.Set("Authorization", "Bearer "+tokenResp.AccessToken)
		w := httptest.NewRecorder()

		handlers.HandleTokenValidate(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.True(t, response["success"].(bool))
		data := response["data"].(map[string]interface{})
		assert.True(t, data["valid"].(bool))
		assert.Equal(t, "tenant123", data["tenant_id"])
		assert.Equal(t, "user456", data["user_id"])
	})

	t.Run("HandleTokenValidate_InvalidMethod", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/auth/validate", nil)
		w := httptest.NewRecorder()

		handlers.HandleTokenValidate(w, req)

		assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
	})

	t.Run("HandleTokenValidate_MissingToken", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/auth/validate", nil)
		w := httptest.NewRecorder()

		handlers.HandleTokenValidate(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "missing_authorization_header")
	})

	t.Run("HandleTokenValidate_InvalidToken", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/auth/validate", nil)
		req.Header.Set("Authorization", "Bearer invalid-token")
		w := httptest.NewRecorder()

		handlers.HandleTokenValidate(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "invalid_token")
	})

	t.Run("HandleTokenRevoke_Success", func(t *testing.T) {
		// First issue a token
		tokenReq := &TokenRequest{
			TenantID: "tenant123",
			UserID:   "user456",
		}

		tokenResp, err := auth.IssueToken(context.Background(), tokenReq)
		require.NoError(t, err)

		// Now revoke it
		req := httptest.NewRequest("POST", "/api/v1/auth/revoke", nil)
		req.Header.Set("Authorization", "Bearer "+tokenResp.AccessToken)
		w := httptest.NewRecorder()

		handlers.HandleTokenRevoke(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.True(t, response["success"].(bool))
		data := response["data"].(map[string]interface{})
		assert.True(t, data["revoked"].(bool))
	})

	t.Run("HandleUserInfo_Success", func(t *testing.T) {
		// Create a request with claims in context (simulating auth middleware)
		now := time.Now()
		claims := &AgentFlowClaims{
			TenantID:    "tenant123",
			UserID:      "user456",
			Roles:       []string{"admin"},
			Permissions: []string{"workflows:*"},
			RegisteredClaims: jwt.RegisteredClaims{
				IssuedAt:  jwt.NewNumericDate(now),
				ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour)),
			},
		}

		req := httptest.NewRequest("GET", "/api/v1/auth/userinfo", nil)
		ctx := context.WithValue(req.Context(), "auth_claims", claims)
		req = req.WithContext(ctx)
		w := httptest.NewRecorder()

		handlers.HandleUserInfo(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.True(t, response["success"].(bool))
		data := response["data"].(map[string]interface{})
		assert.Equal(t, "user456", data["user_id"])
		assert.Equal(t, "tenant123", data["tenant_id"])
	})

	t.Run("HandleUserInfo_Unauthenticated", func(t *testing.T) {
		req := httptest.NewRequest("GET", "/api/v1/auth/userinfo", nil)
		w := httptest.NewRecorder()

		handlers.HandleUserInfo(w, req)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "unauthenticated")
	})

	t.Run("HandleUserInfo_InvalidMethod", func(t *testing.T) {
		req := httptest.NewRequest("POST", "/api/v1/auth/userinfo", nil)
		w := httptest.NewRecorder()

		handlers.HandleUserInfo(w, req)

		assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
	})
}

func TestGetAuthEndpoints(t *testing.T) {
	config := &AuthConfig{
		JWTSecret:   "test-secret-32-characters-long",
		TokenExpiry: time.Hour, // Use longer expiry for tests
		OIDCEnabled: false,
	}

	auth := NewAuthenticator(config)
	logger := logging.NewLogger()
	handlers := NewAuthHandlers(auth, logger)
	middleware := NewAuthMiddleware(auth, logger, config)

	endpoints := GetAuthEndpoints(handlers, middleware)

	expectedEndpoints := []string{
		"POST /api/v1/auth/token",
		"POST /api/v1/auth/validate",
		"POST /api/v1/auth/revoke",
		"GET /api/v1/auth/userinfo",
	}

	for _, endpoint := range expectedEndpoints {
		assert.Contains(t, endpoints, endpoint)
		assert.NotNil(t, endpoints[endpoint])
	}
}
