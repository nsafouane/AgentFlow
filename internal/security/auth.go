// Package security provides security utilities and authentication for AgentFlow
package security

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Authenticator provides authentication interface
type Authenticator interface {
	ValidateToken(ctx context.Context, token string) (*AgentFlowClaims, error)
	IssueToken(ctx context.Context, req *TokenRequest) (*TokenResponse, error)
	RefreshToken(ctx context.Context, refreshToken string) (*TokenResponse, error)
	RevokeToken(ctx context.Context, token string) error
}

// AgentFlowClaims represents JWT claims for AgentFlow
type AgentFlowClaims struct {
	TenantID    string   `json:"tenant_id"`
	UserID      string   `json:"user_id"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
	jwt.RegisteredClaims
}

// TokenRequest represents a token issuance request
type TokenRequest struct {
	TenantID    string   `json:"tenant_id"`
	UserID      string   `json:"user_id"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
	ExpiresIn   int64    `json:"expires_in,omitempty"` // seconds, optional
}

// TokenResponse represents a token response
type TokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token,omitempty"`
	IssuedAt     int64  `json:"issued_at"`
}

// AuthConfig holds authentication configuration
type AuthConfig struct {
	JWTSecret        string        `env:"AF_JWT_SECRET"`
	TokenExpiry      time.Duration `env:"AF_TOKEN_EXPIRY"`
	RefreshExpiry    time.Duration `env:"AF_REFRESH_TOKEN_EXPIRY"`
	OIDCEnabled      bool          `env:"AF_OIDC_ENABLED"`
	OIDCIssuer       string        `env:"AF_OIDC_ISSUER"`
	OIDCClientID     string        `env:"AF_OIDC_CLIENT_ID"`
	OIDCClientSecret string        `env:"AF_OIDC_CLIENT_SECRET"`
}

// DefaultAuthConfig returns default authentication configuration
func DefaultAuthConfig() *AuthConfig {
	return &AuthConfig{
		JWTSecret:     generateDefaultSecret(),
		TokenExpiry:   24 * time.Hour,
		RefreshExpiry: 7 * 24 * time.Hour,
		OIDCEnabled:   false,
	}
}

// LoadAuthConfigFromEnv loads authentication configuration from environment
func LoadAuthConfigFromEnv() *AuthConfig {
	config := DefaultAuthConfig()

	if val := os.Getenv("AF_JWT_SECRET"); val != "" {
		config.JWTSecret = val
	}

	if val := os.Getenv("AF_TOKEN_EXPIRY"); val != "" {
		if duration, err := time.ParseDuration(val); err == nil {
			config.TokenExpiry = duration
		}
	}

	if val := os.Getenv("AF_REFRESH_TOKEN_EXPIRY"); val != "" {
		if duration, err := time.ParseDuration(val); err == nil {
			config.RefreshExpiry = duration
		}
	}

	if val := os.Getenv("AF_OIDC_ENABLED"); val != "" {
		config.OIDCEnabled = val == "true" || val == "1"
	}

	if val := os.Getenv("AF_OIDC_ISSUER"); val != "" {
		config.OIDCIssuer = val
	}

	if val := os.Getenv("AF_OIDC_CLIENT_ID"); val != "" {
		config.OIDCClientID = val
	}

	if val := os.Getenv("AF_OIDC_CLIENT_SECRET"); val != "" {
		config.OIDCClientSecret = val
	}

	return config
}

// jwtAuthenticator implements JWT-based authentication
type jwtAuthenticator struct {
	config        *AuthConfig
	revokedTokens map[string]time.Time // Simple in-memory revocation store
}

// NewAuthenticator creates a new JWT authenticator
func NewAuthenticator(config *AuthConfig) Authenticator {
	if config == nil {
		config = LoadAuthConfigFromEnv()
	}

	return &jwtAuthenticator{
		config:        config,
		revokedTokens: make(map[string]time.Time),
	}
}

// ValidateToken validates a JWT token and returns claims
func (a *jwtAuthenticator) ValidateToken(ctx context.Context, tokenString string) (*AgentFlowClaims, error) {
	// Parse and validate the token
	token, err := jwt.ParseWithClaims(tokenString, &AgentFlowClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Validate signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(a.config.JWTSecret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	// Extract claims
	claims, ok := token.Claims.(*AgentFlowClaims)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token claims")
	}

	// Check if token is revoked
	if revokedAt, exists := a.revokedTokens[tokenString]; exists {
		if claims.IssuedAt.Before(revokedAt) {
			return nil, errors.New("token has been revoked")
		}
	}

	// Validate required claims
	if claims.TenantID == "" {
		return nil, errors.New("missing tenant_id claim")
	}

	if claims.UserID == "" {
		return nil, errors.New("missing user_id claim")
	}

	return claims, nil
}

// IssueToken issues a new JWT token
func (a *jwtAuthenticator) IssueToken(ctx context.Context, req *TokenRequest) (*TokenResponse, error) {
	if req.TenantID == "" {
		return nil, errors.New("tenant_id is required")
	}

	if req.UserID == "" {
		return nil, errors.New("user_id is required")
	}

	now := time.Now()
	expiry := req.ExpiresIn
	if expiry == 0 {
		expiry = int64(a.config.TokenExpiry.Seconds())
	}

	// Create claims
	claims := &AgentFlowClaims{
		TenantID:    req.TenantID,
		UserID:      req.UserID,
		Roles:       req.Roles,
		Permissions: req.Permissions,
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        uuid.New().String(),
			Subject:   req.UserID,
			Audience:  []string{"agentflow"},
			Issuer:    "agentflow-control-plane",
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(time.Duration(expiry) * time.Second)),
			NotBefore: jwt.NewNumericDate(now),
		},
	}

	// Create token
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString([]byte(a.config.JWTSecret))
	if err != nil {
		return nil, fmt.Errorf("failed to sign token: %w", err)
	}

	// Generate refresh token
	refreshToken := generateRefreshToken()

	return &TokenResponse{
		AccessToken:  tokenString,
		TokenType:    "Bearer",
		ExpiresIn:    expiry,
		RefreshToken: refreshToken,
		IssuedAt:     now.Unix(),
	}, nil
}

// RefreshToken refreshes an access token using a refresh token
func (a *jwtAuthenticator) RefreshToken(ctx context.Context, refreshToken string) (*TokenResponse, error) {
	// In a real implementation, you would validate the refresh token
	// against a database or cache. For now, we'll return an error.
	return nil, errors.New("refresh token functionality not implemented")
}

// RevokeToken revokes a token
func (a *jwtAuthenticator) RevokeToken(ctx context.Context, token string) error {
	// Add token to revocation list with current timestamp
	a.revokedTokens[token] = time.Now()

	// In a production system, you would persist this to a database
	// and implement cleanup for expired tokens

	return nil
}

// generateDefaultSecret generates a default JWT secret for development
func generateDefaultSecret() string {
	// In development, use a fixed secret for consistency
	if os.Getenv("AF_ENV") == "development" {
		return "dev-secret-change-in-production-32-chars"
	}

	// Generate a random secret
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to a default secret if random generation fails
		return "fallback-secret-change-in-production"
	}
	return hex.EncodeToString(bytes)
}

// generateRefreshToken generates a random refresh token
func generateRefreshToken() string {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return uuid.New().String()
	}
	return hex.EncodeToString(bytes)
}

// ExtractTokenFromHeader extracts JWT token from Authorization header
func ExtractTokenFromHeader(authHeader string) (string, error) {
	if authHeader == "" {
		return "", errors.New("authorization header is required")
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 {
		return "", errors.New("invalid authorization header format")
	}

	if strings.ToLower(parts[0]) != "bearer" {
		return "", errors.New("authorization header must use Bearer scheme")
	}

	return parts[1], nil
}

// Principal represents an authenticated user or service (for backward compatibility)
type Principal struct {
	ID       string
	TenantID string
	Roles    []string
}

// ToPrincipal converts AgentFlowClaims to Principal for backward compatibility
func (c *AgentFlowClaims) ToPrincipal() *Principal {
	return &Principal{
		ID:       c.UserID,
		TenantID: c.TenantID,
		Roles:    c.Roles,
	}
}
