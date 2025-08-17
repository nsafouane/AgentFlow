# AgentFlow Development Environment Guide

This guide helps you set up a consistent development environment for AgentFlow across different platforms and scenarios.

## Quick Start (Recommended)

The fastest way to get started is using the VS Code devcontainer, which provides a pre-configured environment with all required tools.

### Prerequisites

- [VS Code](https://code.visualstudio.com/)
- [Docker Desktop](https://www.docker.com/products/docker-desktop/)
- [Dev Containers extension](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers)

### Setup Steps

1. **Clone the repository**
   ```bash
   git clone <repository-url>
   cd agentflow
   ```

2. **Open in VS Code**
   ```bash
   code .
   ```

3. **Reopen in Container**
   - VS Code will detect the `.devcontainer` configuration
   - Click "Reopen in Container" when prompted
   - Or use Command Palette: `Dev Containers: Reopen in Container`

4. **Verify Installation**
   ```bash
   af validate
   ```

The setup process takes 3-5 minutes on first run as it downloads and configures all tools.

## What's Included

The devcontainer includes all required development tools with pinned versions:

### Core Tools
- **Go 1.22+** - Primary development language
- **Docker** - Container runtime and builds
- **Task 3.35.1** - Cross-platform task runner
- **Make** - Build automation

### Database Tools
- **PostgreSQL Client 13** - Database operations
- **goose 3.18.0** - Database migrations
- **sqlc 1.25.0** - Type-safe SQL code generation

### Message Bus Tools
- **NATS CLI 0.1.4** - Message bus operations and testing

### Code Quality Tools
- **golangci-lint 1.55.2** - Go linting and static analysis
- **gosec 2.19.0** - Go security analyzer
- **gitleaks 8.18.1** - Secret detection
- **pre-commit 3.6.0** - Git hooks for code quality

### Development Services
- **PostgreSQL 15** - Development database
- **NATS 2.10** - Message bus with JetStream
- **Redis 7** - Caching and session storage

## Environment Validation

Use the `af validate` command to check your environment:

```bash
af validate
```

### Sample Output (Devcontainer)
```json
{
  "version": "1.0.0",
  "timestamp": "2024-01-01T00:00:00Z",
  "environment": {
    "platform": "linux",
    "architecture": "amd64", 
    "container": "devcontainer"
  },
  "tools": {
    "go": {
      "version": "1.22.0",
      "status": "ok"
    },
    "docker": {
      "version": "24.0.7",
      "status": "ok"
    }
  },
  "services": {
    "postgres": {
      "status": "available",
      "connection": "postgresql://agentflow@localhost:5432/agentflow_dev"
    },
    "nats": {
      "status": "available", 
      "connection": "nats://localhost:4222"
    }
  },
  "warnings": [],
  "errors": []
}
```

### Sample Output (Host System)
```json
{
  "environment": {
    "container": "host"
  },
  "warnings": [
    "Running on host system. Consider using VS Code devcontainer for consistent environment.",
    "PostgreSQL client (psql) is not installed",
    "NATS CLI is not installed"
  ]
}
```

## Windows Host Setup (Fallback)

If you cannot use the devcontainer, here's how to set up the development environment on Windows:

### Prerequisites

1. **Install Go 1.22+**
   - Download from [golang.org](https://golang.org/dl/)
   - Add to PATH: `C:\Program Files\Go\bin`
   - Verify: `go version`

2. **Install Git**
   - Download from [git-scm.com](https://git-scm.com/)
   - Use Git Bash for Unix-like commands

3. **Install Docker Desktop**
   - Download from [docker.com](https://www.docker.com/products/docker-desktop/)
   - Enable WSL2 backend for better performance

### Tool Installation

#### Using Chocolatey (Recommended)
```powershell
# Install Chocolatey first: https://chocolatey.org/install

# Install development tools
choco install make
choco install jq
choco install postgresql --params '/Password:password'

# Install Go tools
go install github.com/go-task/task/v3/cmd/task@v3.35.1
go install github.com/pressly/goose/v3/cmd/goose@v3.18.0
go install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.25.0
go install github.com/securecodewarrior/gosec/v2/cmd/gosec@v2.19.0
```

#### Manual Installation

1. **Task Runner**
   ```powershell
   # Download from GitHub releases
   $url = "https://github.com/go-task/task/releases/download/v3.35.1/task_windows_amd64.zip"
   Invoke-WebRequest -Uri $url -OutFile "task.zip"
   Expand-Archive -Path "task.zip" -DestinationPath "C:\tools\task"
   # Add C:\tools\task to PATH
   ```

2. **golangci-lint**
   ```powershell
   # Download and install
   $url = "https://github.com/golangci/golangci-lint/releases/download/v1.55.2/golangci-lint-1.55.2-windows-amd64.zip"
   Invoke-WebRequest -Uri $url -OutFile "golangci-lint.zip"
   Expand-Archive -Path "golangci-lint.zip" -DestinationPath "C:\tools\golangci-lint"
   # Add to PATH
   ```

3. **NATS CLI**
   ```powershell
   $url = "https://github.com/nats-io/natscli/releases/download/v0.1.4/nats-0.1.4-windows-amd64.zip"
   Invoke-WebRequest -Uri $url -OutFile "nats.zip"
   Expand-Archive -Path "nats.zip" -DestinationPath "C:\tools\nats"
   # Add to PATH
   ```

4. **gitleaks**
   ```powershell
   $url = "https://github.com/gitleaks/gitleaks/releases/download/v8.18.1/gitleaks_8.18.1_windows_x64.zip"
   Invoke-WebRequest -Uri $url -OutFile "gitleaks.zip"
   Expand-Archive -Path "gitleaks.zip" -DestinationPath "C:\tools\gitleaks"
   # Add to PATH
   ```

5. **pre-commit**
   ```powershell
   # Install Python first, then:
   pip install pre-commit==3.6.0
   ```

### Windows Path Configuration

Add these directories to your PATH environment variable:
```
C:\Program Files\Go\bin
C:\tools\task
C:\tools\golangci-lint\golangci-lint-1.55.2-windows-amd64
C:\tools\nats\nats-0.1.4-windows-amd64
C:\tools\gitleaks
%USERPROFILE%\go\bin
```

### Windows-Specific Notes

1. **PowerShell vs Command Prompt**
   - Use PowerShell 7+ for better compatibility
   - Git Bash provides Unix-like environment
   - Windows Terminal recommended for better experience

2. **Path Separators**
   - Use forward slashes in Go code: `path/to/file`
   - Use `filepath.Join()` for cross-platform paths
   - Be careful with hardcoded paths in scripts

3. **Line Endings**
   - Configure Git: `git config --global core.autocrlf true`
   - Use `.gitattributes` for consistent line endings
   - Pre-commit hooks will normalize line endings

4. **File Permissions**
   - Windows doesn't have Unix-style permissions
   - Some scripts may need WSL2 for full compatibility
   - Use `icacls` for Windows permission management

## macOS Host Setup (Fallback)

### Prerequisites

1. **Install Homebrew**
   ```bash
   /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"
   ```

2. **Install Xcode Command Line Tools**
   ```bash
   xcode-select --install
   ```

### Tool Installation

```bash
# Core tools
brew install go@1.22
brew install docker
brew install make
brew install jq
brew install postgresql@15

# Development tools
brew install go-task/tap/go-task
brew install golangci-lint
brew install gitleaks
brew install python3

# Install Go tools
go install github.com/pressly/goose/v3/cmd/goose@v3.18.0
go install github.com/sqlc-dev/sqlc/cmd/sqlc@v1.25.0
go install github.com/securecodewarrior/gosec/v2/cmd/gosec@v2.19.0

# Install NATS CLI
curl -L https://github.com/nats-io/natscli/releases/download/v0.1.4/nats-0.1.4-darwin-amd64.tar.gz | tar -xz
sudo mv nats-0.1.4-darwin-amd64/nats /usr/local/bin/

# Install pre-commit
pip3 install pre-commit==3.6.0
```

## Linux Host Setup (Fallback)

### Ubuntu/Debian

```bash
# Update package lists
sudo apt update

# Install core tools
sudo apt install -y golang-1.22 make jq postgresql-client-15 python3-pip

# Install Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh
sudo usermod -aG docker $USER

# Install Task
curl -sL https://github.com/go-task/task/releases/download/v3.35.1/task_linux_amd64.tar.gz | sudo tar -xz -C /usr/local/bin

# Install golangci-lint
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sudo sh -s -- -b /usr/local/bin v1.55.2

# Install other tools (same as macOS Go install commands)
```

### CentOS/RHEL/Fedora

```bash
# Install core tools
sudo dnf install -y golang make jq postgresql python3-pip

# Follow similar steps as Ubuntu, adjusting package manager commands
```

## Development Workflow

### Daily Development

1. **Start Services**
   ```bash
   # In devcontainer (automatic)
   # On host system:
   docker-compose -f .devcontainer/docker-compose.yml up -d
   ```

2. **Validate Environment**
   ```bash
   af validate
   ```

3. **Run Tests**
   ```bash
   task test
   # or
   make test
   ```

4. **Build Project**
   ```bash
   task build
   # or  
   make build
   ```

### Pre-commit Setup

The devcontainer automatically sets up pre-commit hooks. For host systems:

```bash
# Install hooks
pre-commit install

# Run hooks manually
pre-commit run --all-files

# Update hooks
pre-commit autoupdate
```

## Troubleshooting

### Common Issues

#### "Command not found" errors
- **Cause**: Tool not in PATH or not installed
- **Solution**: Run `af validate` to identify missing tools
- **Windows**: Check PATH environment variable
- **macOS/Linux**: Check `$PATH` and shell profile

#### Docker permission errors (Linux)
- **Cause**: User not in docker group
- **Solution**: `sudo usermod -aG docker $USER` then logout/login

#### PostgreSQL connection failures
- **Cause**: Service not running or wrong credentials
- **Solution**: Check `docker-compose ps` and service logs

#### Pre-commit hook failures
- **Cause**: Code quality issues or missing tools
- **Solution**: Fix reported issues or install missing tools

#### Windows path issues
- **Cause**: Backslash vs forward slash confusion
- **Solution**: Use `filepath.Join()` in Go code, check `.gitattributes`

### Getting Help

1. **Check validation output**: `af validate`
2. **Review logs**: Check Docker container logs
3. **Verify versions**: Ensure tool versions match requirements
4. **Clean rebuild**: Remove containers and rebuild
5. **Check documentation**: Review tool-specific documentation

### Performance Tips

1. **Use SSD storage** for better Docker performance
2. **Allocate sufficient RAM** to Docker (4GB+ recommended)
3. **Enable BuildKit** for faster Docker builds
4. **Use Go module cache** to speed up builds
5. **Configure IDE** for optimal Go development

## Messaging test logging (quiet by default)

The `pkg/messaging` package contains integration tests that intentionally exercise negative paths and emit diagnostic messages (for example, message hash validation failures). These logs can be very noisy during a full repo test run.

To keep full test runs quiet by default, noisy messaging diagnostics are gated behind the environment variable `AF_TEST_MESSAGING_NOISY`. Set it to `1` when you want to see messaging internals during development or debugging.

PowerShell example:

```powershell
$env:AF_TEST_MESSAGING_NOISY = '1'
go test -v -count=1 github.com/agentflow/agentflow/pkg/messaging
Remove-Item Env:AF_TEST_MESSAGING_NOISY
```

One-off (cmd wrapper) example:

```powershell
cmd /c "set AF_TEST_MESSAGING_NOISY=1&& go test -v -count=1 github.com/agentflow/agentflow/pkg/messaging"
```

If you want messaging integration tests to always run in CI with full diagnostics, configure your CI job to set `AF_TEST_MESSAGING_NOISY=1` for the messaging step.

## Environment Variables

### Required Variables (Devcontainer sets automatically)
```bash
GOPATH=/go
GOROOT=/usr/local/go
PATH=$PATH:$GOROOT/bin:$GOPATH/bin
```

### Optional Variables
```bash
# Development database
POSTGRES_URL=postgresql://agentflow:agentflow_dev_password@localhost:5432/agentflow_dev

# NATS connection
NATS_URL=nats://localhost:4222

# Redis connection  
REDIS_URL=redis://localhost:6379

# Development mode
AGENTFLOW_ENV=development
AGENTFLOW_LOG_LEVEL=debug
```

## Next Steps

Once your environment is set up:

1. **Explore the codebase**: Start with `README.md` and `docs/ARCHITECTURE.md`
2. **Run the test suite**: `task test` or `make test`
3. **Build the project**: `task build` or `make build`
4. **Review contributing guidelines**: `CONTRIBUTING.md`
5. **Check project status**: Review current development phase and tasks

For questions or issues, refer to the project documentation or open an issue in the repository.