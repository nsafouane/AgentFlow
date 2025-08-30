// Package messaging provides tenant-aware subject building for AgentFlow message bus
package messaging

import (
	"context"
	"fmt"
	"strings"
)

// TenantSubjectBuilder provides utilities for building tenant-scoped NATS subjects
type TenantSubjectBuilder struct {
	baseBuilder *SubjectBuilder
}

// NewTenantSubjectBuilder creates a new tenant-aware subject builder
func NewTenantSubjectBuilder() *TenantSubjectBuilder {
	return &TenantSubjectBuilder{
		baseBuilder: NewSubjectBuilder(),
	}
}

// TenantWorkflowIn builds a tenant-scoped workflow inbound subject
// Pattern: {tenant_id}.workflows.{workflow_id}.in
func (tsb *TenantSubjectBuilder) TenantWorkflowIn(tenantID, workflowID string) string {
	return fmt.Sprintf("%s.workflows.%s.in", tenantID, workflowID)
}

// TenantWorkflowOut builds a tenant-scoped workflow outbound subject
// Pattern: {tenant_id}.workflows.{workflow_id}.out
func (tsb *TenantSubjectBuilder) TenantWorkflowOut(tenantID, workflowID string) string {
	return fmt.Sprintf("%s.workflows.%s.out", tenantID, workflowID)
}

// TenantAgentIn builds a tenant-scoped agent inbound subject
// Pattern: {tenant_id}.agents.{agent_id}.in
func (tsb *TenantSubjectBuilder) TenantAgentIn(tenantID, agentID string) string {
	return fmt.Sprintf("%s.agents.%s.in", tenantID, agentID)
}

// TenantAgentOut builds a tenant-scoped agent outbound subject
// Pattern: {tenant_id}.agents.{agent_id}.out
func (tsb *TenantSubjectBuilder) TenantAgentOut(tenantID, agentID string) string {
	return fmt.Sprintf("%s.agents.%s.out", tenantID, agentID)
}

// TenantToolsCalls builds a tenant-scoped tools calls subject
// Pattern: {tenant_id}.tools.calls
func (tsb *TenantSubjectBuilder) TenantToolsCalls(tenantID string) string {
	return fmt.Sprintf("%s.tools.calls", tenantID)
}

// TenantToolsAudit builds a tenant-scoped tools audit subject
// Pattern: {tenant_id}.tools.audit
func (tsb *TenantSubjectBuilder) TenantToolsAudit(tenantID string) string {
	return fmt.Sprintf("%s.tools.audit", tenantID)
}

// TenantSystemControl builds a tenant-scoped system control subject
// Pattern: {tenant_id}.system.control
func (tsb *TenantSubjectBuilder) TenantSystemControl(tenantID string) string {
	return fmt.Sprintf("%s.system.control", tenantID)
}

// TenantSystemHealth builds a tenant-scoped system health subject
// Pattern: {tenant_id}.system.health
func (tsb *TenantSubjectBuilder) TenantSystemHealth(tenantID string) string {
	return fmt.Sprintf("%s.system.health", tenantID)
}

// Context-aware subject builders that extract tenant ID from context

// WorkflowInFromContext builds a tenant-scoped workflow inbound subject using context
func (tsb *TenantSubjectBuilder) WorkflowInFromContext(ctx context.Context, workflowID string) (string, error) {
	tenantID := MustGetTenantIDFromMessagingContext(ctx)
	return tsb.TenantWorkflowIn(tenantID, workflowID), nil
}

// WorkflowOutFromContext builds a tenant-scoped workflow outbound subject using context
func (tsb *TenantSubjectBuilder) WorkflowOutFromContext(ctx context.Context, workflowID string) (string, error) {
	tenantID := MustGetTenantIDFromMessagingContext(ctx)
	return tsb.TenantWorkflowOut(tenantID, workflowID), nil
}

// AgentInFromContext builds a tenant-scoped agent inbound subject using context
func (tsb *TenantSubjectBuilder) AgentInFromContext(ctx context.Context, agentID string) (string, error) {
	tenantID := MustGetTenantIDFromMessagingContext(ctx)
	return tsb.TenantAgentIn(tenantID, agentID), nil
}

// AgentOutFromContext builds a tenant-scoped agent outbound subject using context
func (tsb *TenantSubjectBuilder) AgentOutFromContext(ctx context.Context, agentID string) (string, error) {
	tenantID := MustGetTenantIDFromMessagingContext(ctx)
	return tsb.TenantAgentOut(tenantID, agentID), nil
}

// ToolsCallsFromContext builds a tenant-scoped tools calls subject using context
func (tsb *TenantSubjectBuilder) ToolsCallsFromContext(ctx context.Context) (string, error) {
	tenantID := MustGetTenantIDFromMessagingContext(ctx)
	return tsb.TenantToolsCalls(tenantID), nil
}

// ToolsAuditFromContext builds a tenant-scoped tools audit subject using context
func (tsb *TenantSubjectBuilder) ToolsAuditFromContext(ctx context.Context) (string, error) {
	tenantID := MustGetTenantIDFromMessagingContext(ctx)
	return tsb.TenantToolsAudit(tenantID), nil
}

// SystemControlFromContext builds a tenant-scoped system control subject using context
func (tsb *TenantSubjectBuilder) SystemControlFromContext(ctx context.Context) (string, error) {
	tenantID := MustGetTenantIDFromMessagingContext(ctx)
	return tsb.TenantSystemControl(tenantID), nil
}

// SystemHealthFromContext builds a tenant-scoped system health subject using context
func (tsb *TenantSubjectBuilder) SystemHealthFromContext(ctx context.Context) (string, error) {
	tenantID := MustGetTenantIDFromMessagingContext(ctx)
	return tsb.TenantSystemHealth(tenantID), nil
}

// Subject validation and parsing utilities

// ValidateTenantSubject validates that a subject follows tenant scoping pattern
func (tsb *TenantSubjectBuilder) ValidateTenantSubject(subject string) error {
	parts := strings.Split(subject, ".")
	if len(parts) < 3 {
		return fmt.Errorf("invalid tenant subject format: %s (expected at least 3 parts)", subject)
	}

	tenantID := parts[0]
	if tenantID == "" {
		return fmt.Errorf("empty tenant ID in subject: %s", subject)
	}

	// Validate tenant ID format (basic UUID validation)
	if len(tenantID) != 36 || strings.Count(tenantID, "-") != 4 {
		return fmt.Errorf("invalid tenant ID format in subject: %s", subject)
	}

	return nil
}

// ExtractTenantFromSubject extracts tenant ID from a tenant-scoped subject
func (tsb *TenantSubjectBuilder) ExtractTenantFromSubject(subject string) (string, error) {
	if err := tsb.ValidateTenantSubject(subject); err != nil {
		return "", err
	}

	parts := strings.Split(subject, ".")
	return parts[0], nil
}

// IsTenantSubject checks if a subject follows tenant scoping pattern
func (tsb *TenantSubjectBuilder) IsTenantSubject(subject string) bool {
	return tsb.ValidateTenantSubject(subject) == nil
}

// BuildTenantWildcardSubject builds a wildcard subject for tenant-scoped subscriptions
// Pattern: {tenant_id}.{category}.*
func (tsb *TenantSubjectBuilder) BuildTenantWildcardSubject(tenantID, category string) string {
	return fmt.Sprintf("%s.%s.*", tenantID, category)
}

// BuildTenantWorkflowWildcard builds a wildcard subject for all tenant workflow messages
// Pattern: {tenant_id}.workflows.*
func (tsb *TenantSubjectBuilder) BuildTenantWorkflowWildcard(tenantID string) string {
	return tsb.BuildTenantWildcardSubject(tenantID, "workflows")
}

// BuildTenantAgentWildcard builds a wildcard subject for all tenant agent messages
// Pattern: {tenant_id}.agents.*
func (tsb *TenantSubjectBuilder) BuildTenantAgentWildcard(tenantID string) string {
	return tsb.BuildTenantWildcardSubject(tenantID, "agents")
}

// BuildTenantToolsWildcard builds a wildcard subject for all tenant tools messages
// Pattern: {tenant_id}.tools.*
func (tsb *TenantSubjectBuilder) BuildTenantToolsWildcard(tenantID string) string {
	return tsb.BuildTenantWildcardSubject(tenantID, "tools")
}

// BuildTenantSystemWildcard builds a wildcard subject for all tenant system messages
// Pattern: {tenant_id}.system.*
func (tsb *TenantSubjectBuilder) BuildTenantSystemWildcard(tenantID string) string {
	return tsb.BuildTenantWildcardSubject(tenantID, "system")
}

// Cross-tenant access prevention utilities

// ValidateSubjectTenantAccess validates that a subject matches the current tenant context
func (tsb *TenantSubjectBuilder) ValidateSubjectTenantAccess(ctx context.Context, subject string) error {
	// Extract tenant from context
	contextTenantID := MustGetTenantIDFromMessagingContext(ctx)

	// Extract tenant from subject
	subjectTenantID, err := tsb.ExtractTenantFromSubject(subject)
	if err != nil {
		return fmt.Errorf("failed to extract tenant from subject: %w", err)
	}

	// Validate tenant match
	if contextTenantID != subjectTenantID {
		return fmt.Errorf("cross-tenant access denied: context tenant %s != subject tenant %s",
			contextTenantID, subjectTenantID)
	}

	return nil
}

// FilterSubjectsByTenant filters a list of subjects to only include those for the current tenant
func (tsb *TenantSubjectBuilder) FilterSubjectsByTenant(ctx context.Context, subjects []string) []string {
	tenantID := MustGetTenantIDFromMessagingContext(ctx)
	var filtered []string

	for _, subject := range subjects {
		if subjectTenantID, err := tsb.ExtractTenantFromSubject(subject); err == nil {
			if subjectTenantID == tenantID {
				filtered = append(filtered, subject)
			}
		}
	}

	return filtered
}

// Migration utilities for backward compatibility

// MigrateToTenantSubject converts a legacy subject to tenant-scoped format
func (tsb *TenantSubjectBuilder) MigrateToTenantSubject(tenantID, legacySubject string) string {
	// If already tenant-scoped, return as-is
	if tsb.IsTenantSubject(legacySubject) {
		return legacySubject
	}

	// Add tenant prefix to legacy subject
	return fmt.Sprintf("%s.%s", tenantID, legacySubject)
}

// StripTenantFromSubject removes tenant prefix from subject (for backward compatibility)
func (tsb *TenantSubjectBuilder) StripTenantFromSubject(subject string) (string, error) {
	parts := strings.Split(subject, ".")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid subject format: %s", subject)
	}

	if !tsb.IsTenantSubject(subject) {
		return subject, nil // Already non-tenant subject
	}

	// Remove tenant prefix
	return strings.Join(parts[1:], "."), nil
}

// MustGetTenantIDFromMessagingContext extracts tenant ID from context or panics
// This function looks for tenant ID in multiple context keys for compatibility
func MustGetTenantIDFromMessagingContext(ctx context.Context) string {
	// Try tenant_context first (from TenantContext)
	if tenantCtx, ok := ctx.Value("tenant_context").(*TenantContext); ok {
		return tenantCtx.TenantID
	}

	// Try tenant_id key (from auth middleware)
	if tenantID, ok := ctx.Value("tenant_id").(string); ok && tenantID != "" {
		return tenantID
	}

	// Try auth_claims (from JWT)
	if claims, ok := ctx.Value("auth_claims").(*AgentFlowClaims); ok {
		return claims.TenantID
	}

	panic("tenant ID not found in context")
}

// TenantContext represents tenant-specific context information (duplicate to avoid import cycle)
type TenantContext struct {
	TenantID       string                 `json:"tenant_id"`
	TenantName     string                 `json:"tenant_name"`
	Permissions    []string               `json:"permissions"`
	ResourceLimits map[string]interface{} `json:"resource_limits"`
}

// AgentFlowClaims represents JWT claims (duplicate to avoid import cycle)
type AgentFlowClaims struct {
	TenantID    string   `json:"tenant_id"`
	UserID      string   `json:"user_id"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
}
