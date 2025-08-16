# Cross-Platform Build Troubleshooting Guide

This guide provides troubleshooting steps for cross-platform build issues in AgentFlow.

## Overview

AgentFlow supports building on multiple platforms:
- **Linux** (amd64, arm64)
- **Windows** (amd64)
- **macOS** (amd64, arm64)
- **WSL2** (Windows Subsystem for Linux)

## Quick Validation

To quickly validate your cross-platform build setup:

```bash
# Using Make
make test-cross-platform

# Using Task
task test-cross-platform

# Direct Go execution
go run scripts/test-cross-platform-build.go
```

## Common Issues and Solutions

### 1. Go Cross-Compilation Issues

#### Problem: "unsupported GOOS/GOARCH pair"
```
go build: unsupported GOOS/GOARCH pair linux/arm
```

**Solution:**
- Verify supported GOOS/GOARCH combinations:
  ```bash
  go tool dist list
  ```
- Use correct architecture names:
  - `amd64` (not `x86_64`)
  - `arm64` (not `aarch64`)

#### Problem: CGO-related build failures
```
# github.com/some/package
cgo: C compiler "gcc" not found
```

**Solution:**
- Disable CGO for cross-compilation:
  ```bash
  CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build
  ```
- Or install cross-compilation toolchain:
  ```bash
  # Ubuntu/Debian
  sudo apt-get install gcc-multilib
  
  # macOS
  xcode-select --install
  ```

### 2. Windows-Specific Issues

#### Problem: Path separator issues
```
cannot find package "./cmd\control-plane"
```

**Solution:**
- Always use forward slashes in Go import paths:
  ```bash
  go build ./cmd/control-plane  # ✅ Correct
  go build .\cmd\control-plane  # ❌ Wrong
  ```

#### Problem: Missing .exe extension
```
Binary not found: bin/windows/control-plane
```

**Solution:**
- Ensure Windows binaries have .exe extension:
  ```bash
  GOOS=windows go build -o bin/windows/control-plane.exe ./cmd/control-plane
  ```

#### Problem: PowerShell execution policy
```
execution of scripts is disabled on this system
```

**Solution:**
- Set execution policy for current session:
  ```powershell
  Set-ExecutionPolicy -ExecutionPolicy Bypass -Scope Process
  ```
- Or run with bypass flag:
  ```powershell
  powershell -ExecutionPolicy Bypass -File script.ps1
  ```

### 3. WSL2-Specific Issues

#### Problem: WSL2 not detected
```
Not running in WSL2, skipping WSL2-specific tests
```

**Solution:**
- Verify WSL2 environment variables:
  ```bash
  echo $WSL_DISTRO_NAME
  cat /proc/version | grep -i microsoft
  ```

#### Problem: File permission issues in WSL2
```
permission denied: ./bin/linux/control-plane
```

**Solution:**
- Set executable permissions:
  ```bash
  chmod +x bin/linux/*
  ```
- Or build with proper permissions:
  ```bash
  GOOS=linux go build -o bin/linux/control-plane ./cmd/control-plane
  chmod +x bin/linux/control-plane
  ```

#### Problem: Windows/Linux path mixing
```
cannot access 'C:\Users\...\bin\linux\control-plane'
```

**Solution:**
- Use WSL2 filesystem paths:
  ```bash
  # ✅ Correct - WSL2 path
  /mnt/c/Users/username/project/bin/linux/control-plane
  
  # ❌ Wrong - Windows path in WSL2
  C:\Users\username\project\bin\linux\control-plane
  ```

### 4. macOS-Specific Issues

#### Problem: Code signing requirements
```
"control-plane" cannot be opened because the developer cannot be verified
```

**Solution:**
- For development builds, disable Gatekeeper temporarily:
  ```bash
  sudo spctl --master-disable
  ```
- Or sign the binary (for distribution):
  ```bash
  codesign -s "Developer ID Application: Your Name" bin/darwin/control-plane
  ```

#### Problem: Architecture mismatch on Apple Silicon
```
bad CPU type in executable
```

**Solution:**
- Build for correct architecture:
  ```bash
  # For Apple Silicon (M1/M2)
  GOOS=darwin GOARCH=arm64 go build -o bin/darwin-arm64/control-plane ./cmd/control-plane
  
  # For Intel Macs
  GOOS=darwin GOARCH=amd64 go build -o bin/darwin-amd64/control-plane ./cmd/control-plane
  ```

### 5. Build Tool Issues

#### Problem: Make not found on Windows
```
'make' is not recognized as an internal or external command
```

**Solutions:**
1. **Use Task instead:**
   ```bash
   # Install Task
   go install github.com/go-task/task/v3/cmd/task@latest
   
   # Use Task commands
   task build-all
   ```

2. **Install Make for Windows:**
   ```bash
   # Using Chocolatey
   choco install make
   
   # Using Scoop
   scoop install make
   ```

3. **Use PowerShell scripts directly:**
   ```powershell
   powershell -ExecutionPolicy Bypass -File scripts/test-cross-platform-build.ps1
   ```

#### Problem: Task not found
```
task: command not found
```

**Solution:**
- Install Task:
  ```bash
  # Using Go
  go install github.com/go-task/task/v3/cmd/task@latest
  
  # Using package managers
  # macOS
  brew install go-task/tap/go-task
  
  # Ubuntu/Debian
  sudo snap install task --classic
  
  # Windows
  choco install go-task
  ```

### 6. Module and Dependency Issues

#### Problem: Module not found in subdirectories
```
go: cannot find main module, but found .git/config
```

**Solution:**
- Ensure you're in the correct directory:
  ```bash
  # Run from project root
  cd /path/to/agentflow
  go build ./cmd/control-plane
  ```

#### Problem: Dependency version conflicts
```
go: inconsistent vendoring
```

**Solution:**
- Clean and update modules:
  ```bash
  go clean -modcache
  go mod tidy
  go mod download
  ```

### 7. Container Build Issues

#### Problem: Docker buildx not available
```
docker: 'buildx' is not a docker command
```

**Solution:**
- Enable Docker buildx:
  ```bash
  # Enable buildx
  docker buildx install
  
  # Create builder instance
  docker buildx create --use --name multiarch
  ```

#### Problem: Multi-arch build failures
```
multiple platforms feature is currently not supported for docker driver
```

**Solution:**
- Use buildx with proper driver:
  ```bash
  docker buildx create --driver docker-container --use
  docker buildx build --platform linux/amd64,linux/arm64 .
  ```

## Environment Validation

### Required Tools

Verify you have the required tools installed:

```bash
# Go version (1.22+)
go version

# Git
git --version

# Docker (for container builds)
docker --version

# Optional: Task
task --version

# Optional: Make
make --version
```

### Environment Variables

Set these environment variables for consistent builds:

```bash
# Go configuration
export GOOS=linux          # Target OS
export GOARCH=amd64         # Target architecture
export CGO_ENABLED=0        # Disable CGO for cross-compilation

# Build configuration
export BIN_DIR=bin          # Binary output directory
export REGISTRY=ghcr.io     # Container registry
export IMAGE_NAME=agentflow/agentflow  # Image name
export TAG=latest           # Image tag
```

### Platform Detection

The build scripts automatically detect your platform:

```bash
# Linux
uname -s  # Linux
uname -m  # x86_64

# macOS
uname -s  # Darwin
uname -m  # x86_64 or arm64

# Windows (in PowerShell)
$env:OS                    # Windows_NT
[Environment]::OSVersion   # Version info

# WSL2
echo $WSL_DISTRO_NAME      # Ubuntu, etc.
cat /proc/version | grep -i microsoft  # Should contain "microsoft"
```

## Testing Your Setup

### Basic Build Test

```bash
# Test basic build
go build ./cmd/af

# Test cross-platform build
GOOS=linux GOARCH=amd64 go build -o bin/test-linux ./cmd/af
GOOS=windows GOARCH=amd64 go build -o bin/test-windows.exe ./cmd/af

# Clean up
rm -f bin/test-*
```

### Full Validation

```bash
# Run comprehensive cross-platform tests
make test-cross-platform

# Or using Task
task test-cross-platform

# Or directly
go run scripts/test-cross-platform-build.go
```

## Performance Optimization

### Build Speed

1. **Use build cache:**
   ```bash
   export GOCACHE=/path/to/cache
   ```

2. **Parallel builds:**
   ```bash
   # Build multiple targets in parallel
   make -j4 build-all
   ```

3. **Incremental builds:**
   ```bash
   # Only rebuild changed packages
   go build -i ./...
   ```

### Disk Space

1. **Clean build artifacts:**
   ```bash
   make clean
   go clean -cache
   go clean -modcache
   ```

2. **Use .dockerignore:**
   ```
   bin/
   .git/
   *.log
   ```

## Getting Help

If you continue to experience issues:

1. **Check the build logs:**
   ```bash
   make build-all 2>&1 | tee build.log
   ```

2. **Verify your environment:**
   ```bash
   go env
   ```

3. **Run with verbose output:**
   ```bash
   go build -v ./cmd/control-plane
   ```

4. **Check for known issues:**
   - Review GitHub issues
   - Check the project documentation
   - Consult the development team

## Contributing

If you discover new cross-platform build issues or solutions:

1. Document the issue and solution
2. Add test cases to the validation scripts
3. Update this troubleshooting guide
4. Submit a pull request

## References

- [Go Cross Compilation](https://golang.org/doc/install/source#environment)
- [Docker Buildx Documentation](https://docs.docker.com/buildx/)
- [Task Documentation](https://taskfile.dev/)
- [Make Documentation](https://www.gnu.org/software/make/manual/)
- [WSL2 Documentation](https://docs.microsoft.com/en-us/windows/wsl/)