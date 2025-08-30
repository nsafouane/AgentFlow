// Package security provides authentication middleware for AgentFlow
package security

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/agentflow/agentflow/internal/logging"
)

// AuthMiddleware provides JWT authentication middleware
type AuthMiddleware struct {
	authenticator Authenticator
	logger        logging.Logger
	config        *AuthConfig
}

// NewAuthMiddleware creates a new authentication middleware
func NewAuthMiddleware(authenticator Authenticator, logger logging.Logger, config *AuthConfig) *AuthMiddleware {
	if config == nil {
		config = LoadAuthConfigFromEnv()
	}

	return &AuthMiddleware{
		authenticator: authenticator,
		logger:        logger,
		config:        config,
	}
}

// Middleware returns the HTTP middleware function
func (am *AuthMiddleware) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip authentication for health check and public endpoints
			if am.isPublicEndpoint(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			// Extract token from Authorization header
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				am.writeAuthError(w, "missing_authorization_header", "Authorization header is required", http.StatusUnauthorized)
				return
			}

			token, err := ExtractTokenFromHeader(authHeader)
			if err != nil {
				am.writeAuthError(w, "invalid_authorization_header", err.Error(), http.StatusUnauthorized)
				return
			}

			// Validate token
			claims, err := am.authenticator.ValidateToken(r.Context(), token)
			if err != nil {
				am.logger.Warn("Token validation failed",
					logging.String("error", err.Error()),
					logging.String("path", r.URL.Path),
					logging.String("method", r.Method),
					logging.String("remote_addr", r.RemoteAddr),
				)
				am.writeAuthError(w, "invalid_token", "Token validation failed", http.StatusUnauthorized)
				return
			}

			// Add claims to request context
			ctx := context.WithValue(r.Context(), "auth_claims", claims)
			ctx = context.WithValue(ctx, "tenant_id", claims.TenantID)
			ctx = context.WithValue(ctx, "user_id", claims.UserID)
			ctx = context.WithValue(ctx, "user_roles", claims.Roles)
			ctx = context.WithValue(ctx, "user_permissions", claims.Permissions)

			// Log successful authentication
			am.logger.Info("Request authenticated",
				logging.String("user_id", claims.UserID),
				logging.String("tenant_id", claims.TenantID),
				logging.Any("roles", claims.Roles),
				logging.String("path", r.URL.Path),
				logging.String("method", r.Method),
			)

			// Continue with authenticated request
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole creates middleware that requires specific roles
func (am *AuthMiddleware) RequireRole(requiredRoles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := GetClaimsFromContext(r.Context())
			if claims == nil {
				am.writeAuthError(w, "unauthenticated", "Authentication required", http.StatusUnauthorized)
				return
			}

			// Check if user has any of the required roles
			hasRole := false
			for _, userRole := range claims.Roles {
				for _, requiredRole := range requiredRoles {
					if userRole == requiredRole {
						hasRole = true
						break
					}
				}
				if hasRole {
					break
				}
			}

			if !hasRole {
				am.logger.Warn("Insufficient role permissions",
					logging.String("user_id", claims.UserID),
					logging.String("tenant_id", claims.TenantID),
					logging.Any("user_roles", claims.Roles),
					logging.Any("required_roles", requiredRoles),
					logging.String("path", r.URL.Path),
				)
				am.writeAuthError(w, "insufficient_permissions", "Insufficient role permissions", http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// RequirePermission creates middleware that requires specific permissions
func (am *AuthMiddleware) RequirePermission(resource, action string) func(http.Handler) http.Handler {
	requiredPermission := resource + ":" + action

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claims := GetClaimsFromContext(r.Context())
			if claims == nil {
				am.writeAuthError(w, "unauthenticated", "Authentication required", http.StatusUnauthorized)
				return
			}

			// Check if user has the required permission or wildcard permission
			hasPermission := false
			resourceWildcard := resource + ":*"

			for _, permission := range claims.Permissions {
				if permission == requiredPermission || permission == resourceWildcard || permission == "*" {
					hasPermission = true
					break
				}
			}

			if !hasPermission {
				am.logger.Warn("Insufficient permissions",
					logging.String("user_id", claims.UserID),
					logging.String("tenant_id", claims.TenantID),
					logging.Any("user_permissions", claims.Permissions),
					logging.String("required_permission", requiredPermission),
					logging.String("path", r.URL.Path),
				)
				am.writeAuthError(w, "insufficient_permissions",
					"User lacks required permission: "+requiredPermission, http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// isPublicEndpoint checks if an endpoint should skip authentication
func (am *AuthMiddleware) isPublicEndpoint(path string) bool {
	publicEndpoints := []string{
		"/health",
		"/api/v1/health",
		"/",
		"/api",
		"/api/v1/auth/token", // Token issuance endpoint
	}

	for _, endpoint := range publicEndpoints {
		if path == endpoint {
			return true
		}
	}

	return false
}

// writeAuthError writes a structured authentication error response
func (am *AuthMiddleware) writeAuthError(w http.ResponseWriter, code, message string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	errorResponse := map[string]interface{}{
		"success": false,
		"error": map[string]interface{}{
			"code":    code,
			"message": message,
		},
	}

	json.NewEncoder(w).Encode(errorResponse)
}

// Context helper functions

// GetClaimsFromContext extracts AgentFlow claims from request context
func GetClaimsFromContext(ctx context.Context) *AgentFlowClaims {
	if claims, ok := ctx.Value("auth_claims").(*AgentFlowClaims); ok {
		return claims
	}
	return nil
}

// GetTenantIDFromContext extracts tenant ID from request context
func GetTenantIDFromContext(ctx context.Context) string {
	if tenantID, ok := ctx.Value("tenant_id").(string); ok {
		return tenantID
	}
	return ""
}

// GetUserIDFromContext extracts user ID from request context
func GetUserIDFromContext(ctx context.Context) string {
	if userID, ok := ctx.Value("user_id").(string); ok {
		return userID
	}
	return ""
}

// GetUserRolesFromContext extracts user roles from request context
func GetUserRolesFromContext(ctx context.Context) []string {
	if roles, ok := ctx.Value("user_roles").([]string); ok {
		return roles
	}
	return []string{}
}

// GetUserPermissionsFromContext extracts user permissions from request context
func GetUserPermissionsFromContext(ctx context.Context) []string {
	if permissions, ok := ctx.Value("user_permissions").([]string); ok {
		return permissions
	}
	return []string{}
}

// HasRole checks if the current user has a specific role
func HasRole(ctx context.Context, role string) bool {
	roles := GetUserRolesFromContext(ctx)
	for _, userRole := range roles {
		if userRole == role {
			return true
		}
	}
	return false
}

// HasPermission checks if the current user has a specific permission
func HasPermission(ctx context.Context, resource, action string) bool {
	permissions := GetUserPermissionsFromContext(ctx)
	requiredPermission := resource + ":" + action
	resourceWildcard := resource + ":*"

	for _, permission := range permissions {
		if permission == requiredPermission || permission == resourceWildcard || permission == "*" {
			return true
		}
	}
	return false
}
