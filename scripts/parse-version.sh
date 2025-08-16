#!/bin/bash
# AgentFlow Version Parsing Script
# Parses and validates version strings according to semantic versioning

set -e

VERSION=$1
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"

# Color codes for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

usage() {
    echo "Usage: $0 <version>"
    echo "Examples:"
    echo "  $0 v1.2.3"
    echo "  $0 1.2.3"
    echo "  $0 0.1.0-alpha.1"
    echo "  $0 v2.0.0-beta.2"
    exit 1
}

parse_version() {
    local version=$1
    
    # Remove 'v' prefix if present
    version=${version#v}
    
    # Validate version format
    if [[ ! $version =~ ^([0-9]+)\.([0-9]+)\.([0-9]+)(-([a-zA-Z0-9.-]+))?$ ]]; then
        echo -e "${RED}Error: Invalid version format: $version${NC}" >&2
        echo -e "${YELLOW}Expected format: MAJOR.MINOR.PATCH[-PRERELEASE]${NC}" >&2
        echo -e "${YELLOW}Examples: 1.2.3, 0.1.0-alpha.1, 2.0.0-beta.2${NC}" >&2
        exit 1
    fi
    
    local major=${BASH_REMATCH[1]}
    local minor=${BASH_REMATCH[2]}
    local patch=${BASH_REMATCH[3]}
    local prerelease=${BASH_REMATCH[5]}
    
    # Output parsed components
    echo "MAJOR=$major"
    echo "MINOR=$minor"
    echo "PATCH=$patch"
    echo "PRERELEASE=${prerelease:-}"
    echo "FULL_VERSION=$version"
    echo "TAG_VERSION=v$version"
    
    # Determine version type
    if [ -n "$prerelease" ]; then
        echo "VERSION_TYPE=prerelease"
        if [[ $prerelease == alpha* ]]; then
            echo "PRERELEASE_TYPE=alpha"
        elif [[ $prerelease == beta* ]]; then
            echo "PRERELEASE_TYPE=beta"
        elif [[ $prerelease == rc* ]]; then
            echo "PRERELEASE_TYPE=rc"
        else
            echo "PRERELEASE_TYPE=other"
        fi
    else
        echo "VERSION_TYPE=release"
        echo "PRERELEASE_TYPE="
    fi
    
    # Determine stability
    if [ "$major" = "0" ]; then
        echo "STABILITY=unstable"
        echo "API_COMPATIBILITY=none"
    else
        echo "STABILITY=stable"
        echo "API_COMPATIBILITY=semantic"
    fi
}

increment_version() {
    local version=$1
    local increment_type=$2
    
    # Parse current version
    version=${version#v}
    if [[ ! $version =~ ^([0-9]+)\.([0-9]+)\.([0-9]+)(-([a-zA-Z0-9.-]+))?$ ]]; then
        echo -e "${RED}Error: Invalid version format for increment: $version${NC}" >&2
        exit 1
    fi
    
    local major=${BASH_REMATCH[1]}
    local minor=${BASH_REMATCH[2]}
    local patch=${BASH_REMATCH[3]}
    
    case $increment_type in
        major)
            major=$((major + 1))
            minor=0
            patch=0
            ;;
        minor)
            minor=$((minor + 1))
            patch=0
            ;;
        patch)
            patch=$((patch + 1))
            ;;
        *)
            echo -e "${RED}Error: Invalid increment type: $increment_type${NC}" >&2
            echo -e "${YELLOW}Valid types: major, minor, patch${NC}" >&2
            exit 1
            ;;
    esac
    
    echo "NEXT_VERSION=$major.$minor.$patch"
    echo "NEXT_TAG_VERSION=v$major.$minor.$patch"
}

compare_versions() {
    local version1=$1
    local version2=$2
    
    # Remove 'v' prefix if present
    version1=${version1#v}
    version2=${version2#v}
    
    # Parse versions
    if [[ ! $version1 =~ ^([0-9]+)\.([0-9]+)\.([0-9]+)(-([a-zA-Z0-9.-]+))?$ ]]; then
        echo -e "${RED}Error: Invalid version1 format: $version1${NC}" >&2
        exit 1
    fi
    local major1=${BASH_REMATCH[1]} minor1=${BASH_REMATCH[2]} patch1=${BASH_REMATCH[3]} pre1=${BASH_REMATCH[5]}
    
    if [[ ! $version2 =~ ^([0-9]+)\.([0-9]+)\.([0-9]+)(-([a-zA-Z0-9.-]+))?$ ]]; then
        echo -e "${RED}Error: Invalid version2 format: $version2${NC}" >&2
        exit 1
    fi
    local major2=${BASH_REMATCH[1]} minor2=${BASH_REMATCH[2]} patch2=${BASH_REMATCH[3]} pre2=${BASH_REMATCH[5]}
    
    # Compare major.minor.patch
    if [ "$major1" -gt "$major2" ]; then
        echo "COMPARISON=greater"
    elif [ "$major1" -lt "$major2" ]; then
        echo "COMPARISON=less"
    elif [ "$minor1" -gt "$minor2" ]; then
        echo "COMPARISON=greater"
    elif [ "$minor1" -lt "$minor2" ]; then
        echo "COMPARISON=less"
    elif [ "$patch1" -gt "$patch2" ]; then
        echo "COMPARISON=greater"
    elif [ "$patch1" -lt "$patch2" ]; then
        echo "COMPARISON=less"
    else
        # Same major.minor.patch, compare prerelease
        if [ -z "$pre1" ] && [ -z "$pre2" ]; then
            echo "COMPARISON=equal"
        elif [ -z "$pre1" ] && [ -n "$pre2" ]; then
            echo "COMPARISON=greater"  # Release > prerelease
        elif [ -n "$pre1" ] && [ -z "$pre2" ]; then
            echo "COMPARISON=less"     # Prerelease < release
        else
            # Both have prerelease, lexicographic comparison
            if [ "$pre1" = "$pre2" ]; then
                echo "COMPARISON=equal"
            elif [ "$pre1" \> "$pre2" ]; then
                echo "COMPARISON=greater"
            else
                echo "COMPARISON=less"
            fi
        fi
    fi
}

validate_version_sequence() {
    local current_version=$1
    local next_version=$2
    
    # Parse and compare versions
    eval "$(compare_versions "$current_version" "$next_version")"
    
    if [ "$COMPARISON" = "greater" ] || [ "$COMPARISON" = "equal" ]; then
        echo "SEQUENCE_VALID=true"
        echo "SEQUENCE_TYPE=forward"
    else
        echo "SEQUENCE_VALID=false"
        echo "SEQUENCE_TYPE=backward"
        echo -e "${YELLOW}Warning: Version sequence goes backward${NC}" >&2
        echo -e "${YELLOW}Current: $current_version, Next: $next_version${NC}" >&2
    fi
}

main() {
    case "${2:-parse}" in
        parse)
            if [ -z "$VERSION" ]; then
                echo -e "${RED}Error: Version argument required${NC}" >&2
                usage
            fi
            parse_version "$VERSION"
            ;;
        increment)
            if [ -z "$VERSION" ] || [ -z "$3" ]; then
                echo -e "${RED}Error: Version and increment type required${NC}" >&2
                echo "Usage: $0 <version> increment <major|minor|patch>" >&2
                exit 1
            fi
            increment_version "$VERSION" "$3"
            ;;
        compare)
            if [ -z "$VERSION" ] || [ -z "$3" ]; then
                echo -e "${RED}Error: Two versions required for comparison${NC}" >&2
                echo "Usage: $0 <version1> compare <version2>" >&2
                exit 1
            fi
            compare_versions "$VERSION" "$3"
            ;;
        validate)
            if [ -z "$VERSION" ] || [ -z "$3" ]; then
                echo -e "${RED}Error: Current and next versions required${NC}" >&2
                echo "Usage: $0 <current_version> validate <next_version>" >&2
                exit 1
            fi
            validate_version_sequence "$VERSION" "$3"
            ;;
        *)
            echo -e "${RED}Error: Unknown command: ${2:-parse}${NC}" >&2
            echo "Available commands: parse, increment, compare, validate" >&2
            exit 1
            ;;
    esac
}

main "$@"