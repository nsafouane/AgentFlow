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

### Incident Response

1. **Detection**: Automated verification detects tampering
2. **Isolation**: Immediately isolate affected systems
3. **Analysis**: Identify first tampered record and timeline
4. **Recovery**: Restore from known-good backup if necessary
5. **Investigation**: Forensic analysis of tampered records

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

**Verification Timeout:**
```bash
# Increase timeout for large chains
export AUDIT_VERIFY_TIMEOUT=300s
af audit verify
```

**Memory Usage:**
```bash
# Process in smaller batches
af audit verify --batch-size=1000
```

**Performance Issues:**
```sql
-- Check for missing indexes
EXPLAIN ANALYZE SELECT * FROM audits WHERE tenant_id = $1 ORDER BY ts;

-- Vacuum and analyze
VACUUM ANALYZE audits;
```

### Debug Mode

```bash
# Enable debug logging
export AUDIT_DEBUG=true
af audit verify --tenant-id=uuid

# Verify specific record
af audit verify-record --id=record-uuid
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