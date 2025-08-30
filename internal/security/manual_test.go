package security

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/agentflow/agentflow/internal/logging"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/require"
)

// TestManualAuthenticationFlow demonstrates the complete authentication flow
func TestManualAuthenticationFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping manual test in short mode")
	}

	fmt.Println("=== Manual Authentication Flow Test ===")

	// Setup
	config := &AuthConfig{
		JWTSecret:   "test-secret-32-characters-long",
		TokenExpiry: time.Hour,
		OIDCEnabled: false,
	}

	auth := NewAuthenticator(config)
	logger := logging.NewLogger()
	middleware := NewAuthMiddleware(auth, logger, config)
	handlers := NewAuthHandlers(auth, logger)

	// Create test server
	router := mux.NewRouter()

	// Public endpoints
	router.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"healthy"}`))
	}).Methods("GET")

	// Auth endpoints
	router.HandleFunc("/api/v1/auth/token", handlers.HandleTokenIssue).Methods("POST")
	router.HandleFunc("/api/v1/auth/validate", handlers.HandleTokenValidate).Methods("POST")

	// Protected auth endpoints
	authRouter := router.PathPrefix("/api/v1/auth").Subrouter()
	authRouter.Use(middleware.Middleware())
	authRouter.HandleFunc("/userinfo", handlers.HandleUserInfo).Methods("GET")

	// Protected endpoints
	protectedRouter := router.PathPrefix("/api/v1").Subrouter()
	protectedRouter.Use(middleware.Middleware())
	protectedRouter.HandleFunc("/workflows", func(w http.ResponseWriter, r *http.Request) {
		claims := GetClaimsFromContext(r.Context())
		response := map[string]interface{}{
			"message":   "Workflows endpoint accessed successfully",
			"user_id":   claims.UserID,
			"tenant_id": claims.TenantID,
			"roles":     claims.Roles,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}).Methods("GET")

	server := httptest.NewServer(router)
	defer server.Close()

	fmt.Printf("Test server started at: %s\n", server.URL)

	// Test 1: Access public endpoint
	fmt.Println("\n1. Testing public endpoint access...")
	resp, err := http.Get(server.URL + "/health")
	require.NoError(t, err)
	defer resp.Body.Close()
	fmt.Printf("   Health check status: %d\n", resp.StatusCode)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	// Test 2: Access protected endpoint without token (should fail)
	fmt.Println("\n2. Testing protected endpoint without token...")
	resp, err = http.Get(server.URL + "/api/v1/workflows")
	require.NoError(t, err)
	defer resp.Body.Close()
	fmt.Printf("   Protected endpoint without token status: %d\n", resp.StatusCode)
	require.Equal(t, http.StatusUnauthorized, resp.StatusCode)

	// Test 3: Issue a token
	fmt.Println("\n3. Issuing authentication token...")
	tokenRequest := TokenIssueRequest{
		TenantID:    "demo-tenant",
		UserID:      "demo-user",
		Roles:       []string{"admin", "developer"},
		Permissions: []string{"workflows:*", "agents:read"},
		ExpiresIn:   "1h",
	}

	tokenBody, _ := json.Marshal(tokenRequest)
	resp, err = http.Post(server.URL+"/api/v1/auth/token", "application/json", bytes.NewReader(tokenBody))
	require.NoError(t, err)
	defer resp.Body.Close()

	var tokenResponse map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&tokenResponse)
	require.NoError(t, err)

	fmt.Printf("   Token issuance status: %d\n", resp.StatusCode)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	require.True(t, tokenResponse["success"].(bool))

	tokenData := tokenResponse["data"].(map[string]interface{})
	accessToken := tokenData["access_token"].(string)
	fmt.Printf("   Access token issued (length: %d)\n", len(accessToken))
	fmt.Printf("   Token type: %s\n", tokenData["token_type"])
	fmt.Printf("   Expires in: %.0f seconds\n", tokenData["expires_in"])

	// Test 4: Validate the token
	fmt.Println("\n4. Validating token...")
	req, _ := http.NewRequest("POST", server.URL+"/api/v1/auth/validate", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)

	client := &http.Client{}
	resp, err = client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	var validateResponse map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&validateResponse)
	require.NoError(t, err)

	fmt.Printf("   Token validation status: %d\n", resp.StatusCode)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	validateData := validateResponse["data"].(map[string]interface{})
	fmt.Printf("   Token valid: %t\n", validateData["valid"])
	fmt.Printf("   User ID: %s\n", validateData["user_id"])
	fmt.Printf("   Tenant ID: %s\n", validateData["tenant_id"])

	// Test 5: Access protected endpoint with token
	fmt.Println("\n5. Accessing protected endpoint with token...")
	req, _ = http.NewRequest("GET", server.URL+"/api/v1/workflows", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err = client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	var workflowResponse map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&workflowResponse)
	require.NoError(t, err)

	fmt.Printf("   Protected endpoint status: %d\n", resp.StatusCode)
	require.Equal(t, http.StatusOK, resp.StatusCode)
	fmt.Printf("   Response message: %s\n", workflowResponse["message"])
	fmt.Printf("   Authenticated user: %s\n", workflowResponse["user_id"])
	fmt.Printf("   User roles: %v\n", workflowResponse["roles"])

	// Test 6: Get user info
	fmt.Println("\n6. Getting user information...")
	req, _ = http.NewRequest("GET", server.URL+"/api/v1/auth/userinfo", nil)
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err = client.Do(req)
	require.NoError(t, err)
	defer resp.Body.Close()

	var userInfoResponse map[string]interface{}
	err = json.NewDecoder(resp.Body).Decode(&userInfoResponse)
	require.NoError(t, err)

	fmt.Printf("   User info status: %d\n", resp.StatusCode)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	userInfoData := userInfoResponse["data"].(map[string]interface{})
	fmt.Printf("   User ID: %s\n", userInfoData["user_id"])
	fmt.Printf("   Tenant ID: %s\n", userInfoData["tenant_id"])
	fmt.Printf("   Roles: %v\n", userInfoData["roles"])
	fmt.Printf("   Permissions: %v\n", userInfoData["permissions"])

	fmt.Println("\n=== Authentication Flow Test Completed Successfully ===")
}

// TestManualOIDCFlagToggle demonstrates OIDC flag behavior
func TestManualOIDCFlagToggle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping manual test in short mode")
	}

	fmt.Println("\n=== Manual OIDC Flag Toggle Test ===")

	// Test with OIDC disabled
	fmt.Println("\n1. Testing with OIDC disabled...")
	config1 := &AuthConfig{
		JWTSecret:   "test-secret-32-characters-long",
		TokenExpiry: time.Hour,
		OIDCEnabled: false,
	}

	auth1, err := NewHybridAuthenticator(config1)
	require.NoError(t, err)
	fmt.Println("   ✓ Hybrid authenticator created with OIDC disabled")

	// Issue and validate token
	tokenReq := &TokenRequest{
		TenantID: "test-tenant",
		UserID:   "test-user",
	}

	tokenResp, err := auth1.IssueToken(context.Background(), tokenReq)
	require.NoError(t, err)
	fmt.Printf("   ✓ JWT token issued successfully (length: %d)\n", len(tokenResp.AccessToken))

	claims, err := auth1.ValidateToken(context.Background(), tokenResp.AccessToken)
	require.NoError(t, err)
	fmt.Printf("   ✓ JWT token validated successfully (user: %s, tenant: %s)\n",
		claims.UserID, claims.TenantID)

	// Test with OIDC enabled but invalid issuer (should fall back to JWT)
	fmt.Println("\n2. Testing with OIDC enabled but invalid issuer...")
	config2 := &AuthConfig{
		JWTSecret:   "test-secret-32-characters-long",
		TokenExpiry: time.Hour,
		OIDCEnabled: true,
		OIDCIssuer:  "https://invalid-issuer.example.com",
	}

	auth2, err := NewHybridAuthenticator(config2)
	require.NoError(t, err)
	fmt.Println("   ✓ Hybrid authenticator created with OIDC enabled (invalid issuer)")

	// Should still work with JWT fallback
	tokenResp2, err := auth2.IssueToken(context.Background(), tokenReq)
	require.NoError(t, err)
	fmt.Printf("   ✓ JWT token issued successfully with OIDC fallback (length: %d)\n",
		len(tokenResp2.AccessToken))

	claims2, err := auth2.ValidateToken(context.Background(), tokenResp2.AccessToken)
	require.NoError(t, err)
	fmt.Printf("   ✓ JWT token validated with OIDC fallback (user: %s, tenant: %s)\n",
		claims2.UserID, claims2.TenantID)

	fmt.Println("\n=== OIDC Flag Toggle Test Completed Successfully ===")
}

// TestManualTokenLifecycle demonstrates token lifecycle management
func TestManualTokenLifecycle(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping manual test in short mode")
	}

	fmt.Println("\n=== Manual Token Lifecycle Test ===")

	config := &AuthConfig{
		JWTSecret:   "test-secret-32-characters-long",
		TokenExpiry: 2 * time.Second, // Short expiry for testing
		OIDCEnabled: false,
	}

	auth := NewAuthenticator(config)
	ctx := context.Background()

	// Test 1: Issue token
	fmt.Println("\n1. Issuing token with short expiry...")
	tokenReq := &TokenRequest{
		TenantID: "lifecycle-tenant",
		UserID:   "lifecycle-user",
		Roles:    []string{"tester"},
	}

	tokenResp, err := auth.IssueToken(ctx, tokenReq)
	require.NoError(t, err)
	fmt.Printf("   ✓ Token issued (expires in %d seconds)\n", tokenResp.ExpiresIn)

	// Test 2: Validate fresh token
	fmt.Println("\n2. Validating fresh token...")
	claims, err := auth.ValidateToken(ctx, tokenResp.AccessToken)
	require.NoError(t, err)
	fmt.Printf("   ✓ Fresh token valid (user: %s)\n", claims.UserID)

	// Test 3: Revoke token
	fmt.Println("\n3. Revoking token...")
	err = auth.RevokeToken(ctx, tokenResp.AccessToken)
	require.NoError(t, err)
	fmt.Println("   ✓ Token revoked successfully")

	// Test 4: Validate revoked token (should fail)
	fmt.Println("\n4. Validating revoked token...")
	_, err = auth.ValidateToken(ctx, tokenResp.AccessToken)
	require.Error(t, err)
	fmt.Printf("   ✓ Revoked token rejected: %s\n", err.Error())

	// Test 5: Issue new token and wait for expiry
	fmt.Println("\n5. Testing token expiry...")
	tokenResp2, err := auth.IssueToken(ctx, tokenReq)
	require.NoError(t, err)
	fmt.Println("   ✓ New token issued")

	// Validate immediately
	_, err = auth.ValidateToken(ctx, tokenResp2.AccessToken)
	require.NoError(t, err)
	fmt.Println("   ✓ New token valid immediately")

	// Wait for expiry
	fmt.Println("   Waiting for token to expire...")
	time.Sleep(3 * time.Second)

	// Validate expired token (should fail)
	_, err = auth.ValidateToken(ctx, tokenResp2.AccessToken)
	require.Error(t, err)
	fmt.Printf("   ✓ Expired token rejected: %s\n", err.Error())

	fmt.Println("\n=== Token Lifecycle Test Completed Successfully ===")
}

// TestManualEnvironmentConfiguration demonstrates environment-based configuration
func TestManualEnvironmentConfiguration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping manual test in short mode")
	}

	fmt.Println("\n=== Manual Environment Configuration Test ===")

	// Save original environment
	originalSecret := os.Getenv("AF_JWT_SECRET")
	originalExpiry := os.Getenv("AF_TOKEN_EXPIRY")
	originalOIDC := os.Getenv("AF_OIDC_ENABLED")

	defer func() {
		// Restore original environment
		if originalSecret != "" {
			os.Setenv("AF_JWT_SECRET", originalSecret)
		} else {
			os.Unsetenv("AF_JWT_SECRET")
		}
		if originalExpiry != "" {
			os.Setenv("AF_TOKEN_EXPIRY", originalExpiry)
		} else {
			os.Unsetenv("AF_TOKEN_EXPIRY")
		}
		if originalOIDC != "" {
			os.Setenv("AF_OIDC_ENABLED", originalOIDC)
		} else {
			os.Unsetenv("AF_OIDC_ENABLED")
		}
	}()

	// Test 1: Default configuration
	fmt.Println("\n1. Testing default configuration...")
	os.Unsetenv("AF_JWT_SECRET")
	os.Unsetenv("AF_TOKEN_EXPIRY")
	os.Unsetenv("AF_OIDC_ENABLED")

	config1 := LoadAuthConfigFromEnv()
	fmt.Printf("   Default JWT secret length: %d\n", len(config1.JWTSecret))
	fmt.Printf("   Default token expiry: %s\n", config1.TokenExpiry)
	fmt.Printf("   Default OIDC enabled: %t\n", config1.OIDCEnabled)

	// Test 2: Environment overrides
	fmt.Println("\n2. Testing environment variable overrides...")
	os.Setenv("AF_JWT_SECRET", "custom-env-secret-32-chars-long")
	os.Setenv("AF_TOKEN_EXPIRY", "30m")
	os.Setenv("AF_OIDC_ENABLED", "true")

	config2 := LoadAuthConfigFromEnv()
	fmt.Printf("   Custom JWT secret: %s\n", config2.JWTSecret)
	fmt.Printf("   Custom token expiry: %s\n", config2.TokenExpiry)
	fmt.Printf("   Custom OIDC enabled: %t\n", config2.OIDCEnabled)

	require.Equal(t, "custom-env-secret-32-chars-long", config2.JWTSecret)
	require.Equal(t, 30*time.Minute, config2.TokenExpiry)
	require.True(t, config2.OIDCEnabled)

	// Test 3: Functional test with custom config
	fmt.Println("\n3. Testing functionality with custom configuration...")
	auth := NewAuthenticator(config2)

	tokenReq := &TokenRequest{
		TenantID: "env-test-tenant",
		UserID:   "env-test-user",
	}

	tokenResp, err := auth.IssueToken(context.Background(), tokenReq)
	require.NoError(t, err)
	fmt.Printf("   ✓ Token issued with custom config (expires in %d seconds)\n", tokenResp.ExpiresIn)

	claims, err := auth.ValidateToken(context.Background(), tokenResp.AccessToken)
	require.NoError(t, err)
	fmt.Printf("   ✓ Token validated with custom config (user: %s)\n", claims.UserID)

	fmt.Println("\n=== Environment Configuration Test Completed Successfully ===")
}

// Manual test instructions
func init() {
	if os.Getenv("AF_RUN_MANUAL_TESTS") == "true" {
		fmt.Println(`=== AgentFlow Authentication Manual Test Instructions ===

To run the manual authentication tests, execute:

1. Basic authentication flow:
   go test -v -run TestManualAuthenticationFlow ./internal/security

2. OIDC flag toggle behavior:
   go test -v -run TestManualOIDCFlagToggle ./internal/security

3. Token lifecycle management:
   go test -v -run TestManualTokenLifecycle ./internal/security

4. Environment configuration:
   go test -v -run TestManualEnvironmentConfiguration ./internal/security

5. All manual tests:
   AF_RUN_MANUAL_TESTS=true go test -v -run TestManual ./internal/security

These tests demonstrate:
- JWT token issuance and validation
- Authentication middleware integration
- OIDC feature flag behavior
- Token expiration and revocation
- Environment-based configuration
- Protected endpoint access
- Error handling scenarios

The tests create a temporary HTTP server and simulate real authentication flows.`)
	}
}
