# Implementation Plan - Relational Storage & Migrations

This implementation plan converts the relational storage and migrations design into a series of prompts for code-generation that will implement each step in a test-driven manner. Each task builds incrementally on previous tasks and focuses on discrete, manageable coding steps that can be executed by a coding agent.

- [x] 1. Core Schema Migrations (tenants, users, agents, workflows, plans, messages, tools, audits, budgets, rbac_roles, rbac_bindings)





  - Create SQL up migrations for all core tables with appropriate primary keys, foreign key constraints, and indexes
  - Implement minimal safe down migrations where applicable (DROP TABLE IF EXISTS for new tables)
  - Add tenant_id columns to all tables for multi-tenant isolation except tenant-scoped tables
  - Create indexes for performance: tenant isolation, query optimization, and JSONB GIN indexes
  - Configure sqlc.yaml for type-safe Go code generation with PostgreSQL engine and pgx/v5 driver
  - Write unit tests for sqlc generated query compilation and forward/back migration clone tests
  - Create manual test procedure: fresh migrate, seed test data, rollback previous, re-migrate cycle
  - Document schema ER diagram showing table relationships and multi-tenant isolation patterns
  - _Requirements: 1.1, 1.2, 1.3, 1.4, 1.5, 9.1, 9.2, 9.3, 9.4, 9.5, 10.1, 10.2, 10.3, 10.4, 10.5_

- [x] 2. Audit Hash-Chain Columns









  - Add prev_hash BYTEA and hash BYTEA NOT NULL columns to audits table
  - Implement hash computation function: SHA256(prev_hash || canonical_json(audit_record))
  - Create audit record insertion logic that maintains chronological hash chain
  - Build hash-chain verification algorithm that validates entire chain integrity
  - Write unit tests for append-only integrity and tamper detection scenarios
  - Create manual test: insert audit records, manually tamper with database record, verify CLI detects failure
  - Document hash-chain rationale, algorithm details, and verification procedures
  - _Requirements: 2.1, 2.2, 2.3, 2.4, 2.5_

- [x] 3. Envelope Hash Persistence (messages table)








  - Add envelope_hash VARCHAR(64) NOT NULL column to messages table
  - Implement insert trigger or application-layer validation to ensure envelope_hash presence
  - Create message integrity validation that recomputes hash and compares with stored value
  - Build replay operation support using envelope_hash for message authenticity verification
  - Write unit tests for missing hash rejection and integrity validation scenarios
  - Create manual test: inspect stored messages, recompute envelope_hash, verify match with Q1.2 canonical serializer
  - Document replay integrity procedures and envelope_hash integration with Q1.2 messaging
  - _Requirements: 3.1, 3.2, 3.3, 3.4, 3.5_

- [x] 4. Redis & Vector Dev Bootstrap





  - Create docker-compose.yml with PostgreSQL, Redis, and vector database (Qdrant) services
  - Implement health check endpoints for all services with appropriate timeouts and retry logic
  - Add service connectivity validation to `af validate` command with status reporting
  - Create conditional test skipping for Windows environments where services may be unavailable
  - Write unit tests for connectivity validation and health check logic
  - Create manual test: start docker-compose services, run `af validate`, verify service status reporting
  - Document local services setup guide and troubleshooting procedures
  - _Requirements: 4.1, 4.2, 4.3, 4.4, 4.5_

- [x] 5. Secrets Provider Stub











  - Create SecretsProvider interface with GetSecret, SetSecret, DeleteSecret, ListSecrets, and Rotate methods
  - Implement EnvironmentProvider and FileProvider with secure value masking in logs
  - Add hot reload capability for secret rotation without application restart
  - Build permission validation and access logging for all secret operations
  - Write unit tests for secret retrieval, masking, and rotation scenarios
  - Create manual test: rotate sample secret file, verify application reload without restart
  - Document secrets usage patterns and future provider expansion plans
  - _Requirements: 5.1, 5.2, 5.3, 5.4, 5.5_

- [x] 6. Audit Verification CLI Subcommand









  - Implement `af audit verify` command that computes and validates entire hash-chain
  - Add throughput metrics reporting and performance optimization for ≥10k entries/sec target
  - Create tamper detection logic that reports first tampered entry index
  - Build exit code handling: 0 for success, >0 for tamper detection or errors
  - Write unit tests for injected tamper fixture detection and exit code validation
  - Create manual test: run against pristine database, manually tamper with audit record, verify failure detection
  - Document forensics verification procedure and troubleshooting guide
  - _Requirements: 6.1, 6.2, 6.3, 6.4, 6.5_

- [x] 7. Backup & Restore Baseline














  - Create pg_dump scripts for schema and selective data tables with compression and parallel options
  - Implement backup artifact integrity hash generation and validation
  - Build restore smoke test automation for CI pipeline integration
  - Add backup/restore roundtrip performance validation with <5 minute target for baseline dataset
  - Write unit tests for backup artifact integrity hash validation
  - Create manual test: simulate accidental table drop, restore from backup, verify data integrity
  - Document disaster recovery baseline procedures with RPO/RTO placeholders for future enhancement
  - _Requirements: 7.1, 7.2, 7.3, 7.4, 7.5_

- [x] 8. MemoryStore Stub (In-Memory + Noop Summarizer)





  - Create minimal in-memory MemoryStore implementation with Save/Query operations using hash map storage
  - Implement placeholder Summarize method that returns constant response for Q2.6 compatibility
  - Add dependency injection wiring for worker/planner integration with experimental feature flag
  - Build deterministic save/query behavior with concurrent access safety using read-write locks
  - Write unit tests for save/query determinism, summarizer no-op assertion, and basic race detector validation
  - Create manual test: sample plan writes to memory store, verify retrieval via debug log or temporary inspection endpoint
  - Document memory store stub limitations and upgrade path to Q2.6 Memory Subsystem MVP
  - _Requirements: 8.1, 8.2, 8.3, 8.4, 8.5_

## Exit Criteria (Gate G2)

This implementation plan ensures all Gate G2 exit criteria are satisfied:

- **All migrations apply & rollback cleanly**: Task 1 implements comprehensive schema with up/down migrations and cross-platform testing
- **Hash-chain integrity passes**: Task 2 implements tamper-evident audit logging with verification
- **Redis/Vector health validated**: Task 4 implements service health checks and `af validate` integration
- **Windows migration run passes**: Task 1 includes cross-platform testing with Windows path validation

## Additional Quantitative Assertions (Gate G2 Augmentation)

- **Audit chain verify throughput ≥ 10k entries/sec**: Task 6 implements performance-optimized verification with throughput metrics
- **`af audit verify` detects injected tamper within first mismatched entry index**: Task 6 includes tamper detection with precise error reporting
- **Backup+restore roundtrip < 5 min for baseline dataset**: Task 7 implements performance-validated backup/restore procedures

Each task follows the test-driven development approach with implementation, unit tests, manual validation, and documentation components as specified in the development plan.