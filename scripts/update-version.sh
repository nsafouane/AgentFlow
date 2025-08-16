#!/bin/bash
# AgentFlow Version Update Script
# Updates version across all relevant files in the project

set -e

NEW_VERSION=$1
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

usage() {
    echo "Usage: $0 <version>"
    echo "Example: $0 0.1.0"
    echo "Example: $0 0.2.0-alpha.1"
    exit 1
}

validate_version() {
    local version=$1
    if [[ ! $version =~ ^[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9.-]+)?$ ]]; then
        echo -e "${RED}Error: Invalid version format: $version${NC}"
        echo "Version must follow semantic versioning: MAJOR.MINOR.PATCH[-PRERELEASE]"
        exit 1
    fi
}

update_file() {
    local file=$1
    local pattern=$2
    local replacement=$3
    
    if [ -f "$file" ]; then
        if sed -i.bak "$pattern" "$file" 2>/dev/null; then
            rm -f "$file.bak"
            echo -e "${GREEN}✓${NC} Updated $file"
        else
            echo -e "${YELLOW}⚠${NC} Could not update $file (file may not contain expected pattern)"
        fi
    else
        echo -e "${YELLOW}⚠${NC} File not found: $file"
    fi
}

main() {
    if [ -z "$NEW_VERSION" ]; then
        echo -e "${RED}Error: Version argument required${NC}"
        usage
    fi

    validate_version "$NEW_VERSION"

    echo -e "${GREEN}Updating AgentFlow version to: $NEW_VERSION${NC}"
    echo

    cd "$PROJECT_ROOT"

    # Update go.mod version comment
    update_file "go.mod" "s|// version: .*|// version: v$NEW_VERSION|" 

    # Update version in CLI (create if doesn't exist)
    if [ ! -f "cmd/af/version.go" ]; then
        mkdir -p cmd/af
        cat > cmd/af/version.go << EOF
package main

// Version information for AgentFlow CLI
const (
    Version = "$NEW_VERSION"
    BuildDate = ""
    GitCommit = ""
)
EOF
        echo -e "${GREEN}✓${NC} Created cmd/af/version.go"
    else
        update_file "cmd/af/version.go" "s|Version = \".*\"|Version = \"$NEW_VERSION\"|"
    fi

    # Update Dockerfile labels
    update_file "Dockerfile" "s|LABEL version=\".*\"|LABEL version=\"$NEW_VERSION\"|"

    # Update docker-compose.yml if it exists
    if [ -f "docker-compose.yml" ]; then
        update_file "docker-compose.yml" "s|image: agentflow.*|image: agentflow:$NEW_VERSION|g"
    fi

    # Update Helm chart if it exists
    if [ -f "deploy/helm/Chart.yaml" ]; then
        update_file "deploy/helm/Chart.yaml" "s|version: .*|version: $NEW_VERSION|"
        update_file "deploy/helm/Chart.yaml" "s|appVersion: .*|appVersion: $NEW_VERSION|"
    fi

    # Update package.json files if they exist (for SDK or dashboard)
    find . -name "package.json" -not -path "./node_modules/*" | while read -r package_file; do
        if [ -f "$package_file" ]; then
            update_file "$package_file" "s|\"version\": \".*\"|\"version\": \"$NEW_VERSION\"|"
        fi
    done

    # Update Python setup files if they exist
    find . -name "setup.py" -o -name "pyproject.toml" | while read -r python_file; do
        if [ -f "$python_file" ]; then
            if [[ "$python_file" == *"setup.py" ]]; then
                update_file "$python_file" "s|version=\".*\"|version=\"$NEW_VERSION\"|"
            else
                update_file "$python_file" "s|version = \".*\"|version = \"$NEW_VERSION\"|"
            fi
        fi
    done

    echo
    echo -e "${GREEN}Version update completed successfully!${NC}"
    echo
    echo "Next steps:"
    echo "1. Review the changes: git diff"
    echo "2. Update CHANGELOG.md with release notes"
    echo "3. Commit the changes: git add . && git commit -m 'chore: bump version to v$NEW_VERSION'"
    echo "4. Create release branch: git checkout -b release/v$NEW_VERSION"
}

main "$@"