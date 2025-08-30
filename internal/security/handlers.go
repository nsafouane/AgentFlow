// Package security provides authentication HTTP handlers for AgentFlow
package security

import (
	"encoding/json"
	"net/http"
	"time"

	"github.com/agentflow/agentflow/internal/logging"
)

// AuthHandlers provides HTTP handlers for authentication endpoints
type AuthHandlers struct {
	authenticator Authenticator
	logger        logging.Logger
}

// NewAuthHandlers creates new authentication handlers
func NewAuthHandlers(authenticator Authenticator, logger logging.Logger) *AuthHandlers {
	return &AuthHandlers{
		authenticator: authenticator,
		logger:        logger,
	}
}

// TokenRequest represents the request body for token issuance
type TokenIssueRequest struct {
	TenantID    string   `json:"tenant_id"`
	UserID      string   `json:"user_id"`
	Roles       []string `json:"roles,omitempty"`
	Permissions []string `json:"permissions,omitempty"`
	ExpiresIn   string   `json:"expires_in,omitempty"` // Duration string like "24h"
}

// HandleTokenIssue handles POST /api/v1/auth/token
func (ah *AuthHandlers) HandleTokenIssue(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		ah.writeError(w, "method_not_allowed", "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var req TokenIssueRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		ah.writeError(w, "invalid_request", "Invalid JSON request body", http.StatusBadRequest)
		return
	}

	// Validate required fields
	if req.TenantID == "" {
		ah.writeError(w, "missing_tenant_id", "tenant_id is required", http.StatusBadRequest)
		return
	}

	if req.UserID == "" {
		ah.writeError(w, "missing_user_id", "user_id is required", http.StatusBadRequest)
		return
	}

	// Parse expires_in duration
	var expiresIn int64
	if req.ExpiresIn != "" {
		duration, err := time.ParseDuration(req.ExpiresIn)
		if err != nil {
			ah.writeError(w, "invalid_expires_in", "Invalid expires_in duration format", http.StatusBadRequest)
			return
		}
		expiresIn = int64(duration.Seconds())
	}

	// Set default roles if none provided
	if len(req.Roles) == 0 {
		req.Roles = []string{"viewer"}
	}

	// Create token request
	tokenReq := &TokenRequest{
		TenantID:    req.TenantID,
		UserID:      req.UserID,
		Roles:       req.Roles,
		Permissions: req.Permissions,
		ExpiresIn:   expiresIn,
	}

	// Issue token
	tokenResp, err := ah.authenticator.IssueToken(r.Context(), tokenReq)
	if err != nil {
		ah.logger.Error("Failed to issue token", err,
			logging.String("tenant_id", req.TenantID),
			logging.String("user_id", req.UserID),
		)
		ah.writeError(w, "token_issuance_failed", "Failed to issue token", http.StatusInternalServerError)
		return
	}

	// Log successful token issuance
	ah.logger.Info("Token issued successfully",
		logging.String("tenant_id", req.TenantID),
		logging.String("user_id", req.UserID),
		logging.Any("roles", req.Roles),
		logging.Any("expires_in", tokenResp.ExpiresIn),
	)

	// Return token response
	ah.writeSuccess(w, tokenResp)
}

// HandleTokenValidate handles POST /api/v1/auth/validate
func (ah *AuthHandlers) HandleTokenValidate(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		ah.writeError(w, "method_not_allowed", "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract token from Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		ah.writeError(w, "missing_authorization_header", "Authorization header is required", http.StatusUnauthorized)
		return
	}

	token, err := ExtractTokenFromHeader(authHeader)
	if err != nil {
		ah.writeError(w, "invalid_authorization_header", err.Error(), http.StatusUnauthorized)
		return
	}

	// Validate token
	claims, err := ah.authenticator.ValidateToken(r.Context(), token)
	if err != nil {
		ah.logger.Warn("Token validation failed",
			logging.String("error", err.Error()),
		)
		ah.writeError(w, "invalid_token", "Token validation failed", http.StatusUnauthorized)
		return
	}

	// Return validation result
	validationResult := map[string]interface{}{
		"valid":       true,
		"tenant_id":   claims.TenantID,
		"user_id":     claims.UserID,
		"roles":       claims.Roles,
		"permissions": claims.Permissions,
		"expires_at":  claims.ExpiresAt.Unix(),
		"issued_at":   claims.IssuedAt.Unix(),
	}

	ah.writeSuccess(w, validationResult)
}

// HandleTokenRevoke handles POST /api/v1/auth/revoke
func (ah *AuthHandlers) HandleTokenRevoke(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		ah.writeError(w, "method_not_allowed", "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract token from Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		ah.writeError(w, "missing_authorization_header", "Authorization header is required", http.StatusUnauthorized)
		return
	}

	token, err := ExtractTokenFromHeader(authHeader)
	if err != nil {
		ah.writeError(w, "invalid_authorization_header", err.Error(), http.StatusUnauthorized)
		return
	}

	// Revoke token
	if err := ah.authenticator.RevokeToken(r.Context(), token); err != nil {
		ah.logger.Error("Failed to revoke token", err)
		ah.writeError(w, "revocation_failed", "Failed to revoke token", http.StatusInternalServerError)
		return
	}

	ah.logger.Info("Token revoked successfully")

	// Return success response
	ah.writeSuccess(w, map[string]interface{}{
		"revoked": true,
		"message": "Token revoked successfully",
	})
}

// HandleUserInfo handles GET /api/v1/auth/userinfo
func (ah *AuthHandlers) HandleUserInfo(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		ah.writeError(w, "method_not_allowed", "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get claims from context (set by auth middleware)
	claims := GetClaimsFromContext(r.Context())
	if claims == nil {
		ah.writeError(w, "unauthenticated", "Authentication required", http.StatusUnauthorized)
		return
	}

	// Return user information
	userInfo := map[string]interface{}{
		"user_id":     claims.UserID,
		"tenant_id":   claims.TenantID,
		"roles":       claims.Roles,
		"permissions": claims.Permissions,
		"issued_at":   claims.IssuedAt.Unix(),
		"expires_at":  claims.ExpiresAt.Unix(),
	}

	ah.writeSuccess(w, userInfo)
}

// Helper methods

func (ah *AuthHandlers) writeSuccess(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := map[string]interface{}{
		"success": true,
		"data":    data,
	}

	json.NewEncoder(w).Encode(response)
}

func (ah *AuthHandlers) writeError(w http.ResponseWriter, code, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	response := map[string]interface{}{
		"success": false,
		"error": map[string]interface{}{
			"code":    code,
			"message": message,
		},
	}

	json.NewEncoder(w).Encode(response)
}

// RegisterAuthRoutes registers authentication routes with a router
func RegisterAuthRoutes(router interface{}, handlers *AuthHandlers, authMiddleware *AuthMiddleware) {
	// This is a helper function to register routes
	// The actual implementation depends on the router type (gorilla/mux, etc.)
	// For now, we'll provide the handler functions that can be registered manually
}

// GetAuthEndpoints returns a map of authentication endpoints and their handlers
func GetAuthEndpoints(handlers *AuthHandlers, authMiddleware *AuthMiddleware) map[string]http.HandlerFunc {
	return map[string]http.HandlerFunc{
		"POST /api/v1/auth/token":    handlers.HandleTokenIssue,
		"POST /api/v1/auth/validate": handlers.HandleTokenValidate,
		"POST /api/v1/auth/revoke":   handlers.HandleTokenRevoke,
		"GET /api/v1/auth/userinfo":  authMiddleware.Middleware()(http.HandlerFunc(handlers.HandleUserInfo)).ServeHTTP,
	}
}
