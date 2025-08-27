# AgentFlow Disaster Recovery Baseline

## Overview

This document outlines the disaster recovery procedures for AgentFlow's relational storage system. It provides baseline procedures for backup creation, restoration, and recovery operations with defined Recovery Point Objective (RPO) and Recovery Time Objective (RTO) targets.

## Recovery Objectives

### Current Baseline Targets

- **RPO (Recovery Point Objective)**: TBD - Maximum acceptable data loss
- **RTO (Recovery Time Objective)**: TBD - Maximum acceptable downtime
- **Backup Frequency**: TBD - How often backups are created
- **Retention Period**: TBD - How long backups are retained

*Note: These targets will be defined based on business requirements and operational needs in future iterations.*

## Backup Strategy

### Backup Types

1. **Schema-Only Backup**
   - Contains database structure without data
   - Used for schema recovery and new environment setup
   - Compressed using gzip (level 6)
   - Typical size: < 1MB

2. **Full Data Backup**
   - Complete database backup including all data
   - Uses PostgreSQL directory format with parallel jobs
   - Compressed using pg_dump built-in compression
   - Typical size: Varies based on data volume

3. **Critical Tables Backup**
   - Selective backup of essential tables (tenants, users, rbac_roles, rbac_bindings, audits)
   - Faster restore for critical operations
   - Compressed using gzip (level 6)
   - Typical size: Smaller subset of full backup

### Backup Components

Each backup consists of:
- **Manifest File**: JSON metadata describing the backup
- **Hash Files**: SHA256 integrity verification for each component
- **Backup Files**: Actual database content (schema, data, critical)

## Backup Procedures

### Creating a Backup

#### Command Line Interface
```bash
# Create backup using CLI
af backup create [backup-directory]

# Create backup using scripts directly
# Linux/macOS
bash scripts/backup-database.sh [db-url] [backup-dir] [jobs] [compression-level]

# Windows
powershell -ExecutionPolicy Bypass -File scripts/backup-database.ps1 -DatabaseUrl [db-url] -BackupDir [backup-dir]
```

#### Environment Variables
- `AF_DATABASE_URL`: Database connection string
- `DATABASE_URL`: Alternative database connection string

#### Backup Process
1. **Validation**: Verify database connectivity and tools availability
2. **Schema Backup**: Create compressed schema-only dump
3. **Data Backup**: Create parallel compressed full data backup
4. **Critical Backup**: Create selective backup of critical tables
5. **Integrity**: Generate SHA256 hashes for all files
6. **Manifest**: Create JSON manifest with backup metadata
7. **Verification**: Validate backup integrity

### Backup Verification

#### Integrity Checking
```bash
# Verify backup integrity
af backup verify <backup-id> [backup-directory] [--json]

# List available backups
af backup list [backup-directory] [--json]
```

#### Verification Process
1. **Manifest Validation**: Verify manifest file integrity
2. **File Validation**: Check SHA256 hashes for all backup files
3. **Completeness**: Ensure all expected files are present
4. **Performance**: Report verification throughput and duration

## Restoration Procedures

### Restore Types

1. **Full Restore**: Complete database restoration from backup
2. **Schema Restore**: Structure-only restoration
3. **Critical Restore**: Restore only critical tables

### Restoration Process

#### Command Line Interface
```bash
# Full database restore
af backup restore <backup-id> [backup-directory] [restore-type]

# Using scripts directly
# Linux/macOS
bash scripts/restore-database.sh <backup-id> [db-url] [backup-dir] [restore-type]

# Windows
powershell -ExecutionPolicy Bypass -File scripts/restore-database.ps1 -BackupId <backup-id> -DatabaseUrl [db-url] -BackupDir [backup-dir] -RestoreType [restore-type]
```

#### Pre-Restore Validation
1. **Backup Integrity**: Verify backup files are intact
2. **Database Connectivity**: Ensure target database is accessible
3. **Confirmation**: Interactive confirmation for destructive operations
4. **Prerequisites**: Check required tools (pg_restore, psql)

#### Restore Steps
1. **Integrity Check**: Validate backup before restoration
2. **Database Preparation**: Drop/recreate database as needed
3. **Data Restoration**: Restore from backup using appropriate method
4. **Smoke Tests**: Verify critical functionality post-restore
5. **Performance Validation**: Ensure restore completes within target time

### Post-Restore Verification

#### Smoke Tests
1. **Table Existence**: Verify all expected tables are present
2. **Record Counts**: Check data volume matches expectations
3. **Foreign Keys**: Validate referential integrity
4. **Audit Chain**: Verify audit hash-chain integrity (if applicable)

## Performance Baselines

### Backup Performance
- **Target Throughput**: Varies by dataset size and hardware
- **Compression Ratio**: ~70-80% size reduction with level 6 compression
- **Parallel Jobs**: 4 jobs default (configurable)
- **Network Impact**: Minimal for local backups

### Restore Performance
- **Target Duration**: < 5 minutes for baseline dataset (1000 records)
- **Throughput**: Varies by restore type and hardware
- **Validation Time**: < 30 seconds for integrity checks

### Performance Monitoring
```bash
# Run performance validation test
bash scripts/test-backup-restore-roundtrip.sh

# Windows
powershell -ExecutionPolicy Bypass -File scripts/test-backup-restore-roundtrip.ps1
```

## Disaster Scenarios

### Scenario 1: Accidental Table Drop
**Symptoms**: Critical table missing, application errors
**Recovery**: 
1. Identify most recent backup
2. Perform critical tables restore
3. Verify data integrity
4. Resume operations

**Estimated RTO**: < 10 minutes

### Scenario 2: Database Corruption
**Symptoms**: Database startup failures, data inconsistencies
**Recovery**:
1. Assess corruption extent
2. Perform full database restore
3. Verify all functionality
4. Update applications

**Estimated RTO**: < 30 minutes

### Scenario 3: Complete Data Loss
**Symptoms**: Database server failure, storage loss
**Recovery**:
1. Provision new database server
2. Perform full restore from latest backup
3. Comprehensive testing
4. Redirect applications

**Estimated RTO**: < 60 minutes (excluding infrastructure provisioning)

## Backup Storage

### Local Storage
- **Location**: `./backups` directory (default)
- **Permissions**: Restricted access (0755 directories, 0644 files)
- **Cleanup**: Manual cleanup required
- **Monitoring**: File system space monitoring recommended

### Future Enhancements
- **Remote Storage**: S3, Azure Blob, GCS integration
- **Encryption**: At-rest encryption for sensitive data
- **Automated Cleanup**: Retention policy enforcement
- **Monitoring**: Automated backup success/failure alerts

## Security Considerations

### Access Control
- Backup files contain sensitive data
- Restrict access to authorized personnel only
- Use secure transfer methods for remote storage

### Data Protection
- Hash verification prevents tampering
- Compressed files reduce exposure surface
- Database credentials masked in logs

### Audit Trail
- All backup/restore operations logged
- Integrity verification results recorded
- Access attempts monitored

## Operational Procedures

### Daily Operations
1. **Backup Creation**: Automated or manual backup creation
2. **Integrity Verification**: Verify recent backups
3. **Storage Monitoring**: Check available disk space
4. **Performance Review**: Monitor backup/restore times

### Weekly Operations
1. **Restore Testing**: Test restore procedures
2. **Performance Validation**: Run roundtrip tests
3. **Documentation Review**: Update procedures as needed
4. **Training**: Team familiarity with procedures

### Monthly Operations
1. **Disaster Simulation**: Practice disaster scenarios
2. **Performance Baseline**: Update performance targets
3. **Procedure Updates**: Incorporate lessons learned
4. **Compliance Review**: Ensure regulatory compliance

## Troubleshooting

### Common Issues

#### Backup Failures
- **Disk Space**: Insufficient storage for backup files
- **Permissions**: Database or file system access issues
- **Network**: Connection timeouts or interruptions
- **Tools**: Missing or incompatible pg_dump version

#### Restore Failures
- **Integrity**: Backup file corruption or tampering
- **Compatibility**: Version mismatches between backup and target
- **Resources**: Insufficient memory or disk space
- **Conflicts**: Existing data or schema conflicts

### Diagnostic Commands
```bash
# Check database connectivity
psql $DATABASE_URL -c "SELECT 1;"

# Verify backup integrity
af backup verify <backup-id>

# Check disk space
df -h ./backups

# Test backup/restore cycle
bash scripts/test-backup-restore-roundtrip.sh
```

## Compliance and Auditing

### Regulatory Requirements
- **Data Retention**: Comply with industry-specific retention requirements
- **Audit Trails**: Maintain records of all backup/restore operations
- **Access Logs**: Track who performed backup/restore operations
- **Integrity Verification**: Demonstrate data integrity over time

### Documentation Requirements
- **Procedure Documentation**: Keep procedures current and accessible
- **Test Results**: Document regular testing and validation
- **Incident Reports**: Record any backup/restore incidents
- **Performance Metrics**: Track and report on RTO/RPO compliance

## Future Enhancements

### Planned Improvements
1. **Automated Scheduling**: Cron-based backup automation
2. **Remote Storage**: Cloud storage integration
3. **Encryption**: End-to-end encryption for backups
4. **Monitoring**: Automated alerting and monitoring
5. **Compression**: Advanced compression algorithms
6. **Incremental Backups**: Reduce backup time and storage
7. **Point-in-Time Recovery**: Granular recovery capabilities
8. **Cross-Region Replication**: Geographic disaster recovery

### Performance Targets (Future)
- **RPO**: < 15 minutes (with automated backups)
- **RTO**: < 5 minutes (with optimized procedures)
- **Backup Frequency**: Every 4 hours during business hours
- **Retention**: 30 days local, 1 year remote

## Contact Information

### Emergency Contacts
- **Database Administrator**: TBD
- **System Administrator**: TBD
- **On-Call Engineer**: TBD

### Escalation Procedures
1. **Level 1**: Automated monitoring alerts
2. **Level 2**: On-call engineer notification
3. **Level 3**: Management escalation
4. **Level 4**: External vendor support

---

**Document Version**: 1.0  
**Last Updated**: 2025-08-27  
**Next Review**: TBD  
**Owner**: AgentFlow Platform Team