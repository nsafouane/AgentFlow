// Package security provides OIDC integration for AgentFlow
package security

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// OIDCProvider represents an OIDC identity provider
type OIDCProvider interface {
	ValidateToken(ctx context.Context, token string) (*AgentFlowClaims, error)
	GetProviderInfo(ctx context.Context) (*OIDCProviderInfo, error)
}

// OIDCProviderInfo contains OIDC provider metadata
type OIDCProviderInfo struct {
	Issuer                 string   `json:"issuer"`
	AuthorizationEndpoint  string   `json:"authorization_endpoint"`
	TokenEndpoint          string   `json:"token_endpoint"`
	JWKSUri                string   `json:"jwks_uri"`
	SupportedScopes        []string `json:"scopes_supported"`
	SupportedResponseTypes []string `json:"response_types_supported"`
}

// OIDCClaims represents standard OIDC claims
type OIDCClaims struct {
	Subject           string   `json:"sub"`
	Name              string   `json:"name"`
	Email             string   `json:"email"`
	EmailVerified     bool     `json:"email_verified"`
	PreferredUsername string   `json:"preferred_username"`
	Groups            []string `json:"groups"`
	Roles             []string `json:"roles"`
	TenantID          string   `json:"tenant_id"`
	jwt.RegisteredClaims
}

// oidcProvider implements OIDC token validation
type oidcProvider struct {
	config       *AuthConfig
	providerInfo *OIDCProviderInfo
	httpClient   *http.Client
}

// NewOIDCProvider creates a new OIDC provider
func NewOIDCProvider(config *AuthConfig) (OIDCProvider, error) {
	if !config.OIDCEnabled {
		return nil, errors.New("OIDC is not enabled")
	}

	if config.OIDCIssuer == "" {
		return nil, errors.New("OIDC issuer is required")
	}

	provider := &oidcProvider{
		config: config,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}

	// Discover provider information
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	providerInfo, err := provider.discoverProvider(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to discover OIDC provider: %w", err)
	}

	provider.providerInfo = providerInfo
	return provider, nil
}

// ValidateToken validates an OIDC token
func (p *oidcProvider) ValidateToken(ctx context.Context, tokenString string) (*AgentFlowClaims, error) {
	// Parse token without verification first to get the header
	token, _, err := new(jwt.Parser).ParseUnverified(tokenString, &OIDCClaims{})
	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	// Get the key ID from token header
	keyID, ok := token.Header["kid"].(string)
	if !ok {
		return nil, errors.New("token missing key ID")
	}

	// Get the public key for verification
	publicKey, err := p.getPublicKey(ctx, keyID)
	if err != nil {
		return nil, fmt.Errorf("failed to get public key: %w", err)
	}

	// Parse and validate token with public key
	validatedToken, err := jwt.ParseWithClaims(tokenString, &OIDCClaims{}, func(token *jwt.Token) (interface{}, error) {
		return publicKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("token validation failed: %w", err)
	}

	// Extract OIDC claims
	oidcClaims, ok := validatedToken.Claims.(*OIDCClaims)
	if !ok || !validatedToken.Valid {
		return nil, errors.New("invalid token claims")
	}

	// Validate issuer
	if oidcClaims.Issuer != p.config.OIDCIssuer {
		return nil, fmt.Errorf("invalid issuer: expected %s, got %s", p.config.OIDCIssuer, oidcClaims.Issuer)
	}

	// Convert OIDC claims to AgentFlow claims
	agentFlowClaims := &AgentFlowClaims{
		TenantID:    oidcClaims.TenantID,
		UserID:      oidcClaims.Subject,
		Roles:       oidcClaims.Roles,
		Permissions: []string{}, // OIDC doesn't typically include permissions
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        oidcClaims.ID,
			Subject:   oidcClaims.Subject,
			Audience:  oidcClaims.Audience,
			Issuer:    oidcClaims.Issuer,
			IssuedAt:  oidcClaims.IssuedAt,
			ExpiresAt: oidcClaims.ExpiresAt,
			NotBefore: oidcClaims.NotBefore,
		},
	}

	// If no tenant ID in token, use a default or derive from email domain
	if agentFlowClaims.TenantID == "" {
		agentFlowClaims.TenantID = p.deriveTenantFromEmail(oidcClaims.Email)
	}

	// If no roles in token, assign default role
	if len(agentFlowClaims.Roles) == 0 {
		agentFlowClaims.Roles = []string{"viewer"}
	}

	return agentFlowClaims, nil
}

// GetProviderInfo returns OIDC provider information
func (p *oidcProvider) GetProviderInfo(ctx context.Context) (*OIDCProviderInfo, error) {
	if p.providerInfo == nil {
		return p.discoverProvider(ctx)
	}
	return p.providerInfo, nil
}

// discoverProvider discovers OIDC provider configuration
func (p *oidcProvider) discoverProvider(ctx context.Context) (*OIDCProviderInfo, error) {
	// Construct well-known configuration URL
	configURL := strings.TrimSuffix(p.config.OIDCIssuer, "/") + "/.well-known/openid_configuration"

	req, err := http.NewRequestWithContext(ctx, "GET", configURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch provider configuration: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("provider configuration request failed with status: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var providerInfo OIDCProviderInfo
	if err := json.Unmarshal(body, &providerInfo); err != nil {
		return nil, fmt.Errorf("failed to parse provider configuration: %w", err)
	}

	return &providerInfo, nil
}

// getPublicKey retrieves the public key for token verification
func (p *oidcProvider) getPublicKey(ctx context.Context, keyID string) (interface{}, error) {
	// In a real implementation, you would:
	// 1. Fetch JWKS from the provider's JWKS URI
	// 2. Parse the keys and find the one with matching key ID
	// 3. Convert to appropriate format for verification
	// 4. Cache keys for performance

	// For now, return an error indicating this needs implementation
	return nil, errors.New("OIDC public key retrieval not implemented - requires JWKS integration")
}

// deriveTenantFromEmail derives tenant ID from email domain
func (p *oidcProvider) deriveTenantFromEmail(email string) string {
	if email == "" {
		return "default"
	}

	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return "default"
	}

	// Use domain as tenant ID (sanitized)
	domain := strings.ToLower(parts[1])
	domain = strings.ReplaceAll(domain, ".", "-")
	return domain
}

// hybridAuthenticator combines JWT and OIDC authentication
type hybridAuthenticator struct {
	jwtAuth  *jwtAuthenticator
	oidcAuth OIDCProvider
	config   *AuthConfig
}

// NewHybridAuthenticator creates an authenticator that supports both JWT and OIDC
func NewHybridAuthenticator(config *AuthConfig) (Authenticator, error) {
	if config == nil {
		config = LoadAuthConfigFromEnv()
	}

	jwtAuth := &jwtAuthenticator{
		config:        config,
		revokedTokens: make(map[string]time.Time),
	}

	hybrid := &hybridAuthenticator{
		jwtAuth: jwtAuth,
		config:  config,
	}

	// Initialize OIDC provider if enabled
	if config.OIDCEnabled {
		oidcAuth, err := NewOIDCProvider(config)
		if err != nil {
			// Log warning but don't fail - fall back to JWT only
			fmt.Printf("Warning: Failed to initialize OIDC provider: %v\n", err)
		} else {
			hybrid.oidcAuth = oidcAuth
		}
	}

	return hybrid, nil
}

// ValidateToken validates a token using OIDC if enabled, otherwise JWT
func (h *hybridAuthenticator) ValidateToken(ctx context.Context, token string) (*AgentFlowClaims, error) {
	// Try OIDC validation first if enabled and provider is available
	if h.config.OIDCEnabled && h.oidcAuth != nil {
		claims, err := h.oidcAuth.ValidateToken(ctx, token)
		if err == nil {
			return claims, nil
		}
		// Log OIDC validation failure but continue to JWT fallback
		fmt.Printf("OIDC validation failed, falling back to JWT: %v\n", err)
	}

	// Fall back to JWT validation
	return h.jwtAuth.ValidateToken(ctx, token)
}

// IssueToken issues a JWT token (OIDC doesn't issue tokens)
func (h *hybridAuthenticator) IssueToken(ctx context.Context, req *TokenRequest) (*TokenResponse, error) {
	return h.jwtAuth.IssueToken(ctx, req)
}

// RefreshToken refreshes a token
func (h *hybridAuthenticator) RefreshToken(ctx context.Context, refreshToken string) (*TokenResponse, error) {
	return h.jwtAuth.RefreshToken(ctx, refreshToken)
}

// RevokeToken revokes a token
func (h *hybridAuthenticator) RevokeToken(ctx context.Context, token string) error {
	return h.jwtAuth.RevokeToken(ctx, token)
}
