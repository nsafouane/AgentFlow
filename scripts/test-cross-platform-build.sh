#!/bin/bash
# Cross-Platform Build Testing Script
# Tests builds on Linux, Windows, and WSL2 environments

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
SERVICES=("control-plane" "worker" "af")
PLATFORMS=("linux/amd64" "windows/amd64" "darwin/amd64")
BIN_DIR="bin"

echo -e "${BLUE}🔨 AgentFlow Cross-Platform Build Tests${NC}"
echo ""

# Function to check if a command exists
command_exists() {
    command -v "$1" >/dev/null 2>&1
}

# Function to detect WSL
is_wsl() {
    if [ -n "${WSL_DISTRO_NAME:-}" ]; then
        return 0
    fi
    
    if [ -f /proc/version ] && grep -qi "microsoft\|wsl" /proc/version; then
        return 0
    fi
    
    return 1
}

# Function to get current platform info
get_platform_info() {
    local os_name
    local arch_name
    
    case "$(uname -s)" in
        Linux*)
            if is_wsl; then
                os_name="WSL2"
            else
                os_name="Linux"
            fi
            ;;
        Darwin*)
            os_name="macOS"
            ;;
        CYGWIN*|MINGW*|MSYS*)
            os_name="Windows"
            ;;
        *)
            os_name="Unknown"
            ;;
    esac
    
    case "$(uname -m)" in
        x86_64|amd64)
            arch_name="amd64"
            ;;
        arm64|aarch64)
            arch_name="arm64"
            ;;
        *)
            arch_name="unknown"
            ;;
    esac
    
    echo "${os_name}/${arch_name}"
}

# Function to test cross-platform build
test_cross_platform_build() {
    local platform=$1
    local goos=${platform%/*}
    local goarch=${platform#*/}
    local platform_dir="${BIN_DIR}/${goos}"
    
    echo -e "${YELLOW}Testing build for ${platform}...${NC}"
    
    # Create platform directory
    mkdir -p "$platform_dir"
    
    local success=true
    local ext=""
    if [ "$goos" = "windows" ]; then
        ext=".exe"
    fi
    
    for service in "${SERVICES[@]}"; do
        local bin_path="${platform_dir}/${service}${ext}"
        
        echo -e "  Building ${service} for ${platform}..."
        
        # Build the service (each cmd is a separate module)
        if (cd "cmd/$service" && GOOS="$goos" GOARCH="$goarch" go build -o "../../$bin_path" . 2>/dev/null); then
            echo -e "  ${GREEN}✅ ${service} build succeeded${NC}"
            
            # Validate binary exists and has correct permissions
            if [ -f "$bin_path" ]; then
                if [ "$goos" != "windows" ]; then
                    chmod +x "$bin_path"
                fi
                echo -e "  ${GREEN}✅ Binary created: ${bin_path}${NC}"
            else
                echo -e "  ${RED}❌ Binary not found: ${bin_path}${NC}"
                success=false
            fi
        else
            echo -e "  ${RED}❌ ${service} build failed${NC}"
            success=false
        fi
    done
    
    if [ "$success" = true ]; then
        echo -e "${GREEN}✅ All builds succeeded for ${platform}${NC}"
        return 0
    else
        echo -e "${RED}❌ Some builds failed for ${platform}${NC}"
        return 1
    fi
}

# Function to test Makefile targets
test_makefile_targets() {
    echo -e "${YELLOW}Testing Makefile cross-platform targets...${NC}"
    
    if ! command_exists make; then
        echo -e "${YELLOW}⚠️  Make not found, skipping Makefile tests${NC}"
        return 0
    fi
    
    local targets=("build-linux" "build-windows" "build-all")
    local success=true
    
    for target in "${targets[@]}"; do
        echo -e "  Testing make ${target}..."
        
        if make "$target" >/dev/null 2>&1; then
            echo -e "  ${GREEN}✅ make ${target} succeeded${NC}"
        else
            echo -e "  ${RED}❌ make ${target} failed${NC}"
            success=false
        fi
    done
    
    if [ "$success" = true ]; then
        echo -e "${GREEN}✅ All Makefile targets succeeded${NC}"
        return 0
    else
        echo -e "${RED}❌ Some Makefile targets failed${NC}"
        return 1
    fi
}

# Function to test Taskfile targets
test_taskfile_targets() {
    echo -e "${YELLOW}Testing Taskfile cross-platform targets...${NC}"
    
    if ! command_exists task; then
        echo -e "${YELLOW}⚠️  Task not found, skipping Taskfile tests${NC}"
        return 0
    fi
    
    local targets=("build-linux" "build-windows" "build-all")
    local success=true
    
    for target in "${targets[@]}"; do
        echo -e "  Testing task ${target}..."
        
        if task "$target" >/dev/null 2>&1; then
            echo -e "  ${GREEN}✅ task ${target} succeeded${NC}"
        else
            echo -e "  ${RED}❌ task ${target} failed${NC}"
            success=false
        fi
    done
    
    if [ "$success" = true ]; then
        echo -e "${GREEN}✅ All Taskfile targets succeeded${NC}"
        return 0
    else
        echo -e "${RED}❌ Some Taskfile targets failed${NC}"
        return 1
    fi
}

# Function to test WSL2 specific functionality
test_wsl2_compatibility() {
    if ! is_wsl; then
        echo -e "${YELLOW}⚠️  Not running in WSL2, skipping WSL2-specific tests${NC}"
        return 0
    fi
    
    echo -e "${YELLOW}Testing WSL2 compatibility...${NC}"
    
    # Test that we can build Linux binaries in WSL2
    local test_bin="bin/test-wsl2"
    
    if (cd "cmd/af" && GOOS=linux GOARCH=amd64 go build -o "../../$test_bin" . 2>/dev/null); then
        echo -e "${GREEN}✅ WSL2 Linux build succeeded${NC}"
        
        # Test that the binary is executable
        if [ -x "$test_bin" ]; then
            echo -e "${GREEN}✅ WSL2 binary is executable${NC}"
        else
            echo -e "${RED}❌ WSL2 binary is not executable${NC}"
            return 1
        fi
        
        # Clean up
        rm -f "$test_bin"
        return 0
    else
        echo -e "${RED}❌ WSL2 Linux build failed${NC}"
        return 1
    fi
}

# Function to validate build artifacts
test_build_artifacts() {
    echo -e "${YELLOW}Validating build artifacts...${NC}"
    
    local success=true
    
    # Check Linux binaries
    if [ -d "${BIN_DIR}/linux" ]; then
        for service in "${SERVICES[@]}"; do
            local bin_path="${BIN_DIR}/linux/${service}"
            if [ -f "$bin_path" ]; then
                if [ -x "$bin_path" ]; then
                    echo -e "  ${GREEN}✅ Linux ${service} is executable${NC}"
                else
                    echo -e "  ${RED}❌ Linux ${service} is not executable${NC}"
                    success=false
                fi
            else
                echo -e "  ${YELLOW}⚠️  Linux ${service} not found${NC}"
            fi
        done
    fi
    
    # Check Windows binaries
    if [ -d "${BIN_DIR}/windows" ]; then
        for service in "${SERVICES[@]}"; do
            local bin_path="${BIN_DIR}/windows/${service}.exe"
            if [ -f "$bin_path" ]; then
                echo -e "  ${GREEN}✅ Windows ${service}.exe exists${NC}"
            else
                echo -e "  ${YELLOW}⚠️  Windows ${service}.exe not found${NC}"
            fi
        done
    fi
    
    if [ "$success" = true ]; then
        echo -e "${GREEN}✅ Build artifact validation passed${NC}"
        return 0
    else
        echo -e "${RED}❌ Build artifact validation failed${NC}"
        return 1
    fi
}

# Function to test Go module compatibility
test_go_modules() {
    echo -e "${YELLOW}Testing Go module compatibility...${NC}"
    
    # Test that all modules can be built
    local modules=("cmd/control-plane" "cmd/worker" "cmd/af" "sdk/go")
    local success=true
    
    # Test root module with go mod tidy instead of build
    echo -e "  Testing root module dependencies..."
    if go mod tidy >/dev/null 2>&1; then
        echo -e "  ${GREEN}✅ Root module dependencies are valid${NC}"
    else
        echo -e "  ${RED}❌ Root module dependencies failed${NC}"
        success=false
    fi
    
    for module in "${modules[@]}"; do
        if [ -f "${module}/go.mod" ]; then
            echo -e "  Testing module: ${module}"
            
            if (cd "$module" && go build . >/dev/null 2>&1); then
                echo -e "  ${GREEN}✅ Module ${module} builds successfully${NC}"
            else
                echo -e "  ${RED}❌ Module ${module} build failed${NC}"
                success=false
            fi
        fi
    done
    
    if [ "$success" = true ]; then
        echo -e "${GREEN}✅ All Go modules build successfully${NC}"
        return 0
    else
        echo -e "${RED}❌ Some Go modules failed to build${NC}"
        return 1
    fi
}

# Main test execution
main() {
    local exit_code=0
    
    echo -e "${BLUE}Current platform: $(get_platform_info)${NC}"
    echo -e "${BLUE}Go version: $(go version)${NC}"
    echo ""
    
    # Test Go modules first
    if ! test_go_modules; then
        exit_code=1
    fi
    echo ""
    
    # Test cross-platform builds
    for platform in "${PLATFORMS[@]}"; do
        if ! test_cross_platform_build "$platform"; then
            exit_code=1
        fi
        echo ""
    done
    
    # Test build tools
    if ! test_makefile_targets; then
        exit_code=1
    fi
    echo ""
    
    if ! test_taskfile_targets; then
        exit_code=1
    fi
    echo ""
    
    # Test WSL2 compatibility if applicable
    if ! test_wsl2_compatibility; then
        exit_code=1
    fi
    echo ""
    
    # Validate build artifacts
    if ! test_build_artifacts; then
        exit_code=1
    fi
    echo ""
    
    if [ $exit_code -eq 0 ]; then
        echo -e "${GREEN}🎉 All cross-platform build tests passed!${NC}"
    else
        echo -e "${RED}❌ Some cross-platform build tests failed${NC}"
    fi
    
    return $exit_code
}

# Run tests if script is executed directly
if [[ "${BASH_SOURCE[0]}" == "${0}" ]]; then
    main "$@"
fi