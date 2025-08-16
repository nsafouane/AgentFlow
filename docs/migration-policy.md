# Database Migration Policy

## Overview

This document defines the database migration policy for AgentFlow, establishing standards for schema changes, naming conventions, and operational procedures to ensure safe and reliable database evolution.

## Migration Tooling

AgentFlow uses the following tools for database schema management:

- **goose v3.17.0**: Database migration tool for schema versioning and execution
- **sqlc v1.25.0**: Type-safe Go code generation from SQL queries
- **PostgreSQL**: Primary database engine with full ACID compliance

## Naming Conventions

### Migration File Naming

Migration files MUST follow this exact pattern:
```
YYYYMMDDHHMMSS_description.sql
```

**Components:**
- `YYYY`: 4-digit year
- `MM`: 2-digit month (01-12)
- `DD`: 2-digit day (01-31)
- `HH`: 2-digit hour (00-23)
- `MM`: 2-digit minute (00-59)
- `SS`: 2-digit second (00-59)
- `description`: Lowercase description with underscores, no spaces

**Examples:**
- ✅ `20240101120000_initial_schema.sql`
- ✅ `20240115143022_add_user_authentication.sql`
- ✅ `20240220091500_create_workflow_tables.sql`
- ❌ `001_initial.sql` (missing timestamp)
- ❌ `20240101_add user table.sql` (spaces in description)
- ❌ `20240101120000_AddUserTable.sql` (camelCase description)

### Description Guidelines

Migration descriptions should be:
- **Descriptive**: Clearly indicate what the migration does
- **Concise**: Keep under 50 characters when possible
- **Action-oriented**: Use verbs like `add`, `create`, `modify`, `drop`, `rename`
- **Consistent**: Follow established patterns within the project

**Common Patterns:**
- `create_[table_name]_table`
- `add_[column_name]_to_[table_name]`
- `modify_[table_name]_[description]`
- `drop_[table_name]_table`
- `rename_[old_name]_to_[new_name]`

## Migration Structure

### Required Directives

Every migration file MUST contain both up and down migrations:

```sql
-- +goose Up
-- Migration logic for applying changes
CREATE TABLE example (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

-- +goose Down
-- Migration logic for reverting changes
DROP TABLE IF EXISTS example;
```

### Best Practices

1. **Always include both Up and Down**: Every migration must be reversible
2. **Use IF EXISTS/IF NOT EXISTS**: Protect against partial application
3. **Order matters**: Structure migrations so Up comes before Down
4. **Comments**: Include explanatory comments for complex operations
5. **Transactions**: Goose automatically wraps migrations in transactions

## Reversibility Policy

### Reversibility Stance: **REQUIRED**

AgentFlow maintains a **strict reversibility policy** for all database migrations:

#### Requirements

1. **Every migration MUST be reversible**: All migrations must include a functional `-- +goose Down` section
2. **Data preservation**: Down migrations should preserve data when possible
3. **Safe rollbacks**: Rollbacks must not cause data corruption or application failures
4. **Testing required**: Both up and down migrations must be tested before deployment

#### Reversible Operations

| Operation | Up Migration | Down Migration | Notes |
|-----------|--------------|----------------|-------|
| Create Table | `CREATE TABLE` | `DROP TABLE IF EXISTS` | Safe to drop if no critical data |
| Add Column | `ALTER TABLE ADD COLUMN` | `ALTER TABLE DROP COLUMN IF EXISTS` | Consider data loss implications |
| Create Index | `CREATE INDEX` | `DROP INDEX IF EXISTS` | Always safe to reverse |
| Add Constraint | `ALTER TABLE ADD CONSTRAINT` | `ALTER TABLE DROP CONSTRAINT IF EXISTS` | Ensure constraint name consistency |

#### Non-Reversible Operations

Some operations are inherently destructive and require special handling:

| Operation | Approach | Example |
|-----------|----------|---------|
| Drop Column | Create backup table or use feature flags | Document data recovery procedure |
| Rename Column | Use multi-step migration with deprecation period | Old column → New column → Drop old |
| Data Transformation | Preserve original data in backup columns | Keep transformation logic reversible |
| Drop Table | Require explicit confirmation and backup | Use `-- DESTRUCTIVE:` comment prefix |

#### Destructive Migration Handling

For migrations that cannot be safely reversed:

1. **Prefix with warning**: Use `-- DESTRUCTIVE:` comment at the top
2. **Require approval**: Additional review process for destructive changes
3. **Backup strategy**: Document data backup and recovery procedures
4. **Staged rollout**: Deploy in stages with validation checkpoints

```sql
-- DESTRUCTIVE: This migration drops the deprecated user_sessions table
-- Backup created: user_sessions_backup_20240101
-- Recovery procedure: docs/runbooks/data-recovery.md

-- +goose Up
DROP TABLE IF EXISTS user_sessions;

-- +goose Down
-- Cannot automatically reverse - requires manual data restoration
-- See: docs/runbooks/data-recovery.md#user-sessions-restoration
SELECT 'MANUAL_RECOVERY_REQUIRED' as status;
```

## Development Workflow

### Creating New Migrations

1. **Generate migration file**:
   ```bash
   # Using Makefile
   make migrate-create NAME=add_user_roles
   
   # Using Taskfile
   task migrate-create NAME=add_user_roles
   
   # Direct goose command
   goose -dir migrations create add_user_roles sql
   ```

2. **Edit migration file**: Add both up and down logic
3. **Test locally**: Run up and down migrations on development database
4. **Code review**: Ensure migration follows policy guidelines
5. **Deploy**: Apply migration through CI/CD pipeline

### Testing Migrations

#### Local Testing

```bash
# Check current status
make migrate-status

# Apply migrations
make migrate-up

# Test rollback
make migrate-down

# Verify database state
make migrate-status
```

#### Automated Testing

Migration tests are automatically run in CI/CD:
- Migration file syntax validation
- Up/down migration execution
- Generated code compilation
- Cross-platform compatibility

### Migration Execution

#### Development Environment

```bash
# Set database URL
export DATABASE_URL="postgres://user:pass@localhost:5432/agentflow_dev?sslmode=disable"

# Run migrations
make migrate-up
```

#### Production Environment

1. **Backup database**: Always backup before migration
2. **Maintenance window**: Schedule during low-traffic periods
3. **Staged deployment**: Apply to staging environment first
4. **Monitoring**: Monitor application health during and after migration
5. **Rollback plan**: Have tested rollback procedure ready

## Code Generation

### sqlc Configuration

AgentFlow uses sqlc for type-safe database access:

```yaml
# sqlc.yaml
version: "2"
sql:
  - engine: "postgresql"
    queries: "internal/storage/queries"
    schema: "migrations"
    gen:
      go:
        package: "queries"
        out: "internal/storage/queries"
        sql_package: "pgx/v5"
        emit_json_tags: true
        emit_interface: true
```

### Query Organization

1. **File structure**: Organize queries by domain/table
2. **Naming**: Use descriptive query names with consistent patterns
3. **Comments**: Document complex queries and business logic
4. **Parameters**: Use named parameters for clarity

```sql
-- name: GetUserByID :one
SELECT id, email, created_at, updated_at
FROM users
WHERE id = $1 AND deleted_at IS NULL;

-- name: ListActiveUsers :many
SELECT id, email, created_at
FROM users
WHERE deleted_at IS NULL
ORDER BY created_at DESC
LIMIT $1 OFFSET $2;
```

### Code Generation Workflow

```bash
# Generate type-safe Go code
make sqlc-generate

# Verify generated code compiles
go test ./internal/storage/queries/
```

## Error Handling

### Migration Failures

1. **Automatic rollback**: Failed migrations are automatically rolled back
2. **Error logging**: All migration errors are logged with context
3. **Manual intervention**: Some failures may require manual database fixes
4. **Recovery procedures**: Document recovery steps for common failure scenarios

### Common Issues

| Issue | Cause | Resolution |
|-------|-------|------------|
| Syntax Error | Invalid SQL in migration | Fix SQL and re-run |
| Constraint Violation | Data doesn't meet new constraints | Clean data or modify constraints |
| Lock Timeout | Long-running migration blocks other operations | Optimize migration or schedule maintenance window |
| Disk Space | Large table operations exceed available space | Free space or use streaming operations |

## Monitoring and Observability

### Migration Metrics

- Migration execution time
- Success/failure rates
- Rollback frequency
- Database size changes

### Alerting

- Failed migrations trigger immediate alerts
- Long-running migrations generate warnings
- Rollback events are logged and monitored

## Security Considerations

### Access Control

1. **Principle of least privilege**: Migration accounts have minimal required permissions
2. **Audit logging**: All migration activities are logged
3. **Secure credentials**: Database credentials stored in secure credential management
4. **Network security**: Migrations run over encrypted connections

### Data Protection

1. **PII handling**: Special care for migrations affecting personal data
2. **Backup encryption**: All backups are encrypted at rest
3. **Data retention**: Follow data retention policies during schema changes
4. **Compliance**: Ensure migrations maintain regulatory compliance

## Compliance and Governance

### Change Management

1. **Approval process**: All production migrations require approval
2. **Documentation**: Changes are documented in ADRs when appropriate
3. **Risk assessment**: High-risk migrations undergo additional review
4. **Rollback testing**: Rollback procedures are tested before deployment

### Audit Requirements

1. **Change tracking**: All schema changes are tracked and auditable
2. **Approval records**: Maintain records of who approved what changes
3. **Execution logs**: Complete logs of migration execution
4. **Impact assessment**: Document business impact of schema changes

## Tools and Commands Reference

### Makefile Commands

```bash
make migrate-up          # Run all pending migrations
make migrate-down        # Rollback last migration
make migrate-status      # Show migration status
make migrate-create NAME=description  # Create new migration
make sqlc-generate       # Generate type-safe Go code
```

### Taskfile Commands

```bash
task migrate-up          # Run all pending migrations
task migrate-down        # Rollback last migration
task migrate-status      # Show migration status
task migrate-create NAME=description  # Create new migration
task sqlc-generate       # Generate type-safe Go code
```

### Direct goose Commands

```bash
# Basic operations
goose -dir migrations postgres $DATABASE_URL up
goose -dir migrations postgres $DATABASE_URL down
goose -dir migrations postgres $DATABASE_URL status
goose -dir migrations postgres $DATABASE_URL version

# Advanced operations
goose -dir migrations postgres $DATABASE_URL up-to VERSION
goose -dir migrations postgres $DATABASE_URL down-to VERSION
goose -dir migrations postgres $DATABASE_URL redo
goose -dir migrations postgres $DATABASE_URL reset
```

## Troubleshooting

### Common Problems

1. **Migration stuck**: Check for long-running transactions or locks
2. **Version mismatch**: Ensure goose version consistency across environments
3. **Path issues**: Verify migration directory paths, especially on Windows
4. **Permission errors**: Check database user permissions
5. **Connection issues**: Verify database connectivity and credentials

### Recovery Procedures

1. **Manual rollback**: Steps for manual migration rollback
2. **Data recovery**: Procedures for recovering from data loss
3. **Schema repair**: Fixing corrupted schema state
4. **Version table repair**: Repairing goose version tracking

## Version History

| Version | Date | Changes |
|---------|------|---------|
| 1.0 | 2024-01-01 | Initial migration policy |

## References

- [Goose Documentation](https://github.com/pressly/goose)
- [sqlc Documentation](https://docs.sqlc.dev/)
- [PostgreSQL Documentation](https://www.postgresql.org/docs/)
- [AgentFlow Architecture Documentation](./ARCHITECTURE.md)