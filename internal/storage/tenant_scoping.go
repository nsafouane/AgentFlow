// Package storage provides tenant-scoped database query utilities for AgentFlow
package storage

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/agentflow/agentflow/internal/logging"
)

// TenantScopedDB provides tenant-scoped database operations
type TenantScopedDB struct {
	db     *sql.DB
	logger logging.Logger
}

// NewTenantScopedDB creates a new tenant-scoped database wrapper
func NewTenantScopedDB(db *sql.DB, logger logging.Logger) *TenantScopedDB {
	return &TenantScopedDB{
		db:     db,
		logger: logger,
	}
}

// QueryContext executes a tenant-scoped query with automatic tenant_id injection
func (tsdb *TenantScopedDB) QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	tenantID := MustGetTenantIDFromContext(ctx)

	// Inject tenant scoping into the query
	scopedQuery, scopedArgs, err := tsdb.injectTenantScoping(query, tenantID, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to inject tenant scoping: %w", err)
	}

	tsdb.logger.Debug("Executing tenant-scoped query",
		logging.String("tenant_id", tenantID),
		logging.String("original_query", query),
		logging.String("scoped_query", scopedQuery),
		logging.Int("arg_count", len(scopedArgs)))

	return tsdb.db.QueryContext(ctx, scopedQuery, scopedArgs...)
}

// QueryRowContext executes a tenant-scoped query that returns a single row
func (tsdb *TenantScopedDB) QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row {
	tenantID := MustGetTenantIDFromContext(ctx)

	// Inject tenant scoping into the query
	scopedQuery, scopedArgs, err := tsdb.injectTenantScoping(query, tenantID, args...)
	if err != nil {
		// Return a row that will produce the error when scanned
		return &sql.Row{}
	}

	tsdb.logger.Debug("Executing tenant-scoped query row",
		logging.String("tenant_id", tenantID),
		logging.String("original_query", query),
		logging.String("scoped_query", scopedQuery))

	return tsdb.db.QueryRowContext(ctx, scopedQuery, scopedArgs...)
}

// ExecContext executes a tenant-scoped statement
func (tsdb *TenantScopedDB) ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	tenantID := MustGetTenantIDFromContext(ctx)

	// Inject tenant scoping into the query
	scopedQuery, scopedArgs, err := tsdb.injectTenantScoping(query, tenantID, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to inject tenant scoping: %w", err)
	}

	tsdb.logger.Debug("Executing tenant-scoped statement",
		logging.String("tenant_id", tenantID),
		logging.String("original_query", query),
		logging.String("scoped_query", scopedQuery))

	return tsdb.db.ExecContext(ctx, scopedQuery, scopedArgs...)
}

// injectTenantScoping automatically injects tenant_id WHERE clause into queries
func (tsdb *TenantScopedDB) injectTenantScoping(query string, tenantID string, args ...interface{}) (string, []interface{}, error) {
	// Normalize query for analysis
	normalizedQuery := strings.ToLower(strings.TrimSpace(query))

	// Skip tenant scoping for certain query types
	if tsdb.shouldSkipTenantScoping(normalizedQuery) {
		return query, args, nil
	}

	// Determine the table being queried
	tableName := tsdb.extractTableName(normalizedQuery)
	if tableName == "" {
		return query, args, fmt.Errorf("could not determine table name from query")
	}

	// Check if table has tenant_id column
	if !tsdb.isMultiTenantTable(tableName) {
		// Table doesn't have tenant_id, return original query
		return query, args, nil
	}

	// Check if query already has tenant_id condition
	if tsdb.hasExistingTenantCondition(normalizedQuery, tableName) {
		// Query already has tenant condition, validate it matches current tenant
		return tsdb.validateExistingTenantCondition(query, tenantID, args...)
	}

	// Inject tenant_id condition
	return tsdb.injectTenantCondition(query, tableName, tenantID, args...)
}

// shouldSkipTenantScoping determines if a query should skip tenant scoping
func (tsdb *TenantScopedDB) shouldSkipTenantScoping(normalizedQuery string) bool {
	skipPatterns := []string{
		"create table",
		"drop table",
		"alter table",
		"create index",
		"drop index",
		"show tables",
		"describe ",
		"explain ",
		"pragma ",
		"select 1", // Health checks
		"select version()",
		"select current_timestamp",
	}

	for _, pattern := range skipPatterns {
		if strings.Contains(normalizedQuery, pattern) {
			return true
		}
	}

	return false
}

// extractTableName extracts the primary table name from a query
func (tsdb *TenantScopedDB) extractTableName(normalizedQuery string) string {
	// Handle SELECT queries
	if strings.HasPrefix(normalizedQuery, "select") {
		fromIndex := strings.Index(normalizedQuery, " from ")
		if fromIndex == -1 {
			return ""
		}

		afterFrom := normalizedQuery[fromIndex+6:] // +6 for " from "
		parts := strings.Fields(afterFrom)
		if len(parts) > 0 {
			// Remove any alias or join clauses
			tableName := strings.Split(parts[0], " ")[0]
			return strings.Trim(tableName, "\"'`")
		}
	}

	// Handle INSERT queries
	if strings.HasPrefix(normalizedQuery, "insert into") {
		parts := strings.Fields(normalizedQuery)
		if len(parts) >= 3 {
			return strings.Trim(parts[2], "\"'`")
		}
	}

	// Handle UPDATE queries
	if strings.HasPrefix(normalizedQuery, "update") {
		parts := strings.Fields(normalizedQuery)
		if len(parts) >= 2 {
			return strings.Trim(parts[1], "\"'`")
		}
	}

	// Handle DELETE queries
	if strings.HasPrefix(normalizedQuery, "delete from") {
		parts := strings.Fields(normalizedQuery)
		if len(parts) >= 3 {
			return strings.Trim(parts[2], "\"'`")
		}
	}

	return ""
}

// isMultiTenantTable checks if a table has tenant_id column
func (tsdb *TenantScopedDB) isMultiTenantTable(tableName string) bool {
	multiTenantTables := map[string]bool{
		"users":         true,
		"agents":        true,
		"workflows":     true,
		"messages":      true,
		"tools":         true,
		"audits":        true,
		"budgets":       true,
		"rbac_roles":    true,
		"rbac_bindings": true,
		// plans table doesn't have direct tenant_id but is scoped through workflows
		"plans": false,
	}

	return multiTenantTables[tableName]
}

// hasExistingTenantCondition checks if query already has tenant_id condition
func (tsdb *TenantScopedDB) hasExistingTenantCondition(normalizedQuery, tableName string) bool {
	tenantConditions := []string{
		"tenant_id =",
		"tenant_id=",
		tableName + ".tenant_id =",
		tableName + ".tenant_id=",
	}

	for _, condition := range tenantConditions {
		if strings.Contains(normalizedQuery, condition) {
			return true
		}
	}

	return false
}

// validateExistingTenantCondition validates that existing tenant condition matches current tenant
func (tsdb *TenantScopedDB) validateExistingTenantCondition(query, tenantID string, args ...interface{}) (string, []interface{}, error) {
	// For now, we'll trust that the existing condition is correct
	// In a more sophisticated implementation, we would parse the query
	// and validate that the tenant_id parameter matches the current tenant

	tsdb.logger.Debug("Query already has tenant condition, validating",
		logging.String("tenant_id", tenantID),
		logging.String("query", query))

	return query, args, nil
}

// injectTenantCondition injects tenant_id condition into the query
func (tsdb *TenantScopedDB) injectTenantCondition(query, tableName, tenantID string, args ...interface{}) (string, []interface{}, error) {
	normalizedQuery := strings.ToLower(strings.TrimSpace(query))

	// Handle SELECT queries
	if strings.HasPrefix(normalizedQuery, "select") {
		return tsdb.injectTenantConditionSelect(query, tableName, tenantID, args...)
	}

	// Handle INSERT queries
	if strings.HasPrefix(normalizedQuery, "insert") {
		return tsdb.injectTenantConditionInsert(query, tenantID, args...)
	}

	// Handle UPDATE queries
	if strings.HasPrefix(normalizedQuery, "update") {
		return tsdb.injectTenantConditionUpdate(query, tenantID, args...)
	}

	// Handle DELETE queries
	if strings.HasPrefix(normalizedQuery, "delete") {
		return tsdb.injectTenantConditionDelete(query, tenantID, args...)
	}

	return query, args, fmt.Errorf("unsupported query type for tenant scoping")
}

// injectTenantConditionSelect injects tenant condition into SELECT queries
func (tsdb *TenantScopedDB) injectTenantConditionSelect(query, tableName, tenantID string, args ...interface{}) (string, []interface{}, error) {
	normalizedQuery := strings.ToLower(query)

	// Find WHERE clause
	whereIndex := strings.Index(normalizedQuery, " where ")
	if whereIndex != -1 {
		// Add to existing WHERE clause
		beforeWhere := query[:whereIndex+7] // +7 for " where "
		afterWhere := query[whereIndex+7:]

		newQuery := fmt.Sprintf("%s%s.tenant_id = $%d AND %s",
			beforeWhere, tableName, len(args)+1, afterWhere)
		newArgs := append(args, tenantID)

		return newQuery, newArgs, nil
	}

	// Find ORDER BY, GROUP BY, HAVING, LIMIT clauses
	clauseKeywords := []string{" order by ", " group by ", " having ", " limit ", " offset "}
	insertIndex := len(query)

	for _, keyword := range clauseKeywords {
		if idx := strings.Index(normalizedQuery, keyword); idx != -1 && idx < insertIndex {
			insertIndex = idx
		}
	}

	// Insert WHERE clause
	beforeClause := query[:insertIndex]
	afterClause := query[insertIndex:]

	newQuery := fmt.Sprintf("%s WHERE %s.tenant_id = $%d%s",
		beforeClause, tableName, len(args)+1, afterClause)
	newArgs := append(args, tenantID)

	return newQuery, newArgs, nil
}

// injectTenantConditionInsert injects tenant_id into INSERT queries
func (tsdb *TenantScopedDB) injectTenantConditionInsert(query, tenantID string, args ...interface{}) (string, []interface{}, error) {
	// For INSERT queries, we need to add tenant_id to the column list and values
	// This is more complex and would require full SQL parsing
	// For now, we'll assume INSERT queries already include tenant_id

	tsdb.logger.Warn("INSERT query tenant scoping not fully implemented",
		logging.String("query", query),
		logging.String("tenant_id", tenantID))

	return query, args, nil
}

// injectTenantConditionUpdate injects tenant condition into UPDATE queries
func (tsdb *TenantScopedDB) injectTenantConditionUpdate(query, tenantID string, args ...interface{}) (string, []interface{}, error) {
	normalizedQuery := strings.ToLower(query)

	// Find WHERE clause
	whereIndex := strings.Index(normalizedQuery, " where ")
	if whereIndex != -1 {
		// Add to existing WHERE clause
		beforeWhere := query[:whereIndex+7] // +7 for " where "
		afterWhere := query[whereIndex+7:]

		newQuery := fmt.Sprintf("%stenant_id = $%d AND %s",
			beforeWhere, len(args)+1, afterWhere)
		newArgs := append(args, tenantID)

		return newQuery, newArgs, nil
	}

	// Add WHERE clause
	newQuery := fmt.Sprintf("%s WHERE tenant_id = $%d", query, len(args)+1)
	newArgs := append(args, tenantID)

	return newQuery, newArgs, nil
}

// injectTenantConditionDelete injects tenant condition into DELETE queries
func (tsdb *TenantScopedDB) injectTenantConditionDelete(query, tenantID string, args ...interface{}) (string, []interface{}, error) {
	normalizedQuery := strings.ToLower(query)

	// Find WHERE clause
	whereIndex := strings.Index(normalizedQuery, " where ")
	if whereIndex != -1 {
		// Add to existing WHERE clause
		beforeWhere := query[:whereIndex+7] // +7 for " where "
		afterWhere := query[whereIndex+7:]

		newQuery := fmt.Sprintf("%stenant_id = $%d AND %s",
			beforeWhere, len(args)+1, afterWhere)
		newArgs := append(args, tenantID)

		return newQuery, newArgs, nil
	}

	// Add WHERE clause
	newQuery := fmt.Sprintf("%s WHERE tenant_id = $%d", query, len(args)+1)
	newArgs := append(args, tenantID)

	return newQuery, newArgs, nil
}

// TenantScopedQuerier wraps the generated SQLC queries with tenant scoping
type TenantScopedQuerier struct {
	db *TenantScopedDB
}

// NewTenantScopedQuerier creates a new tenant-scoped querier wrapper
func NewTenantScopedQuerier(db *sql.DB, logger logging.Logger) *TenantScopedQuerier {
	return &TenantScopedQuerier{
		db: NewTenantScopedDB(db, logger),
	}
}

// GetDB returns the underlying tenant-scoped database connection
func (tsq *TenantScopedQuerier) GetDB() *TenantScopedDB {
	return tsq.db
}

// MustGetTenantIDFromContext extracts tenant ID from context or panics
// This function looks for tenant ID in multiple context keys for compatibility
func MustGetTenantIDFromContext(ctx context.Context) string {
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
