# Operational Runbooks

This directory contains operational runbooks for troubleshooting and maintaining the AgentFlow system. These runbooks provide step-by-step procedures for common operational scenarios and incident response.

## Available Runbooks

### Build & CI/CD Issues
- [Build Failure Troubleshooting](build-failure.md) - Diagnose and resolve CI/CD pipeline failures
- [Security Scan Failures](security-scan-failures.md) - Handle security tool failures and vulnerability remediation
- [Cross-Platform Build Issues](cross-platform-builds.md) - Resolve platform-specific build problems

### System Operations
- [Message Backlog Management](message-backlog.md) - Handle NATS JetStream message queue backlogs
- [Database Migration Issues](database-migrations.md) - Troubleshoot schema migration problems
- [Container Registry Issues](container-registry.md) - Resolve image push/pull and signing problems

### Performance & Monitoring
- [Cost Spike Investigation](cost-spike.md) - Investigate and mitigate unexpected cost increases
- [Performance Degradation](performance-degradation.md) - Diagnose and resolve performance issues
- [Resource Exhaustion](resource-exhaustion.md) - Handle memory, CPU, and storage issues

### Security Incidents
- [Security Incident Response](security-incident.md) - Respond to security alerts and breaches
- [Certificate Expiration](certificate-expiration.md) - Handle SSL/TLS certificate renewals
- [Access Control Issues](access-control.md) - Troubleshoot authentication and authorization problems

## Runbook Structure

Each runbook follows a standardized format:

1. **Symptoms** - How to identify the issue
2. **Immediate Actions** - Steps to take immediately
3. **Investigation** - How to diagnose the root cause
4. **Resolution** - Step-by-step fix procedures
5. **Prevention** - How to prevent recurrence
6. **Escalation** - When and how to escalate

## Quick Reference

### Emergency Contacts
- **On-Call Engineer**: [To be defined in future specs]
- **Security Team**: [To be defined in future specs]
- **Infrastructure Team**: [To be defined in future specs]

### Critical System URLs
- **CI/CD Dashboard**: [To be defined in Q1.2 Messaging spec]
- **Monitoring Dashboard**: [To be defined in Q1.8 Observability spec]
- **Log Aggregation**: [To be defined in Q1.8 Observability spec]

### Common Commands
```bash
# Validate environment
af validate

# Check system health
af health --verbose

# View recent logs
af logs --tail=100

# Emergency stop
af stop --force
```

## Contributing to Runbooks

When creating or updating runbooks:

1. Follow the standardized structure above
2. Include actual commands and examples
3. Test procedures in a safe environment
4. Link to relevant monitoring dashboards
5. Update this index when adding new runbooks

## Related Documentation

- [Architecture Documentation](../ARCHITECTURE.md)
- [Security Baseline](../security-baseline.md)
- [Risk Register](../risk-register.yaml)
- [ADR Directory](../adr/)

---

**Note**: This is a living document that will be expanded as the AgentFlow system evolves. Many runbooks reference future specifications that will be implemented in subsequent development phases.

**Last Updated**: 2025-08-16  
**Next Review**: Q1 Gate G0 Review