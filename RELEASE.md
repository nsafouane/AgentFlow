# AgentFlow Release Engineering Guide

## Versioning Scheme

AgentFlow follows [Semantic Versioning 2.0.0](https://semver.org/) with pre-1.0 adaptations for rapid development.

### Version Format: MAJOR.MINOR.PATCH

- **MAJOR**: Incompatible API changes (reserved for 1.0+ releases)
- **MINOR**: New functionality in a backwards compatible manner
- **PATCH**: Backwards compatible bug fixes

### Pre-1.0 Versioning Rules

During the pre-1.0 development phase (0.x.y), we use a modified semantic versioning approach:

- **0.MINOR.PATCH** format
- **MINOR** increments for breaking changes (what would be MAJOR in 1.0+)
- **PATCH** increments for new features and bug fixes
- Breaking changes are acceptable and expected in 0.x releases

### Version Examples

```
0.1.0 - Initial foundation release
0.1.1 - Bug fixes and minor improvements
0.2.0 - Breaking changes to API structure
0.2.1 - New features without breaking changes
0.3.0 - Major architectural changes
1.0.0 - First stable release with API compatibility guarantees
```

## Tagging Policy

### Tag Format

- **Release Tags**: `v{MAJOR}.{MINOR}.{PATCH}` (e.g., `v0.1.0`, `v1.2.3`)
- **Pre-release Tags**: `v{MAJOR}.{MINOR}.{PATCH}-{IDENTIFIER}` (e.g., `v0.1.0-alpha.1`, `v0.1.0-beta.2`, `v0.1.0-rc.1`)

### Tag Creation Rules

1. **Automated Tagging**: Tags are created automatically by CI/CD pipeline on merge to main
2. **Manual Tagging**: Emergency releases may be tagged manually by maintainers
3. **Tag Protection**: Release tags are protected and cannot be deleted or moved
4. **Signed Tags**: All release tags must be GPG signed

### Pre-release Identifiers

- **alpha**: Early development, may have significant bugs
- **beta**: Feature complete, undergoing testing
- **rc** (release candidate): Stable, final testing before release

## Branching Model

### Branch Structure

```
main (protected)
├── develop (integration branch)
├── feature/feature-name
├── hotfix/issue-description
└── release/v0.x.0
```

### Branch Policies

- **main**: Production-ready code, protected branch
- **develop**: Integration branch for ongoing development
- **feature/***: Feature development branches
- **hotfix/***: Critical bug fixes for production
- **release/***: Release preparation branches

### Merge Strategy

- **Feature branches**: Squash and merge to develop
- **Release branches**: Merge commit to main (preserves release history)
- **Hotfix branches**: Merge to both main and develop
- **No direct commits** to main or develop branches

## Release Process

### 1. Release Planning

- Review milestone completion
- Validate all exit criteria
- Update CHANGELOG.md
- Create release branch from develop

### 2. Release Preparation

```bash
# Create release branch
git checkout develop
git pull origin develop
git checkout -b release/v0.x.0

# Update version in relevant files
./scripts/update-version.sh 0.x.0

# Update CHANGELOG.md
# Add release notes and migration guide if needed

# Commit version updates
git add .
git commit -m "chore: prepare release v0.x.0"
git push origin release/v0.x.0
```

### 3. Release Testing

- Run full test suite including integration tests
- Execute security scans
- Perform cross-platform builds
- Validate container images and signatures

### 4. Release Execution

```bash
# Merge release branch to main
git checkout main
git merge --no-ff release/v0.x.0
git push origin main

# Tag the release (automated by CI)
# CI will create tag v0.x.0 and trigger release workflow

# Merge back to develop
git checkout develop
git merge --no-ff release/v0.x.0
git push origin develop

# Delete release branch
git branch -d release/v0.x.0
git push origin --delete release/v0.x.0
```

### 5. Post-Release

- Verify release artifacts are published
- Update documentation
- Announce release
- Monitor for issues

## Hotfix Process

### Critical Bug Fixes

```bash
# Create hotfix branch from main
git checkout main
git pull origin main
git checkout -b hotfix/critical-issue-fix

# Implement fix
# Update version (patch increment)
./scripts/update-version.sh 0.x.y

# Update CHANGELOG.md with hotfix details

# Commit and push
git add .
git commit -m "fix: critical issue description"
git push origin hotfix/critical-issue-fix

# Create PR to main
# After approval, merge to main and develop
```

## Version Management Scripts

### Version Update Script

Location: `scripts/update-version.sh`

```bash
#!/bin/bash
# Updates version across all relevant files
NEW_VERSION=$1
if [ -z "$NEW_VERSION" ]; then
    echo "Usage: $0 <version>"
    exit 1
fi

# Update go.mod version comment
sed -i "s/\/\/ version: .*/\/\/ version: v$NEW_VERSION/" go.mod

# Update version in CLI
sed -i "s/Version: \".*\"/Version: \"$NEW_VERSION\"/" cmd/af/version.go

# Update Docker labels
sed -i "s/LABEL version=\".*\"/LABEL version=\"$NEW_VERSION\"/" Dockerfile

echo "Updated version to $NEW_VERSION"
```

### Version Parsing Script

Location: `scripts/parse-version.sh`

```bash
#!/bin/bash
# Parses and validates version strings
VERSION=$1

if [[ ! $VERSION =~ ^v?([0-9]+)\.([0-9]+)\.([0-9]+)(-[a-zA-Z0-9.-]+)?$ ]]; then
    echo "Invalid version format: $VERSION"
    exit 1
fi

MAJOR=${BASH_REMATCH[1]}
MINOR=${BASH_REMATCH[2]}
PATCH=${BASH_REMATCH[3]}
PRERELEASE=${BASH_REMATCH[4]}

echo "MAJOR=$MAJOR"
echo "MINOR=$MINOR"
echo "PATCH=$PATCH"
echo "PRERELEASE=$PRERELEASE"
```

## CI/CD Integration

### Automated Release Workflow

The CI/CD pipeline automatically handles:

1. **Version Detection**: Detects version changes in release commits
2. **Tag Creation**: Creates signed git tags for releases
3. **Artifact Building**: Builds multi-architecture containers and binaries
4. **Artifact Signing**: Signs all artifacts with Cosign
5. **SBOM Generation**: Creates software bill of materials
6. **Release Publishing**: Publishes to GitHub Releases and container registry

### Release Triggers

- **Automatic**: Merge to main branch with version change
- **Manual**: Workflow dispatch with version parameter
- **Scheduled**: Nightly pre-releases from develop branch

## Quality Gates

### Pre-Release Validation

Before any release, the following must pass:

- [ ] All CI/CD pipelines green
- [ ] Security scans pass (no High/Critical vulnerabilities)
- [ ] Cross-platform builds successful
- [ ] Integration tests pass
- [ ] Documentation updated
- [ ] CHANGELOG.md updated
- [ ] Migration guide provided (if needed)

### Release Artifacts

Each release must include:

- [ ] Signed container images (multi-arch)
- [ ] Binary releases for supported platforms
- [ ] SBOM and provenance attestation
- [ ] Release notes and changelog
- [ ] Migration documentation (if applicable)

## Emergency Release Procedures

### Critical Security Fixes

1. **Immediate Response**: Create hotfix branch from main
2. **Minimal Changes**: Fix only the security issue
3. **Expedited Review**: Security team review required
4. **Fast-Track Release**: Skip normal release timeline
5. **Security Advisory**: Publish security advisory with CVE if applicable

### Rollback Procedures

1. **Identify Issue**: Confirm release needs rollback
2. **Revert Tag**: Create new release reverting problematic changes
3. **Communicate**: Notify users of rollback and timeline
4. **Root Cause**: Investigate and document failure
5. **Process Improvement**: Update release process to prevent recurrence

## Version Compatibility

### API Compatibility

- **Pre-1.0**: No compatibility guarantees, breaking changes allowed
- **1.0+**: Semantic versioning compatibility guarantees
- **Deprecation**: 2 minor version deprecation notice before removal

### Database Migrations

- **Forward Compatibility**: New versions can read old data
- **Migration Path**: Automated migrations for schema changes
- **Rollback Support**: Database rollback procedures documented

## Release Communication

### Release Notes Template

```markdown
# AgentFlow v0.x.0

## 🚀 New Features
- Feature 1 description
- Feature 2 description

## 🐛 Bug Fixes
- Bug fix 1 description
- Bug fix 2 description

## 💥 Breaking Changes
- Breaking change 1 with migration guide
- Breaking change 2 with migration guide

## 📚 Documentation
- Documentation updates
- New guides and tutorials

## 🔒 Security
- Security improvements
- Vulnerability fixes

## ⚡ Performance
- Performance improvements
- Optimization details

## 🛠️ Internal Changes
- Refactoring and internal improvements
- Developer experience enhancements

## 📦 Dependencies
- Dependency updates
- Security updates

## Migration Guide
[Detailed migration instructions if needed]

## Full Changelog
[Link to full changelog]
```

### Communication Channels

- **GitHub Releases**: Primary release announcement
- **Documentation**: Updated with new features
- **Security Advisories**: For security-related releases
- **Community**: Discord/Slack announcements for major releases

## Metrics and Monitoring

### Release Metrics

- Release frequency
- Time from feature complete to release
- Hotfix frequency
- Rollback frequency
- Release artifact download counts

### Quality Metrics

- Test coverage per release
- Security scan results
- Performance regression detection
- User-reported issues per release

This release engineering guide ensures consistent, reliable, and secure releases while maintaining development velocity and quality standards.