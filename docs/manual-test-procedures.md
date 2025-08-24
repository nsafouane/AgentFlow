# Manual Test Procedures - Database Operations

This document provides step-by-step manual testing procedures for validating database migrations, schema integrity, and cross-platform compatibility.

## Prerequisites

- PostgreSQL 15+ running locally or via Docker
- Go 1.22+ installed
- goose migration tool installed (`go install github.com/pressly/goose/v3/cmd/goose@latest`)
- sqlc installed (`go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest`)

## Test Environment Setup

### 1. Start PostgreSQL Database

Using Docker:
```bash
docker run --name agentflow-test-db \
  -e POSTGRES_DB=agentflow_test \
  -e POSTGRES_USER=agentflow \
  -e POSTGRES_PASSWORD=test_password \
  -p 5432:5432 \
  -d postgres:15-alpine
```

### 2. Set Connection String

```bash
# Windows (PowerShell)
$env:DATABASE_URL = "postgres://agentflow:test_password@localhost:5432/agentflow_test?sslmode=disable"

# Linux/macOS
export DATABASE_URL="postgres://agentflow:test_password@localhost:5432/agentflow_test?sslmode=disable"
```

## Core Schema Migration Test Procedure

### Test 1: Fresh Migration Application

**Objective**: Verify that migrations apply cleanly on a fresh database.

**Steps**:

1. **Clean Database State**
   ```bash
   # Connect to database and drop all tables if they exist
   psql $DATABASE_URL -c "DROP SCHEMA public CASCADE; CREATE SCHEMA public;"
   ```

2. **Apply All Migrations**
   ```bash
   cd migrations
   goose postgres $DATABASE_URL up
   ```

3. **Verify Migration Status**
   ```bash
   goose postgres $DATABASE_URL status
   ```
   
   **Expected Result**: All migrations should show as "Applied"

4. **Verify Table Creation**
   ```bash
   psql $DATABASE_URL -c "\dt"
   ```
   
   **Expected Result**: Should list all core tables:
   - tenants
   - users
   - agents
   - workflows
   - plans
   - messages
   - tools
   - audits
   - budgets
   - rbac_roles
   - rbac_bindings

5. **Verify Indexes**
   ```bash
   psql $DATABASE_URL -c "\di"
   ```
   
   **Expected Result**: Should show all tenant isolation and performance indexes

### Test 2: Seed Test Data

**Objective**: Verify that the schema supports realistic data insertion.

**Steps**:

1. **Insert Test Tenant**
   ```sql
   INSERT INTO tenants (name, tier, settings) 
   VALUES ('test-tenant', 'premium', '{"feature_flags": {"advanced_analytics": true}}');
   ```

2. **Insert Test User**
   ```sql
   INSERT INTO users (tenant_id, email, role) 
   VALUES (
     (SELECT id FROM tenants WHERE name = 'test-tenant'),
     'admin@test-tenant.com',
     'admin'
   );
   ```

3. **Insert Test Agent**
   ```sql
   INSERT INTO agents (tenant_id, name, type, config_json) 
   VALUES (
     (SELECT id FROM tenants WHERE name = 'test-tenant'),
     'customer-support-agent',
     'llm',
     '{"model": "gpt-4", "temperature": 0.7}'
   );
   ```

4. **Insert Test Workflow**
   ```sql
   INSERT INTO workflows (tenant_id, name, planner_type, config_yaml) 
   VALUES (
     (SELECT id FROM tenants WHERE name = 'test-tenant'),
     'customer-support-flow',
     'fsm',
     'version: 1.0\nsteps:\n  - name: classify\n    agent: customer-support-agent'
   );
   ```

5. **Insert Test Message**
   ```sql
   INSERT INTO messages (id, tenant_id, from_agent, to_agent, type, payload, ts, envelope_hash) 
   VALUES (
     gen_random_uuid(),
     (SELECT id FROM tenants WHERE name = 'test-tenant'),
     'user',
     'customer-support-agent',
     'request',
     '{"text": "I need help with my account"}',
     NOW(),
     'sha256:abcd1234567890abcdef1234567890abcdef1234567890abcdef1234567890ab'
   );
   ```

6. **Insert Test Audit Record**
   ```sql
   INSERT INTO audits (tenant_id, actor_type, actor_id, action, resource_type, resource_id, details, hash) 
   VALUES (
     (SELECT id FROM tenants WHERE name = 'test-tenant'),
     'user',
     'admin@test-tenant.com',
     'create',
     'workflow',
     (SELECT id FROM workflows WHERE name = 'customer-support-flow')::text,
     '{"workflow_name": "customer-support-flow"}',
     '\x1234567890abcdef1234567890abcdef1234567890abcdef1234567890abcdef12'
   );
   ```

7. **Verify Data Integrity**
   ```sql
   -- Check tenant isolation
   SELECT COUNT(*) FROM users WHERE tenant_id = (SELECT id FROM tenants WHERE name = 'test-tenant');
   SELECT COUNT(*) FROM agents WHERE tenant_id = (SELECT id FROM tenants WHERE name = 'test-tenant');
   
   -- Check foreign key constraints
   SELECT u.email, t.name FROM users u JOIN tenants t ON u.tenant_id = t.id;
   SELECT a.name, t.name FROM agents a JOIN tenants t ON a.tenant_id = t.id;
   ```

### Test 3: Migration Rollback

**Objective**: Verify that down migrations work correctly.

**Steps**:

1. **Check Current Migration Status**
   ```bash
   goose postgres $DATABASE_URL status
   ```

2. **Rollback One Migration**
   ```bash
   goose postgres $DATABASE_URL down
   ```

3. **Verify Rollback**
   ```bash
   psql $DATABASE_URL -c "\dt"
   ```
   
   **Expected Result**: Core tables should be removed, only migration_baseline should remain

4. **Verify Data Cleanup**
   ```bash
   psql $DATABASE_URL -c "SELECT COUNT(*) FROM migration_baseline;"
   ```
   
   **Expected Result**: Should return the baseline record

### Test 4: Re-Migration

**Objective**: Verify that migrations can be re-applied after rollback.

**Steps**:

1. **Re-apply Latest Migration**
   ```bash
   goose postgres $DATABASE_URL up
   ```

2. **Verify Tables Recreated**
   ```bash
   psql $DATABASE_URL -c "\dt"
   ```
   
   **Expected Result**: All core tables should be present again

3. **Verify Schema Integrity**
   ```bash
   psql $DATABASE_URL -c "\d tenants"
   psql $DATABASE_URL -c "\d users"
   psql $DATABASE_URL -c "\d agents"
   ```
   
   **Expected Result**: Table structures should match original schema

## Cross-Platform Testing

### Windows-Specific Tests

**Objective**: Verify migrations work correctly on Windows with proper path handling.

**Steps**:

1. **Test Windows Path Handling**
   ```powershell
   # Ensure migrations directory uses Windows paths correctly
   cd migrations
   Get-ChildItem *.sql | ForEach-Object { Write-Host $_.FullName }
   ```

2. **Run Migration with Windows Paths**
   ```powershell
   goose postgres $env:DATABASE_URL up
   ```

3. **Verify sqlc Generation on Windows**
   ```powershell
   cd ..
   sqlc generate
   go test ./internal/storage/queries -v
   ```

### WSL2-Specific Tests

**Objective**: Verify compatibility with WSL2 development environment.

**Steps**:

1. **Test WSL2 Path Translation**
   ```bash
   # In WSL2 terminal
   cd /mnt/c/path/to/agentflow/migrations
   goose postgres $DATABASE_URL up
   ```

2. **Verify Cross-Platform File Access**
   ```bash
   # Verify files are accessible from both Windows and WSL2
   ls -la *.sql
   ```

## Performance Validation

### Index Performance Test

**Objective**: Verify that indexes provide expected performance benefits.

**Steps**:

1. **Insert Large Dataset**
   ```sql
   -- Insert 1000 test tenants
   INSERT INTO tenants (name, tier) 
   SELECT 'tenant-' || generate_series(1, 1000), 'free';
   
   -- Insert 10000 test users (10 per tenant)
   INSERT INTO users (tenant_id, email, role)
   SELECT t.id, 'user-' || u.user_num || '@' || t.name || '.com', 'viewer'
   FROM tenants t
   CROSS JOIN generate_series(1, 10) u(user_num);
   ```

2. **Test Tenant Isolation Query Performance**
   ```sql
   EXPLAIN ANALYZE 
   SELECT * FROM users 
   WHERE tenant_id = (SELECT id FROM tenants WHERE name = 'tenant-500');
   ```
   
   **Expected Result**: Should use index scan on idx_users_tenant_id

3. **Test JSONB GIN Index Performance**
   ```sql
   -- Insert test data with JSONB
   UPDATE agents SET config_json = '{"model": "gpt-4", "temperature": 0.7, "max_tokens": 1000}'
   WHERE tenant_id = (SELECT id FROM tenants WHERE name = 'tenant-1');
   
   EXPLAIN ANALYZE
   SELECT * FROM agents 
   WHERE config_json @> '{"model": "gpt-4"}';
   ```
   
   **Expected Result**: Should use GIN index scan on idx_agents_config_gin

## Cleanup

### Test Environment Cleanup

**Steps**:

1. **Stop Test Database**
   ```bash
   docker stop agentflow-test-db
   docker rm agentflow-test-db
   ```

2. **Clean Generated Files** (if needed)
   ```bash
   # Remove generated sqlc files for clean regeneration
   rm -f internal/storage/queries/*.sql.go
   rm -f internal/storage/queries/models.go
   rm -f internal/storage/queries/querier.go
   ```

## Success Criteria

All manual tests should pass with the following criteria:

- ✅ Fresh migrations apply without errors
- ✅ All core tables and indexes are created
- ✅ Test data can be inserted and queried correctly
- ✅ Foreign key constraints enforce data integrity
- ✅ Multi-tenant isolation works correctly
- ✅ Down migrations clean up properly
- ✅ Re-migration works after rollback
- ✅ Cross-platform compatibility (Windows, Linux, WSL2)
- ✅ Performance indexes are utilized correctly
- ✅ SQLC code generation works without errors
- ✅ Unit tests pass for generated code

## Troubleshooting

### Common Issues

1. **Connection Refused**
   - Verify PostgreSQL is running
   - Check connection string format
   - Verify network connectivity

2. **Permission Denied**
   - Check database user permissions
   - Verify database exists
   - Check file permissions on migration files

3. **Migration Conflicts**
   - Check for duplicate migration timestamps
   - Verify migration file naming convention
   - Check for syntax errors in SQL

4. **SQLC Generation Errors**
   - Verify sqlc.yaml configuration
   - Check SQL query syntax
   - Ensure database schema is up to date

5. **Cross-Platform Path Issues**
   - Use forward slashes in configuration
   - Verify file paths are accessible
   - Check for Windows-specific path separators