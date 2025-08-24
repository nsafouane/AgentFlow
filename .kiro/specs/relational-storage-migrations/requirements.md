# Requirements Document - Relational Storage & Migrations

## Introduction

This specification defines the requirements for implementing AgentFlow's relational storage layer and database migration system. The storage system serves as the foundational data layer for the entire AgentFlow platform, supporting multi-tenant operations, audit trails, and secure data persistence. This implementation builds upon the messaging backbone (Q1.2) and enables the control plane API (Q1.4), orchestrator (Q1.5), and all subsequent components.

The storage system must provide production-grade reliability, security, and performance while maintaining strict data integrity through hash-chain auditing and supporting cross-platform development environments.

## Requirements

### Requirement 1

**User Story:** As a database administrator, I want a comprehensive database schema that supports all AgentFlow core entities, so that the platform can store and manage tenants, users, agents, workflows, plans, messages, tools, audits, budgets, and RBAC data.

#### Acceptance Criteria

1. WHEN core schema migrations are implemented THEN the system SHALL create tables for tenants, users, agents, workflows, plans, messages, tools, audits, budgets, rbac_roles, and rbac_bindings
2. WHEN migrations are applied THEN all tables SHALL include appropriate primary keys, foreign key constraints, and indexes for performance
3. WHEN schema is designed THEN all tables SHALL include tenant_id for multi-tenant isolation except for tenant-scoped tables
4. WHEN migrations run THEN the system SHALL support both up and down migrations with minimal safe rollback where applicable
5. IF a migration fails THEN the system SHALL provide clear error messages and maintain database consistency

### Requirement 2

**User Story:** As a security engineer, I want tamper-evident audit logging with hash-chain integrity, so that I can detect any unauthorized modifications to audit records and maintain compliance.

#### Acceptance Criteria

1. WHEN audit hash-chain columns are implemented THEN the system SHALL add prev_hash and hash columns to the audits table
2. WHEN audit records are inserted THEN the system SHALL compute hash as H(prev_hash || serialized_record)
3. WHEN hash-chain integrity is verified THEN the system SHALL detect any tampered or missing audit records
4. WHEN audit verification runs THEN the system SHALL report the first tamper index if integrity is compromised
5. IF audit records are modified outside the application THEN the hash-chain verification SHALL fail and alert administrators

### Requirement 3

**User Story:** As a platform operator, I want message envelope hash persistence, so that I can verify message integrity during replay operations and detect any message tampering.

#### Acceptance Criteria

1. WHEN envelope hash persistence is implemented THEN the messages table SHALL store the envelope_hash from Q1.2 message contract
2. WHEN messages are inserted THEN the system SHALL validate that envelope_hash is present and correctly formatted
3. WHEN message integrity is checked THEN the system SHALL recompute envelope_hash and compare with stored value
4. WHEN replay operations occur THEN the system SHALL use envelope_hash to verify message authenticity
5. IF envelope_hash is missing or invalid THEN the system SHALL reject the message insertion

### Requirement 4

**User Story:** As a developer, I want Redis and vector database services available in the development environment, so that I can develop and test features that depend on caching and vector operations.

#### Acceptance Criteria

1. WHEN Redis and vector dev bootstrap is implemented THEN docker-compose SHALL include Redis and vector database services
2. WHEN services are started THEN the system SHALL perform health checks to verify connectivity
3. WHEN `af validate` runs THEN it SHALL report the status of Redis and vector database connections
4. WHEN running on Windows THEN connectivity tests SHALL conditionally skip if services are unavailable
5. IF services are not running THEN the system SHALL provide clear guidance on how to start them

### Requirement 5

**User Story:** As a security administrator, I want a secrets provider interface, so that I can securely manage and rotate secrets without hardcoding them in configuration files.

#### Acceptance Criteria

1. WHEN secrets provider stub is implemented THEN the system SHALL provide an interface supporting environment and file-based providers
2. WHEN secrets are retrieved THEN the system SHALL mask sensitive values in logs and debug output
3. WHEN secret rotation occurs THEN the system SHALL reload secrets without requiring application restart
4. WHEN secrets are accessed THEN the system SHALL validate permissions and log access attempts
5. IF secret retrieval fails THEN the system SHALL provide fallback mechanisms and alert administrators

### Requirement 6

**User Story:** As a compliance officer, I want an audit verification CLI command, so that I can verify the integrity of audit trails and detect any tampering for compliance reporting.

#### Acceptance Criteria

1. WHEN audit verification CLI is implemented THEN `af audit verify` SHALL compute and validate the entire hash-chain
2. WHEN hash-chain is intact THEN the command SHALL report successful verification with throughput metrics
3. WHEN tampering is detected THEN the command SHALL report the first tampered entry index and exit with non-zero code
4. WHEN verification runs THEN the system SHALL achieve ≥10k entries/sec throughput on development hardware
5. IF database is unavailable THEN the command SHALL provide clear error messages and troubleshooting guidance

### Requirement 7

**User Story:** As a database administrator, I want backup and restore capabilities, so that I can protect against data loss and recover from disasters within defined RPO/RTO targets.

#### Acceptance Criteria

1. WHEN backup and restore baseline is implemented THEN the system SHALL provide pg_dump scripts for schema and selective data tables
2. WHEN backups are created THEN the system SHALL generate integrity hashes for backup artifacts
3. WHEN restore is performed THEN the system SHALL validate backup integrity before restoration
4. WHEN backup/restore is tested THEN the complete roundtrip SHALL complete in <5 minutes for baseline dataset
5. IF backup fails THEN the system SHALL provide detailed error logs and recovery procedures

### Requirement 8

**User Story:** As a developer, I want a memory store stub implementation, so that I can develop and test memory-dependent features while waiting for the full memory subsystem in Q2.6.

#### Acceptance Criteria

1. WHEN memory store stub is implemented THEN the system SHALL provide in-memory Save/Query operations
2. WHEN summarization is requested THEN the stub SHALL return a constant placeholder response
3. WHEN memory operations are tested THEN the system SHALL demonstrate deterministic save/query behavior
4. WHEN integrated with worker/planner THEN the memory store SHALL be available via dependency injection with experimental flag
5. IF memory operations fail THEN the system SHALL provide clear error messages and fallback behavior

### Requirement 9

**User Story:** As a database developer, I want type-safe SQL code generation, so that I can write database queries with compile-time safety and avoid runtime SQL errors.

#### Acceptance Criteria

1. WHEN sqlc is configured THEN the system SHALL generate type-safe Go code from SQL queries
2. WHEN queries are written THEN sqlc SHALL validate SQL syntax and generate appropriate Go structs
3. WHEN database schema changes THEN generated code SHALL reflect the updated schema
4. WHEN queries are executed THEN the system SHALL provide compile-time type safety for parameters and results
5. IF SQL queries are invalid THEN sqlc generation SHALL fail with clear error messages

### Requirement 10

**User Story:** As a platform operator, I want cross-platform migration support, so that I can run database migrations consistently across Linux, Windows, and WSL2 development environments.

#### Acceptance Criteria

1. WHEN migrations run on Windows THEN path handling SHALL work correctly with Windows-style paths
2. WHEN migrations are tested THEN the system SHALL validate successful execution on Linux, Windows, and WSL2
3. WHEN migration tools are used THEN goose commands SHALL work consistently across all platforms
4. WHEN migration scripts are executed THEN Windows-specific path separators SHALL be handled correctly
5. IF platform-specific issues occur THEN the system SHALL provide platform-specific troubleshooting guidance