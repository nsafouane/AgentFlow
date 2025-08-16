# Security Baseline & Exception Process

## Overview

This document establishes the security baseline for the AgentFlow project, defining security standards, tooling requirements, vulnerability management processes, and exception handling procedures. The security baseline ensures consistent security practices across all development phases and provides clear guidelines for maintaining security posture.

## Security Standards

### Severity Classification

AgentFlow uses a standardized severity classification system aligned with industry standards:

| Severity | Description | Response Time | Action Required |
|----------|-------------|---------------|-----------------|
| **Critical** | Vulnerabilities that can be exploited remotely with no authentication, leading to complete system compromise | Immediate (< 4 hours) | Block deployment, immediate fix required |
| **High** | Vulnerabilities that can lead to significant data exposure or system compromise with minimal user interaction | 24 hours | Block deployment, fix required before merge |
| **Medium** | Vulnerabilities that require user interaction or have limited impact scope | 7 days | Fix required, may be temporarily accepted with mitigation |
| **Low** | Vulnerabilities with minimal security impact or requiring significant user interaction | 30 days | Fix recommended, acceptable with documentation |
| **Info** | Security-related findings that don't represent immediate vulnerabilities | 90 days | Documentation and awareness, no blocking |

### Security Tools & Thresholds

#### Static Application Security Testing (SAST)

**Tool**: gosec  
**Threshold**: High/Critical findings block deployment  
**Configuration**: `.gosec.json` with project-specific rules  
**Scope**: All Go source code, excluding test files and generated code  

```json
{
  "severity": "medium",
  "confidence": "medium",
  "exclude": {
    "G101": false,
    "G102": false,
    "G103": false
  },
  "exclude-dirs": ["vendor", "node_modules", ".git"],
  "exclude-files": ["*_test.go", "*.pb.go"]
}
```

#### Dependency Vulnerability Scanning

**Tools**: govulncheck, OSV Scanner  
**Threshold**: High/Critical vulnerabilities block deployment  
**Database**: Latest vulnerability databases (updated daily)  
**Scope**: All direct and transitive dependencies  

**Update Policy**:
- Critical vulnerabilities: Update within 24 hours
- High vulnerabilities: Update within 7 days
- Medium vulnerabilities: Update within 30 days
- Low vulnerabilities: Update within 90 days

#### Secret Detection

**Tool**: gitleaks  
**Threshold**: Any secrets found block deployment  
**Configuration**: `.gitleaks.toml` with custom patterns  
**Scope**: Full git history and working directory  

**Patterns Detected**:
- API keys and tokens
- Database credentials
- Private keys and certificates
- Cloud provider credentials
- JWT secrets
- Custom application secrets

#### Container Security Scanning

**Tools**: syft (SBOM), grype (vulnerabilities)  
**Threshold**: High/Critical vulnerabilities in base images block deployment  
**Scope**: All container images and base layers  
**Base Image Policy**: Only use approved, regularly updated base images  

**Approved Base Images**:
- `golang:1.22-alpine` (for Go applications)
- `ubuntu:22.04` (for general purpose)
- `postgres:15-alpine` (for database)
- `redis:7-alpine` (for caching)

### Compliance Requirements

#### Software Bill of Materials (SBOM)

- **Format**: SPDX 2.3 and CycloneDX 1.4
- **Generation**: Automated for every build
- **Content**: All dependencies with version information
- **Attestation**: Signed with build provenance

#### Supply Chain Security

- **Signing**: All artifacts signed with Cosign keyless signing
- **Provenance**: SLSA Level 2 build provenance
- **Verification**: Signatures verified before deployment
- **Transparency**: All signatures logged in Rekor transparency log

## Vulnerability Management Process

### Detection and Assessment

1. **Automated Scanning**: Security tools run on every commit and daily scheduled scans
2. **Vulnerability Database Updates**: Daily updates to vulnerability databases
3. **Assessment**: Security team reviews findings within SLA timeframes
4. **Classification**: Vulnerabilities classified according to severity matrix

### Response Workflow

```mermaid
flowchart TD
    A[Vulnerability Detected] --> B{Severity Assessment}
    B -->|Critical| C[Immediate Response]
    B -->|High| D[24h Response]
    B -->|Medium| E[7d Response]
    B -->|Low| F[30d Response]
    
    C --> G[Block Deployment]
    D --> G
    E --> H[Create Issue]
    F --> H
    
    G --> I[Emergency Fix]
    H --> J[Planned Fix]
    
    I --> K[Security Review]
    J --> K
    
    K --> L[Deploy Fix]
    L --> M[Verify Resolution]
    M --> N[Close Issue]
```

### Remediation Priorities

1. **Critical/High**: Immediate remediation required
2. **Medium**: Planned remediation with temporary mitigation
3. **Low**: Scheduled remediation in next maintenance window
4. **Info**: Documentation and awareness, no immediate action

## Exception Process

### Exception Criteria

Security exceptions may be granted under the following conditions:

1. **False Positive**: Vulnerability doesn't apply to our use case
2. **Compensating Controls**: Alternative security measures are in place
3. **Business Critical**: Fixing would break critical business functionality
4. **No Fix Available**: No patch or workaround exists from vendor
5. **Acceptable Risk**: Risk is within acceptable tolerance levels

### Exception Request Process

#### 1. Exception Request Submission

Submit exception request using the following template:

```yaml
# Security Exception Request
exception_id: "SEC-YYYY-NNNN"
date_requested: "YYYY-MM-DD"
requestor: "name@company.com"
approver: "security-team@company.com"

vulnerability:
  id: "CVE-YYYY-NNNN"
  severity: "High"
  tool: "gosec"
  description: "Brief description of vulnerability"
  affected_component: "component/file/dependency"

justification:
  type: "false_positive|compensating_controls|business_critical|no_fix|acceptable_risk"
  reason: "Detailed explanation of why exception is needed"
  business_impact: "Impact if vulnerability is not fixed immediately"

risk_assessment:
  likelihood: "very_low|low|medium|high|very_high"
  impact: "low|medium|high|critical"
  overall_risk: "low|medium|high|critical"
  
mitigation:
  controls: "List of compensating controls or mitigations"
  monitoring: "Additional monitoring or detection measures"
  timeline: "Expected timeline for permanent fix"

expiry:
  date: "YYYY-MM-DD"
  review_date: "YYYY-MM-DD"
  auto_expire: true
```

#### 2. Security Review

- **Initial Review**: Security team reviews within 2 business days
- **Risk Assessment**: Detailed risk analysis performed
- **Stakeholder Input**: Relevant teams consulted as needed
- **Decision**: Approve, reject, or request modifications

#### 3. Approval Workflow

| Severity | Approver Required | Additional Requirements |
|----------|------------------|------------------------|
| Critical | CISO + Engineering Manager | Executive approval |
| High | Security Team Lead + Engineering Manager | Risk committee review |
| Medium | Security Team Member | Engineering team lead approval |
| Low | Security Team Member | Self-approval with documentation |

#### 4. Exception Implementation

- **Documentation**: Exception recorded in security register
- **Configuration**: Security tools configured to ignore specific findings
- **Monitoring**: Additional monitoring implemented if required
- **Communication**: Relevant teams notified of exception

#### 5. Exception Management

- **Regular Review**: Monthly review of all active exceptions
- **Expiry Handling**: Automatic expiry with renewal process
- **Audit Trail**: Complete audit trail maintained
- **Reporting**: Regular reporting to management on exception status

### Exception Configuration

#### gosec Exceptions

```json
{
  "exclude": {
    "G101": ["file1.go", "file2.go"],
    "G102": ["server.go:25-30"]
  },
  "exclude-generated": true,
  "exclude-dirs": ["vendor", "test"]
}
```

#### gitleaks Exceptions

```toml
[allowlist]
description = "Approved exceptions"
files = [
  "docs/examples/config.yaml",
  "test/fixtures/secrets.txt"
]
regexes = [
  "EXAMPLE_",
  "TEST_SECRET_"
]
```

#### Dependency Exceptions

```yaml
# .security-exceptions.yml
dependencies:
  - name: "vulnerable-package"
    version: "1.0.0"
    cve: "CVE-2021-12345"
    exception_id: "SEC-2024-001"
    expires: "2024-12-31"
    reason: "No fix available, compensating controls in place"
```

## Security Monitoring & Alerting

### Continuous Monitoring

- **Daily Scans**: Automated security scans run daily
- **Real-time Alerts**: Critical findings trigger immediate alerts
- **Trend Analysis**: Weekly analysis of security trends
- **Compliance Reporting**: Monthly compliance status reports

### Alert Configuration

| Event | Severity | Notification | Response Time |
|-------|----------|--------------|---------------|
| Critical vulnerability found | Critical | Immediate (Slack + Email + PagerDuty) | < 1 hour |
| High vulnerability found | High | Within 1 hour (Slack + Email) | < 4 hours |
| Secret detected | Critical | Immediate (Slack + Email + PagerDuty) | < 1 hour |
| Security scan failure | Medium | Within 4 hours (Slack) | < 24 hours |
| Exception expiry | Low | 7 days before expiry (Email) | Before expiry |

### Metrics & KPIs

#### Security Metrics

- **Mean Time to Detection (MTTD)**: Average time to detect vulnerabilities
- **Mean Time to Resolution (MTTR)**: Average time to fix vulnerabilities
- **Vulnerability Density**: Number of vulnerabilities per KLOC
- **Exception Rate**: Percentage of findings with approved exceptions
- **Compliance Score**: Percentage of security controls implemented

#### Target KPIs

- MTTD: < 24 hours
- MTTR: < 7 days (High), < 30 days (Medium)
- Vulnerability Density: < 1 per KLOC
- Exception Rate: < 5%
- Compliance Score: > 95%

## Tool Configuration & Maintenance

### Tool Updates

- **Security Tools**: Updated monthly or when critical updates available
- **Vulnerability Databases**: Updated daily
- **Rule Sets**: Reviewed quarterly and updated as needed
- **Thresholds**: Reviewed annually or after significant changes

### Configuration Management

- **Version Control**: All security configurations stored in git
- **Change Management**: Changes require security team approval
- **Testing**: Configuration changes tested in staging environment
- **Rollback**: Rollback procedures documented and tested

### Performance Optimization

- **Scan Duration**: Target < 10 minutes for full security scan
- **False Positive Rate**: Target < 5% false positive rate
- **Resource Usage**: Optimize for CI/CD pipeline efficiency
- **Caching**: Implement caching for improved performance

## Training & Awareness

### Developer Training

- **Secure Coding**: Annual secure coding training required
- **Tool Usage**: Training on security tools and processes
- **Incident Response**: Training on security incident procedures
- **Best Practices**: Regular sharing of security best practices

### Security Champions

- **Program**: Security champion program in each development team
- **Responsibilities**: Promote security awareness and best practices
- **Training**: Additional security training for champions
- **Communication**: Regular security updates and communications

## Compliance & Audit

### Regulatory Compliance

- **Standards**: Compliance with OWASP, NIST, and industry standards
- **Documentation**: Comprehensive documentation of security controls
- **Evidence**: Audit trail and evidence collection
- **Reporting**: Regular compliance reporting to stakeholders

### Internal Audits

- **Frequency**: Quarterly internal security audits
- **Scope**: Review of security controls and processes
- **Findings**: Documentation and remediation of audit findings
- **Improvement**: Continuous improvement based on audit results

### External Audits

- **Frequency**: Annual external security assessments
- **Scope**: Comprehensive security review by third party
- **Certification**: Maintain relevant security certifications
- **Remediation**: Address external audit findings promptly

---

**Document Version**: 1.0  
**Last Updated**: 2024-01-16  
**Next Review**: 2024-04-16  
**Owner**: Security Team  
**Approver**: CISO