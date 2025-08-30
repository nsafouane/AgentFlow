// Package security provides tenant isolation and multi-tenancy enforcement for AgentFlow
package security

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/agentflow/agentflow/internal/logging"
)

// TenantContext represents tenant-specific context information
type TenantContext struct {
	TenantID       string                 `json:"tenant_id"`
	TenantName     string                 `json:"tenant_name"`
	Permissions    []string               `json:"permissions"`
	ResourceLimits map[string]interface{} `json:"resource_limits"`
}

// TenantIsolationMiddleware provides tenant isolation and cross-tenant access prevention
type TenantIsolationMiddleware struct {
	logger logging.Logger
	db     *sql.DB // Database connection for tenant validation
}

// NewTenantIsolationMiddleware creates a new tenant isolation middleware
func NewTenantIsolationMiddleware(logger logging.Logger, db *sql.DB) *TenantIsolationMiddleware {
	return &TenantIsolationMiddleware{
		logger: logger,
		db:     db,
	}
}

// Middleware returns the HTTP middleware function for tenant isolation
func (tim *TenantIsolationMiddleware) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip tenant isolation for public endpoints
			if tim.isPublicEndpoint(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			// Extract claims from context (should be set by auth middleware)
			claims := GetClaimsFromContext(r.Context())
			if claims == nil {
				tim.writeError(w, "unauthenticated", "Authentication required for tenant isolation", http.StatusUnauthorized)
				return
			}

			// Validate tenant exists and is active
			tenantContext, err := tim.validateAndLoadTenant(r.Context(), claims.TenantID)
			if err != nil {
				tim.logger.Error("Tenant validation failed", err,
					logging.String("tenant_id", claims.TenantID),
					logging.String("user_id", claims.UserID),
					logging.String("path", r.URL.Path))

				tim.auditCrossTenantAttempt(r.Context(), claims, "tenant_validation_failed", err.Error())
				tim.writeError(w, "invalid_tenant", "Tenant validation failed", http.StatusForbidden)
				return
			}

			// Check for cross-tenant access attempts in request parameters
			if err := tim.validateRequestTenantScope(r, claims.TenantID); err != nil {
				tim.logger.Warn("Cross-tenant access attempt detected",
					logging.String("tenant_id", claims.TenantID),
					logging.String("user_id", claims.UserID),
					logging.String("path", r.URL.Path),
					logging.String("method", r.Method),
					logging.String("error", err.Error()))

				tim.auditCrossTenantAttempt(r.Context(), claims, "cross_tenant_access", err.Error())
				tim.writeError(w, "cross_tenant_access_denied", "Cross-tenant access is not permitted", http.StatusForbidden)
				return
			}

			// Add tenant context to request context
			ctx := WithTenantContext(r.Context(), tenantContext)

			// Log successful tenant isolation
			tim.logger.Debug("Request tenant isolation validated",
				logging.String("tenant_id", claims.TenantID),
				logging.String("tenant_name", tenantContext.TenantName),
				logging.String("user_id", claims.UserID),
				logging.String("path", r.URL.Path))

			// Continue with tenant-scoped request
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// validateAndLoadTenant validates tenant exists and loads tenant context
func (tim *TenantIsolationMiddleware) validateAndLoadTenant(ctx context.Context, tenantID string) (*TenantContext, error) {
	if tenantID == "" {
		return nil, fmt.Errorf("tenant_id is required")
	}

	// Query tenant from database
	query := `SELECT id, name, tier, settings FROM tenants WHERE id = $1`
	var id, name, tier string
	var settings []byte

	err := tim.db.QueryRowContext(ctx, query, tenantID).Scan(&id, &name, &tier, &settings)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("tenant not found: %s", tenantID)
		}
		return nil, fmt.Errorf("failed to query tenant: %w", err)
	}

	// Parse tenant settings
	var settingsMap map[string]interface{}
	if len(settings) > 0 {
		if err := json.Unmarshal(settings, &settingsMap); err != nil {
			tim.logger.Warn("Failed to parse tenant settings",
				logging.String("tenant_id", tenantID),
				logging.String("error", err.Error()))
			settingsMap = make(map[string]interface{})
		}
	} else {
		settingsMap = make(map[string]interface{})
	}

	// Extract resource limits from settings
	resourceLimits := make(map[string]interface{})
	if limits, ok := settingsMap["resource_limits"].(map[string]interface{}); ok {
		resourceLimits = limits
	}

	return &TenantContext{
		TenantID:       id,
		TenantName:     name,
		Permissions:    []string{}, // Will be populated from RBAC
		ResourceLimits: resourceLimits,
	}, nil
}

// validateRequestTenantScope checks for cross-tenant access attempts in request
func (tim *TenantIsolationMiddleware) validateRequestTenantScope(r *http.Request, userTenantID string) error {
	// Check query parameters for tenant_id
	if queryTenantID := r.URL.Query().Get("tenant_id"); queryTenantID != "" {
		if queryTenantID != userTenantID {
			return fmt.Errorf("query parameter tenant_id (%s) does not match user tenant (%s)", queryTenantID, userTenantID)
		}
	}

	// Check for tenant_id in request headers
	if headerTenantID := r.Header.Get("X-Tenant-ID"); headerTenantID != "" {
		if headerTenantID != userTenantID {
			return fmt.Errorf("header X-Tenant-ID (%s) does not match user tenant (%s)", headerTenantID, userTenantID)
		}
	}

	// Check URL path for tenant IDs (common pattern: /api/v1/tenants/{tenant_id}/...)
	pathParts := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	for i, part := range pathParts {
		if part == "tenants" && i+1 < len(pathParts) {
			pathTenantID := pathParts[i+1]
			// Skip if it's not a UUID-like string (could be "current" or other keywords)
			if len(pathTenantID) == 36 && strings.Count(pathTenantID, "-") == 4 {
				if pathTenantID != userTenantID {
					return fmt.Errorf("path tenant_id (%s) does not match user tenant (%s)", pathTenantID, userTenantID)
				}
			}
		}
	}

	return nil
}

// auditCrossTenantAttempt logs cross-tenant access attempts for security monitoring
func (tim *TenantIsolationMiddleware) auditCrossTenantAttempt(ctx context.Context, claims *AgentFlowClaims, attemptType, details string) {
	// Create audit entry in database
	auditQuery := `
		INSERT INTO audits (tenant_id, actor_type, actor_id, action, resource_type, resource_id, details, prev_hash, hash)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	auditDetails := map[string]interface{}{
		"attempt_type": attemptType,
		"details":      details,
		"user_id":      claims.UserID,
		"roles":        claims.Roles,
		"timestamp":    "now", // Will be replaced by database timestamp
	}

	detailsJSON, _ := json.Marshal(auditDetails)

	// For now, use placeholder hash values - in production this would use the hash chain
	prevHash := []byte{}
	hash := []byte("placeholder_hash")

	_, err := tim.db.ExecContext(ctx, auditQuery,
		claims.TenantID,
		"user",
		claims.UserID,
		"cross_tenant_access_attempt",
		"security_violation",
		"",
		detailsJSON,
		prevHash,
		hash)

	if err != nil {
		tim.logger.Error("Failed to audit cross-tenant access attempt", err,
			logging.String("tenant_id", claims.TenantID),
			logging.String("user_id", claims.UserID),
			logging.String("attempt_type", attemptType))
	}
}

// isPublicEndpoint checks if an endpoint should skip tenant isolation
func (tim *TenantIsolationMiddleware) isPublicEndpoint(path string) bool {
	publicEndpoints := []string{
		"/health",
		"/api/v1/health",
		"/",
		"/api",
		"/api/v1/auth/token",
		"/api/v1/auth/validate",
	}

	for _, endpoint := range publicEndpoints {
		if path == endpoint {
			return true
		}
	}

	return false
}

// writeError writes a structured error response
func (tim *TenantIsolationMiddleware) writeError(w http.ResponseWriter, code, message string, statusCode int) {
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

// WithTenantContext adds tenant context to the request context
func WithTenantContext(ctx context.Context, tenant *TenantContext) context.Context {
	return context.WithValue(ctx, "tenant_context", tenant)
}

// GetTenantContext extracts tenant context from request context
func GetTenantContext(ctx context.Context) (*TenantContext, error) {
	if tenant, ok := ctx.Value("tenant_context").(*TenantContext); ok {
		return tenant, nil
	}
	return nil, fmt.Errorf("tenant context not found")
}

// GetTenantIDFromTenantContext extracts tenant ID from tenant context
func GetTenantIDFromTenantContext(ctx context.Context) (string, error) {
	tenant, err := GetTenantContext(ctx)
	if err != nil {
		return "", err
	}
	return tenant.TenantID, nil
}

// MustGetTenantID extracts tenant ID from context or panics
func MustGetTenantID(ctx context.Context) string {
	// First try to get from tenant context
	if tenantID, err := GetTenantIDFromTenantContext(ctx); err == nil {
		return tenantID
	}

	// Fallback to auth claims context
	if tenantID := GetTenantIDFromContext(ctx); tenantID != "" {
		return tenantID
	}

	panic("tenant ID not found in context")
}
