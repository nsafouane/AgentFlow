-- +goose Up
-- Core schema migration for AgentFlow relational storage
-- Implements all core tables with multi-tenant isolation, proper constraints, and performance indexes

-- Create extension for UUID generation if not exists
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- ============================================================================
-- CORE TENANT TABLES
-- ============================================================================

-- Tenants table - root of multi-tenant hierarchy
CREATE TABLE tenants (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name VARCHAR(255) NOT NULL UNIQUE,
    tier VARCHAR(50) NOT NULL DEFAULT 'free',
    settings JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- Users table - tenant-scoped user management
CREATE TABLE users (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'viewer',
    hashed_secret VARCHAR(255),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(tenant_id, email)
);

-- ============================================================================
-- AGENT & WORKFLOW TABLES
-- ============================================================================

-- Agents table - tenant-scoped agent definitions
CREATE TABLE agents (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(100) NOT NULL,
    role VARCHAR(100),
    config_json JSONB NOT NULL DEFAULT '{}',
    policies_json JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(tenant_id, name)
);

-- Workflows table - tenant-scoped workflow definitions
CREATE TABLE workflows (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    version VARCHAR(50) NOT NULL DEFAULT '1.0.0',
    config_yaml TEXT NOT NULL,
    planner_type VARCHAR(50) NOT NULL,
    template_version_constraint VARCHAR(100),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(tenant_id, name, version)
);

-- Plans table - workflow execution plans
CREATE TABLE plans (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    workflow_id UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    state JSONB NOT NULL DEFAULT '{}',
    steps JSONB NOT NULL DEFAULT '[]',
    assignments JSONB NOT NULL DEFAULT '{}',
    cost JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- ============================================================================
-- MESSAGING & COMMUNICATION TABLES
-- ============================================================================

-- Messages table - integrated with Q1.2 messaging backbone
CREATE TABLE messages (
    id UUID PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    trace_id VARCHAR(32),
    span_id VARCHAR(16),
    from_agent VARCHAR(255) NOT NULL,
    to_agent VARCHAR(255) NOT NULL,
    type VARCHAR(20) NOT NULL CHECK (type IN ('request', 'response', 'event', 'control')),
    payload JSONB NOT NULL DEFAULT '{}',
    metadata JSONB NOT NULL DEFAULT '{}',
    cost JSONB NOT NULL DEFAULT '{}',
    ts TIMESTAMP WITH TIME ZONE NOT NULL,
    envelope_hash VARCHAR(64) NOT NULL
);

-- ============================================================================
-- TOOLS & EXECUTION TABLES
-- ============================================================================

-- Tools table - tenant-scoped tool definitions
CREATE TABLE tools (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    schema JSONB NOT NULL,
    permissions JSONB NOT NULL DEFAULT '{}',
    cost_model JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(tenant_id, name)
);

-- ============================================================================
-- AUDIT & SECURITY TABLES
-- ============================================================================

-- Audits table - tamper-evident audit logging with hash-chain
CREATE TABLE audits (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    actor_type VARCHAR(50) NOT NULL,
    actor_id VARCHAR(255) NOT NULL,
    action VARCHAR(100) NOT NULL,
    resource_type VARCHAR(100) NOT NULL,
    resource_id VARCHAR(255),
    details JSONB NOT NULL DEFAULT '{}',
    ts TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    prev_hash BYTEA,
    hash BYTEA NOT NULL
);

-- ============================================================================
-- BUDGET & COST MANAGEMENT TABLES
-- ============================================================================

-- Budgets table - tenant-scoped budget management
CREATE TABLE budgets (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL, -- 'workflow', 'user', 'global'
    resource_id UUID, -- workflow_id, user_id, or NULL for global
    limits JSONB NOT NULL DEFAULT '{}', -- token limits, dollar limits, etc.
    current_usage JSONB NOT NULL DEFAULT '{}',
    period VARCHAR(50) NOT NULL DEFAULT 'monthly',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(tenant_id, name)
);

-- ============================================================================
-- RBAC TABLES
-- ============================================================================

-- RBAC Roles table - tenant-scoped role definitions
CREATE TABLE rbac_roles (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    name VARCHAR(100) NOT NULL,
    permissions JSONB NOT NULL DEFAULT '[]',
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(tenant_id, name)
);

-- RBAC Bindings table - user-role assignments
CREATE TABLE rbac_bindings (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role_id UUID NOT NULL REFERENCES rbac_roles(id) ON DELETE CASCADE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    UNIQUE(tenant_id, user_id, role_id)
);

-- ============================================================================
-- PERFORMANCE INDEXES
-- ============================================================================

-- Tenant isolation indexes (critical for multi-tenant performance)
CREATE INDEX idx_users_tenant_id ON users(tenant_id);
CREATE INDEX idx_agents_tenant_id ON agents(tenant_id);
CREATE INDEX idx_workflows_tenant_id ON workflows(tenant_id);
CREATE INDEX idx_plans_workflow_id ON plans(workflow_id);
CREATE INDEX idx_messages_tenant_id ON messages(tenant_id);
CREATE INDEX idx_tools_tenant_id ON tools(tenant_id);
CREATE INDEX idx_audits_tenant_id ON audits(tenant_id);
CREATE INDEX idx_budgets_tenant_id ON budgets(tenant_id);
CREATE INDEX idx_rbac_roles_tenant_id ON rbac_roles(tenant_id);
CREATE INDEX idx_rbac_bindings_tenant_id ON rbac_bindings(tenant_id);

-- Query performance indexes
CREATE INDEX idx_messages_trace_id ON messages(trace_id);
CREATE INDEX idx_messages_ts ON messages(ts);
CREATE INDEX idx_audits_ts ON audits(ts);
CREATE INDEX idx_audits_actor ON audits(tenant_id, actor_type, actor_id);
CREATE INDEX idx_workflows_planner_type ON workflows(tenant_id, planner_type);
CREATE INDEX idx_budgets_type_resource ON budgets(tenant_id, type, resource_id);
CREATE INDEX idx_rbac_bindings_user ON rbac_bindings(tenant_id, user_id);

-- JSONB GIN indexes for flexible queries
CREATE INDEX idx_plans_state_gin ON plans USING GIN(state);
CREATE INDEX idx_agents_config_gin ON agents USING GIN(config_json);
CREATE INDEX idx_agents_policies_gin ON agents USING GIN(policies_json);
CREATE INDEX idx_messages_payload_gin ON messages USING GIN(payload);
CREATE INDEX idx_messages_metadata_gin ON messages USING GIN(metadata);
CREATE INDEX idx_tools_schema_gin ON tools USING GIN(schema);
CREATE INDEX idx_tools_permissions_gin ON tools USING GIN(permissions);
CREATE INDEX idx_audits_details_gin ON audits USING GIN(details);
CREATE INDEX idx_budgets_limits_gin ON budgets USING GIN(limits);
CREATE INDEX idx_budgets_usage_gin ON budgets USING GIN(current_usage);
CREATE INDEX idx_rbac_roles_permissions_gin ON rbac_roles USING GIN(permissions);

-- Remove the baseline table from initial migration
DROP TABLE IF EXISTS migration_baseline;

-- +goose Down
-- Rollback core schema migration
-- Drop tables in reverse dependency order

DROP TABLE IF EXISTS rbac_bindings;
DROP TABLE IF EXISTS rbac_roles;
DROP TABLE IF EXISTS budgets;
DROP TABLE IF EXISTS audits;
DROP TABLE IF EXISTS tools;
DROP TABLE IF EXISTS messages;
DROP TABLE IF EXISTS plans;
DROP TABLE IF EXISTS workflows;
DROP TABLE IF EXISTS agents;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS tenants;

-- Recreate baseline table for rollback compatibility
CREATE TABLE IF NOT EXISTS migration_baseline (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    description TEXT NOT NULL DEFAULT 'Initial migration baseline'
);

INSERT INTO migration_baseline (description) VALUES ('AgentFlow initial schema baseline');