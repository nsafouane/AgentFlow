# SBOM & Provenance Verification Procedures

## Overview

This document provides comprehensive procedures for verifying Software Bill of Materials (SBOM) and provenance attestation for AgentFlow artifacts. These procedures ensure supply chain security and compliance with security requirements.

## Prerequisites

### Required Tools

- **Docker**: For container image inspection and pulling
- **Cosign**: For signature and attestation verification
- **Syft**: For SBOM generation and validation
- **jq**: For JSON processing and validation

### Installation

#### Linux/WSL2
```bash
# Install Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sh get-docker.sh

# Install Cosign
curl -O -L "https://github.com/sigstore/cosign/releases/latest/download/cosign-linux-amd64"
sudo mv cosign-linux-amd64 /usr/local/bin/cosign
sudo chmod +x /usr/local/bin/cosign

# Install Syft
curl -sSfL https://raw.githubusercontent.com/anchore/syft/main/install.sh | sh -s -- -b /usr/local/bin

# Install jq
sudo apt-get update && sudo apt-get install -y jq
```

#### Windows (PowerShell)
```powershell
# Install Docker Desktop
# Download from https://www.docker.com/products/docker-desktop

# Install Cosign
Invoke-WebRequest -Uri "https://github.com/sigstore/cosign/releases/latest/download/cosign-windows-amd64.exe" -OutFile "cosign.exe"
Move-Item cosign.exe C:\Windows\System32\

# Install Syft
Invoke-WebRequest -Uri "https://github.com/anchore/syft/releases/latest/download/syft_windows_amd64.zip" -OutFile "syft.zip"
Expand-Archive syft.zip -DestinationPath C:\Windows\System32\

# Install jq
# Download from https://stedolan.github.io/jq/download/
```

#### macOS
```bash
# Install using Homebrew
brew install docker cosign syft jq
```

## Verification Procedures

### 1. Automated Verification

#### Using Validation Scripts

**Linux/macOS:**
```bash
# Validate all artifacts (containers and local files)
./scripts/validate-sbom-provenance.sh

# Validate specific image tag
./scripts/validate-sbom-provenance.sh v1.0.0

# Validate only container artifacts
./scripts/validate-sbom-provenance.sh latest true false

# Validate only local SBOM files
./scripts/validate-sbom-provenance.sh latest false true
```

**Windows:**
```powershell
# Validate all artifacts
.\scripts\validate-sbom-provenance.ps1

# Validate specific image tag
.\scripts\validate-sbom-provenance.ps1 -ImageTag "v1.0.0"

# Validate only container artifacts
.\scripts\validate-sbom-provenance.ps1 -ValidateLocal $false

# Validate only local SBOM files
.\scripts\validate-sbom-provenance.ps1 -ValidateContainers $false
```

#### Using Unit Tests

```bash
# Run SBOM and provenance validation tests
go test -v ./scripts/validate-sbom-provenance_test.go

# Run specific test
go test -v ./scripts/validate-sbom-provenance_test.go -run TestSPDXSBOMValidation
```

### 2. Manual Verification

#### Interactive Manual Testing

```bash
# Run all manual tests
./scripts/test-sbom-provenance-manual.sh

# Run specific test category
./scripts/test-sbom-provenance-manual.sh latest artifacts
./scripts/test-sbom-provenance-manual.sh latest provenance
./scripts/test-sbom-provenance-manual.sh latest signatures
./scripts/test-sbom-provenance-manual.sh latest local
```

#### Step-by-Step Manual Verification

##### Container Image SBOM Verification

1. **Pull the container image:**
   ```bash
   docker pull ghcr.io/agentflow/agentflow/control-plane:latest
   ```

2. **Generate SBOM using Syft:**
   ```bash
   # SPDX format
   syft ghcr.io/agentflow/agentflow/control-plane:latest -o spdx-json=sbom.spdx.json
   
   # CycloneDX format
   syft ghcr.io/agentflow/agentflow/control-plane:latest -o cyclonedx-json=sbom.cyclonedx.json
   ```

3. **Validate SBOM structure:**
   ```bash
   # Check SPDX SBOM
   jq '.spdxVersion, (.packages | length)' sbom.spdx.json
   
   # Check CycloneDX SBOM
   jq '.specVersion, (.components | length)' sbom.cyclonedx.json
   ```

##### Container Signature Verification

1. **Verify container signature:**
   ```bash
   cosign verify \
     --certificate-identity-regexp="https://github.com/agentflow/agentflow" \
     --certificate-oidc-issuer="https://token.actions.githubusercontent.com" \
     ghcr.io/agentflow/agentflow/control-plane:latest
   ```

2. **Expected output:**
   - Verification successful message
   - Certificate details
   - Signature metadata

##### Provenance Attestation Verification

1. **Verify provenance attestation:**
   ```bash
   cosign verify-attestation \
     --certificate-identity-regexp="https://github.com/agentflow/agentflow" \
     --certificate-oidc-issuer="https://token.actions.githubusercontent.com" \
     --type slsaprovenance \
     ghcr.io/agentflow/agentflow/control-plane:latest
   ```

2. **Decode and inspect provenance:**
   ```bash
   cosign verify-attestation \
     --certificate-identity-regexp="https://github.com/agentflow/agentflow" \
     --certificate-oidc-issuer="https://token.actions.githubusercontent.com" \
     --type slsaprovenance \
     ghcr.io/agentflow/agentflow/control-plane:latest | \
     jq -r '.payload' | base64 -d | jq '.'
   ```

### 3. CI/CD Integration Verification

#### GitHub Actions Workflow Validation

1. **Check workflow configuration:**
   ```bash
   # Verify SBOM generation is enabled
   grep -r "sbom.*true\|syft" .github/workflows/

   # Verify provenance attestation is enabled
   grep -r "provenance.*true\|attestation" .github/workflows/

   # Verify signing is enabled
   grep -r "cosign" .github/workflows/
   ```

2. **Validate workflow execution:**
   - Check GitHub Actions runs for successful SBOM generation
   - Verify attestation creation in workflow logs
   - Confirm artifact signing completion

#### Artifact Registry Verification

1. **Check published artifacts:**
   ```bash
   # List available tags
   docker search ghcr.io/agentflow/agentflow

   # Inspect manifest for SBOM metadata
   docker manifest inspect ghcr.io/agentflow/agentflow/control-plane:latest
   ```

2. **Verify artifact metadata:**
   - SBOM annotations in manifest
   - Signature presence
   - Attestation availability

## SBOM Structure Validation

### SPDX Format Requirements

A valid SPDX SBOM must contain:

- **spdxVersion**: SPDX specification version (e.g., "SPDX-2.3")
- **dataLicense**: License for SBOM data (typically "CC0-1.0")
- **SPDXID**: Unique identifier for the document
- **name**: Name of the analyzed artifact
- **documentNamespace**: Unique namespace URI
- **creationInfo**: Creation metadata including tools and timestamp
- **packages**: Array of software packages with metadata
- **relationships**: Dependencies and relationships between packages

### CycloneDX Format Requirements

A valid CycloneDX SBOM must contain:

- **bomFormat**: Format identifier ("CycloneDX")
- **specVersion**: CycloneDX specification version (e.g., "1.4")
- **serialNumber**: Unique identifier for the BOM
- **version**: BOM version number
- **metadata**: Creation metadata including tools and timestamp
- **components**: Array of software components with metadata

### Validation Criteria

#### SBOM Content Validation

1. **Package/Component Count**: Must contain at least 1 package/component
2. **Version Information**: Packages should include version information where available
3. **License Information**: License data should be present for identified packages
4. **Dependency Relationships**: Relationships between packages should be documented

#### SBOM Quality Metrics

- **Coverage**: Percentage of dependencies with complete metadata
- **Accuracy**: Correctness of version and license information
- **Completeness**: Presence of all required SPDX/CycloneDX fields
- **Freshness**: SBOM generation timestamp should be recent

## Provenance Attestation Validation

### SLSA Provenance Requirements

A valid SLSA provenance attestation must contain:

- **_type**: Statement type ("https://in-toto.io/Statement/v0.1")
- **subject**: Array of artifacts with names and digests
- **predicateType**: Provenance type ("https://slsa.dev/provenance/v0.2")
- **predicate**: Provenance metadata including:
  - **builder**: Builder information and ID
  - **buildType**: Type of build system used
  - **invocation**: Build invocation details
  - **buildConfig**: Build configuration
  - **metadata**: Build metadata and timestamps

### Verification Steps

1. **Attestation Presence**: Verify attestation exists for artifact
2. **Signature Validation**: Confirm attestation is properly signed
3. **Content Validation**: Validate attestation structure and content
4. **Builder Verification**: Confirm builder identity matches expected
5. **Source Verification**: Validate source repository and commit information

## Troubleshooting

### Common Issues

#### SBOM Generation Failures

**Issue**: Syft fails to generate SBOM
```
Error: failed to catalog: unable to determine file type
```

**Solution**:
1. Ensure image is accessible and properly tagged
2. Check network connectivity to registry
3. Verify Syft version compatibility
4. Try with different output formats

#### Signature Verification Failures

**Issue**: Cosign verification fails
```
Error: no matching signatures
```

**Solutions**:
1. Verify image reference is correct and complete
2. Check certificate identity and OIDC issuer parameters
3. Ensure image was signed (not applicable for local builds)
4. Verify network access to transparency log

#### Attestation Verification Failures

**Issue**: Provenance attestation not found
```
Error: no matching attestations
```

**Solutions**:
1. Confirm attestation type is correct ("slsaprovenance")
2. Verify image was built with attestation enabled
3. Check GitHub Actions workflow configuration
4. Ensure proper permissions for attestation creation

### Debugging Commands

#### Verbose Verification

```bash
# Enable verbose output for debugging
export COSIGN_EXPERIMENTAL=1

# Verify with detailed output
cosign verify --verbose \
  --certificate-identity-regexp="https://github.com/agentflow/agentflow" \
  --certificate-oidc-issuer="https://token.actions.githubusercontent.com" \
  ghcr.io/agentflow/agentflow/control-plane:latest
```

#### SBOM Debugging

```bash
# Generate SBOM with debug output
syft ghcr.io/agentflow/agentflow/control-plane:latest -v

# Validate JSON structure
jq empty sbom.spdx.json && echo "Valid JSON" || echo "Invalid JSON"

# Check specific SBOM fields
jq '.spdxVersion, .packages[0].name, .creationInfo.created' sbom.spdx.json
```

## Compliance and Reporting

### Security Compliance

This verification process supports compliance with:

- **NIST SSDF**: Secure Software Development Framework
- **SLSA**: Supply-chain Levels for Software Artifacts
- **SPDX**: Software Package Data Exchange standard
- **CycloneDX**: OWASP CycloneDX standard

### Audit Trail

Maintain records of:

1. **Verification Results**: Success/failure status for each artifact
2. **SBOM Content**: Package inventories and vulnerability status
3. **Provenance Data**: Build source and process information
4. **Signature Status**: Cryptographic verification results
5. **Timestamps**: When verification was performed

### Reporting

Generate compliance reports using:

```bash
# Generate verification report
./scripts/validate-sbom-provenance.sh latest true true > verification-report.txt

# Extract key metrics
grep -E "(SUCCESS|ERROR|WARNING)" verification-report.txt | sort | uniq -c
```

## Integration with Development Workflow

### Pre-commit Validation

Add SBOM validation to pre-commit hooks:

```yaml
# .pre-commit-config.yaml
repos:
  - repo: local
    hooks:
      - id: sbom-validation
        name: Validate local SBOM files
        entry: ./scripts/validate-sbom-provenance.sh
        args: [latest, false, true]
        language: system
        pass_filenames: false
```

### CI/CD Integration

Ensure validation runs in CI/CD:

```yaml
# .github/workflows/ci.yml
- name: Validate SBOM and Provenance
  run: |
    ./scripts/validate-sbom-provenance.sh latest true true
```

### Release Validation

Include in release process:

```bash
# Before creating release
./scripts/validate-sbom-provenance.sh $RELEASE_TAG true true

# Verify release artifacts
./scripts/test-sbom-provenance-manual.sh $RELEASE_TAG all
```

## References

- [SPDX Specification](https://spdx.github.io/spdx-spec/)
- [CycloneDX Specification](https://cyclonedx.org/specification/overview/)
- [SLSA Provenance](https://slsa.dev/provenance/v0.2)
- [Cosign Documentation](https://docs.sigstore.dev/cosign/overview/)
- [Syft Documentation](https://github.com/anchore/syft)
- [NIST SSDF](https://csrc.nist.gov/Projects/ssdf)