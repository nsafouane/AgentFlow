# Audit Hash-Chain Documentation

## Overview

AgentFlow implements a tamper-evident audit logging system using cryptographic hash chains. This system ensures the integrity of audit records and provides forensic capabilities to detect unauthorized modifications to audit trails.

## Architecture

### Hash-Chain Algorithm

The audit system uses SHA-256 hash chains where each audit record contains:
- `prev_hash`: Hash of the previous audit record in the chain
- `hash`: SHA-256 hash of the current record's canonical representation

**Hash Computation Formula:**
```
hash = SHA256(prev_hash || canonical_json(audit_record))
```

Where:
- `prev_hash` is the hash from the previous audit record (nil for genesis record)
- `canonical_json` is the deterministic JSON serialization of the audit record
- `||` represents byte concatenation

### Canonical Record Format

The canonical audit record structure used for hash computation:

```json
{
  "tenant_id": "uuid-string",
  "actor_type": "user|system|agent",
  "actor_id": "identifier",
  "action": "create|update|delete|execute",
  "resource_type": "workflow|agent|tool|user",
  "resource_id": "optional-resource-identifier",
  "details": {...},
  "ts": "2025-01-01T12:00:00Z"
}
```

**Note:** The `id` field is excluded from hash computation to avoid circular dependencies.

## Database Schema

```sql
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
    prev_hash BYTEA,           -- Hash of previous record (NULL for genesis)
    hash BYTEA NOT NULL        -- SHA-256 hash of this record
);
```

## API Usage

### Creating Audit Records

```go
import "github.com/agentflow/agentflow/internal/storage/audit"

// Create audit service
service := audit.NewService(queries)

// Create audit record
params := audit.CreateAuditParams{
    TenantID:     tenantUUID,
    ActorType:    "user",
    ActorID:      "user-123",
    Action:       "create",
    ResourceType: "workflow",
    ResourceID:   &workflowID,
    Details:      map[string]interface{}{
        "workflow_name": "customer-support",
        "version": "1.0.0",
    },
}

auditRecord, err := service.CreateAudit(ctx, params)
if err != nil {
    return fmt.Errorf("failed to create audit: %w", err)
}
```

### Verifying Chain Integrity

```go
// Verify entire audit chain for a tenant
result, err := service.VerifyChainIntegrity(ctx, tenantUUID)
if err != nil {
    return fmt.Errorf("verification failed: %w", err)
}

if !result.Valid {
    log.Errorf("Audit chain compromised at record %d: %s", 
        *result.FirstTamperedIndex, result.ErrorMessage)
    // Trigger security incident response
}
```

## CLI Verification

### Basic Verification

```bash
# Verify all tenants
af audit verify

# Verify specific tenant
af audit verify --tenant-id=550e8400-e29b-41d4-a716-446655440000

# JSON output for automation
af audit verify --json
```

### Example Output

**Human-readable:**
```
Audit Hash-Chain Verification
Status: success
Total Records: 1,247
Verified Records: 1,247
Throughput: 12,450 entries/sec
Duration: 100ms
✓ Hash-chain integrity verified successfully
```

**JSON format:**
```json
{
  "status": "success",
  "timestamp": "2025-01-01T12:00:00Z",
  "total_records": 1247,
  "verified_records": 1247,
  "throughput_per_sec": 12450,
  "duration": "100ms"
}
```

**Tamper Detection:**
```json
{
  "status": "tampered",
  "timestamp": "2025-01-01T12:00:00Z",
  "total_records": 1247,
  "verified_records": 856,
  "throughput_per_sec": 12450,
  "duration": "100ms",
  "first_tampered_index": 856,
  "error_message": "hash mismatch at record 856"
}
```

## Security Properties

### Tamper Evidence

1. **Append-Only**: Records cannot be deleted without breaking the chain
2. **Modification Detection**: Any change to a record invalidates subsequent hashes
3. **Insertion Detection**: Cannot insert records without recomputing all subsequent hashes
4. **Chronological Integrity**: Timestamp-based ordering prevents reordering attacks

### Performance Characteristics

- **Verification Throughput**: ≥10,000 entries/second on development hardware
- **Hash Computation**: ~0.1ms per record (SHA-256 + JSON serialization)
- **Storage Overhead**: 32 bytes per record for hash storage
- **Index Performance**: O(log n) lookup by tenant, timestamp, or actor

## Threat Model

### Protected Against

- **Malicious Insiders**: Database administrators cannot modify records without detection
- **Compromised Applications**: Attackers cannot cover their tracks by modifying audit logs
- **Data Corruption**: Hardware failures or software bugs are detected during verification
- **Compliance Violations**: Regulatory auditors can verify log integrity

### Not Protected Against

- **Genesis Record Tampering**: First record has no previous hash to validate against
- **Complete Chain Replacement**: Replacing entire audit table with valid but false chain
- **Time-of-Check vs Time-of-Use**: Records verified as valid may be modified after verification
- **Cryptographic Attacks**: SHA-256 collision or preimage attacks (theoretical)

## Operational Procedures

### Regular Verification

```bash
# Daily verification (recommended)
0 2 * * * /usr/local/bin/af audit verify --json > /var/log/audit-verify.log

# Alert on failures
if ! af audit verify --json | jq -e '.status == "success"'; then
    echo "ALERT: Audit chain integrity compromised" | mail -s "Security Alert" security@company.com
fi
```

### Forensics Verification Procedure

When tampering is detected, follow this systematic forensics procedure:

#### 1. Immediate Response
```bash
# Capture current state
af audit verify --json > incident-$(date +%Y%m%d-%H%M%S).json

# Isolate affected tenant
TENANT_ID=$(jq -r '.tenant_id // "all"' incident-*.json)
echo "Affected tenant: $TENANT_ID"

# Stop all write operations to audit table
# (Implementation depends on your access control system)
```

#### 2. Evidence Collection
```bash
# Get detailed information about tampered record
TAMPERED_INDEX=$(jq -r '.first_tampered_index' incident-*.json)
echo "First tampered record index: $TAMPERED_INDEX"

# Extract tampered record details
psql "$DATABASE_URL" -c "
SELECT 
    id,
    tenant_id,
    actor_type,
    actor_id,
    action,
    resource_type,
    resource_id,
    details,
    ts,
    encode(prev_hash, 'hex') as prev_hash_hex,
    encode(hash, 'hex') as hash_hex
FROM audits 
WHERE tenant_id = '$TENANT_ID'
ORDER BY ts 
OFFSET $TAMPERED_INDEX LIMIT 1;
" > tampered-record-details.txt
```

#### 3. Chain Analysis
```bash
# Analyze chain break point
psql "$DATABASE_URL" -c "
WITH audit_chain AS (
    SELECT 
        ROW_NUMBER() OVER (ORDER BY ts) - 1 as index,
        id,
        actor_type,
        actor_id,
        action,
        resource_type,
        ts,
        encode(prev_hash, 'hex') as prev_hash_hex,
        encode(hash, 'hex') as hash_hex
    FROM audits 
    WHERE tenant_id = '$TENANT_ID'
    ORDER BY ts
)
SELECT * FROM audit_chain 
WHERE index BETWEEN $((TAMPERED_INDEX - 2)) AND $((TAMPERED_INDEX + 2));
" > chain-context.txt
```

#### 4. Timeline Reconstruction
```bash
# Identify when tampering likely occurred
psql "$DATABASE_URL" -c "
SELECT 
    'Last valid backup' as event,
    MAX(ts) as timestamp
FROM audit_backups 
WHERE status = 'verified'
UNION ALL
SELECT 
    'First tampered record' as event,
    ts as timestamp
FROM audits 
WHERE tenant_id = '$TENANT_ID'
ORDER BY ts 
OFFSET $TAMPERED_INDEX LIMIT 1;
" > tampering-timeline.txt
```

#### 5. Impact Assessment
```bash
# Count affected records
TOTAL_RECORDS=$(jq -r '.total_records' incident-*.json)
VERIFIED_RECORDS=$(jq -r '.verified_records' incident-*.json)
AFFECTED_RECORDS=$((TOTAL_RECORDS - VERIFIED_RECORDS))

echo "Impact Assessment:" > impact-assessment.txt
echo "Total records: $TOTAL_RECORDS" >> impact-assessment.txt
echo "Verified records: $VERIFIED_RECORDS" >> impact-assessment.txt
echo "Potentially affected: $AFFECTED_RECORDS" >> impact-assessment.txt

# Identify affected resources
psql "$DATABASE_URL" -c "
SELECT 
    resource_type,
    COUNT(*) as affected_count,
    array_agg(DISTINCT resource_id) as affected_resources
FROM audits 
WHERE tenant_id = '$TENANT_ID'
AND ts >= (
    SELECT ts FROM audits 
    WHERE tenant_id = '$TENANT_ID'
    ORDER BY ts 
    OFFSET $TAMPERED_INDEX LIMIT 1
)
GROUP BY resource_type;
" >> impact-assessment.txt
```

### Incident Response

1. **Detection**: Automated verification detects tampering
2. **Isolation**: Immediately isolate affected systems
3. **Analysis**: Follow forensics verification procedure above
4. **Recovery**: Restore from known-good backup if necessary
5. **Investigation**: Complete forensic analysis of tampered records
6. **Remediation**: Implement additional security controls
7. **Documentation**: Create incident report with evidence

### Backup Integration

```bash
# Include audit verification in backup validation
pg_dump agentflow_prod | gzip > backup.sql.gz
af audit verify --json > backup-audit-status.json

# Verify backup integrity
gunzip -c backup.sql.gz | psql agentflow_test
af audit verify --json | jq '.status == "success"'
```

## Performance Tuning

### Database Optimization

```sql
-- Optimize verification queries
CREATE INDEX idx_audits_tenant_ts ON audits(tenant_id, ts);
CREATE INDEX idx_audits_verification ON audits(tenant_id, ts) INCLUDE (prev_hash, hash);

-- Partition large audit tables
CREATE TABLE audits_2025_01 PARTITION OF audits 
FOR VALUES FROM ('2025-01-01') TO ('2025-02-01');
```

### Application Optimization

```go
// Batch verification for large chains
const batchSize = 1000
for offset := 0; offset < totalRecords; offset += batchSize {
    batch := records[offset:min(offset+batchSize, totalRecords)]
    if !verifyBatch(batch) {
        return fmt.Errorf("verification failed at batch starting %d", offset)
    }
}
```

## Compliance Considerations

### SOC 2 Type II

- **CC6.1**: Audit logs provide evidence of system changes
- **CC6.2**: Hash-chain prevents unauthorized log modifications
- **CC6.3**: Regular verification ensures ongoing integrity

### GDPR Article 32

- **Security Measures**: Cryptographic integrity protection
- **Breach Detection**: Automated tamper detection
- **Audit Trail**: Complete record of data processing activities

### HIPAA § 164.312(b)

- **Audit Controls**: Comprehensive audit logging
- **Integrity**: Hash-chain prevents unauthorized modifications
- **Transmission Security**: Audit records protected in transit and at rest

## Troubleshooting

### Common Issues

#### Verification Timeout
**Symptoms:** `af audit verify` hangs or times out on large audit chains

**Solutions:**
```bash
# Increase timeout for large chains
export AUDIT_VERIFY_TIMEOUT=300s
af audit verify

# Process specific tenant only
af audit verify --tenant-id=550e8400-e29b-41d4-a716-446655440000

# Check database connection
psql "$DATABASE_URL" -c "SELECT COUNT(*) FROM audits;"
```

#### Memory Usage Issues
**Symptoms:** High memory consumption during verification, OOM errors

**Solutions:**
```bash
# Process in smaller batches (not yet implemented - future enhancement)
af audit verify --batch-size=1000

# Verify one tenant at a time
for tenant in $(psql "$DATABASE_URL" -t -c "SELECT id FROM tenants;"); do
    af audit verify --tenant-id="$tenant"
done
```

#### Performance Issues
**Symptoms:** Slow verification, throughput below 10k entries/sec

**Diagnosis:**
```sql
-- Check for missing indexes
EXPLAIN ANALYZE SELECT * FROM audits WHERE tenant_id = $1 ORDER BY ts;

-- Check table statistics
SELECT 
    schemaname,
    tablename,
    n_tup_ins,
    n_tup_upd,
    n_tup_del,
    last_vacuum,
    last_analyze
FROM pg_stat_user_tables 
WHERE tablename = 'audits';
```

**Solutions:**
```sql
-- Vacuum and analyze
VACUUM ANALYZE audits;

-- Rebuild indexes if needed
REINDEX TABLE audits;

-- Check for table bloat
SELECT 
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) as size,
    pg_size_pretty(pg_relation_size(schemaname||'.'||tablename)) as table_size,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename) - pg_relation_size(schemaname||'.'||tablename)) as index_size
FROM pg_tables 
WHERE tablename = 'audits';
```

#### Database Connection Errors
**Symptoms:** "Failed to connect to database" errors

**Diagnosis:**
```bash
# Test basic connectivity
psql "$DATABASE_URL" -c "SELECT 1;"

# Check if database exists
psql "$(echo $DATABASE_URL | sed 's/\/[^\/]*$/\/postgres/')" -c "SELECT datname FROM pg_database WHERE datname = 'agentflow_dev';"

# Verify credentials
echo "Database URL: $DATABASE_URL"
```

**Solutions:**
```bash
# Set correct database URL
export DATABASE_URL="postgres://agentflow:dev_password@localhost:5432/agentflow_dev?sslmode=disable"

# Start PostgreSQL service
# On Windows: net start postgresql-x64-15
# On Linux: sudo systemctl start postgresql
# On macOS: brew services start postgresql

# Create database if missing
createdb -h localhost -U agentflow agentflow_dev
```

#### Hash Mismatch Errors
**Symptoms:** Verification fails with "hash mismatch at record X"

**Diagnosis:**
```bash
# Get details of problematic record
TENANT_ID="your-tenant-id"
RECORD_INDEX=X  # Replace with actual index

psql "$DATABASE_URL" -c "
SELECT 
    id,
    actor_type,
    actor_id,
    action,
    resource_type,
    details,
    ts,
    encode(prev_hash, 'hex') as prev_hash_hex,
    encode(hash, 'hex') as hash_hex
FROM audits 
WHERE tenant_id = '$TENANT_ID'
ORDER BY ts 
OFFSET $RECORD_INDEX LIMIT 1;
"
```

**Possible Causes:**
1. **Data Tampering**: Unauthorized modification of audit records
2. **Corruption**: Hardware failure or software bug corrupted data
3. **Migration Issues**: Problems during database migration or upgrade
4. **Clock Skew**: System time changes affecting timestamp-based ordering

**Investigation Steps:**
```bash
# Check for recent database changes
psql "$DATABASE_URL" -c "
SELECT 
    schemaname,
    tablename,
    n_tup_upd,
    n_tup_del,
    last_vacuum,
    last_analyze
FROM pg_stat_user_tables 
WHERE tablename = 'audits';
"

# Review database logs for suspicious activity
# Location varies by system: /var/log/postgresql/, /opt/homebrew/var/log/

# Check system integrity
# Run filesystem check, memory test, etc.
```

#### Exit Code Issues
**Symptoms:** Unexpected exit codes from `af audit verify`

**Expected Exit Codes:**
- `0`: Verification successful, no tampering detected
- `1`: Tampering detected OR database/system error
- `>1`: Unexpected error (should not occur)

**Debugging:**
```bash
# Capture exit code
af audit verify --tenant-id="$TENANT_ID"
echo "Exit code: $?"

# Get detailed error information
af audit verify --tenant-id="$TENANT_ID" --json | jq '.error_message'

# Check system logs
journalctl -u agentflow --since "1 hour ago"  # Linux
# or check Windows Event Viewer
```

### Debug Mode

```bash
# Enable debug logging (future enhancement)
export AUDIT_DEBUG=true
af audit verify --tenant-id=uuid

# Verify specific record (future enhancement)
af audit verify-record --id=record-uuid

# Manual hash computation for debugging
psql "$DATABASE_URL" -c "
SELECT 
    id,
    tenant_id,
    actor_type,
    actor_id,
    action,
    resource_type,
    resource_id,
    details,
    ts
FROM audits 
WHERE id = 'record-uuid';
"
```

### Performance Benchmarking

```bash
# Measure verification performance
time af audit verify --tenant-id="$TENANT_ID" --json

# Expected performance targets:
# - Throughput: ≥10,000 entries/second
# - Latency: <100ms for chains up to 1,000 records
# - Memory: <100MB for chains up to 100,000 records

# If performance is below targets:
# 1. Check database indexes
# 2. Verify hardware resources (CPU, memory, disk I/O)
# 3. Consider database tuning (shared_buffers, work_mem)
# 4. Review network latency between application and database
```

### Recovery Procedures

#### Restore from Backup
```bash
# If tampering is confirmed, restore from last known good backup
pg_dump agentflow_prod > backup-before-restore.sql
pg_restore --clean --if-exists backup-verified-YYYYMMDD.sql

# Verify restored chain
af audit verify --json | jq '.status == "success"'
```

#### Partial Chain Recovery
```bash
# If only recent records are affected, truncate to last valid record
LAST_VALID_INDEX=X  # Determined from forensics analysis

psql "$DATABASE_URL" -c "
DELETE FROM audits 
WHERE tenant_id = '$TENANT_ID'
AND ts > (
    SELECT ts FROM audits 
    WHERE tenant_id = '$TENANT_ID'
    ORDER BY ts 
    OFFSET $LAST_VALID_INDEX LIMIT 1
);
"

# Verify truncated chain
af audit verify --tenant-id="$TENANT_ID"
```

## Future Enhancements

### Planned Features

1. **Merkle Tree Integration**: Batch verification with logarithmic complexity
2. **Cross-Tenant Verification**: Global integrity across all tenants
3. **Blockchain Anchoring**: Periodic hash anchoring to public blockchain
4. **Hardware Security Modules**: HSM-based hash computation for enhanced security
5. **Real-time Verification**: Streaming verification for high-throughput systems

### Research Areas

1. **Post-Quantum Cryptography**: Migration to quantum-resistant hash functions
2. **Zero-Knowledge Proofs**: Verify integrity without revealing audit content
3. **Distributed Verification**: Multi-party verification protocols
4. **Homomorphic Hashing**: Compute on encrypted audit records

## References

- [NIST SP 800-57](https://csrc.nist.gov/publications/detail/sp/800-57-part-1/rev-5/final): Cryptographic Key Management
- [RFC 6234](https://tools.ietf.org/html/rfc6234): SHA-256 Specification
- [OWASP Logging Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Logging_Cheat_Sheet.html)
- [SOC 2 Trust Services Criteria](https://www.aicpa.org/content/dam/aicpa/interestareas/frc/assuranceadvisoryservices/downloadabledocuments/trust-services-criteria.pdf)