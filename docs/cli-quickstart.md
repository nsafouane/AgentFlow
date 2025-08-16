# AgentFlow CLI Quickstart

The AgentFlow CLI (`af`) is a command-line tool for managing AgentFlow development environments, validating configurations, and performing common development tasks.

## Installation

The CLI is built as part of the AgentFlow project. To build and install:

```bash
# From the project root
cd cmd/af
go build -o af .

# On Windows
go build -o af.exe .

# Install globally (optional)
go install .
```

## Commands

### `af validate`

Validates your development environment and outputs a comprehensive JSON report of tool availability, service connectivity, and configuration status.

#### Usage

```bash
af validate
```

#### Output

The command outputs a JSON structure containing:

- **version**: CLI version
- **timestamp**: Validation timestamp in RFC3339 format
- **environment**: Platform information (OS, architecture, container type)
- **tools**: Status of required development tools
- **services**: Connectivity status of required services
- **warnings**: Non-critical issues that should be addressed
- **errors**: Critical issues that prevent proper operation

#### Example Output

```json
{
  "version": "1.0.0",
  "timestamp": "2025-08-16T11:02:59Z",
  "environment": {
    "platform": "windows",
    "architecture": "amd64",
    "container": "host"
  },
  "tools": {
    "go": {
      "version": "1.25.0",
      "status": "ok"
    },
    "docker": {
      "version": "28.0.4",
      "status": "ok"
    },
    "golangci-lint": {
      "version": "",
      "status": "warning",
      "message": "golangci-lint is not installed"
    }
  },
  "services": {
    "postgres": {
      "status": "unavailable",
      "connection": "Failed to connect to PostgreSQL at localhost:5432"
    },
    "nats": {
      "status": "unavailable",
      "connection": "Failed to connect to NATS at localhost:4222"
    }
  },
  "warnings": [
    "golangci-lint is not installed",
    "Running on host system. Consider using VS Code devcontainer for consistent environment."
  ],
  "errors": []
}
```

## Environment Detection

The CLI automatically detects your development environment:

### Container Types

- **devcontainer**: Running in VS Code devcontainer
- **codespaces**: Running in GitHub Codespaces
- **docker**: Running in a Docker container
- **host**: Running on the host system

### Platform Support

- **Linux**: Full support with all tools
- **Windows**: Full support with Windows-specific path handling
- **macOS**: Full support (Darwin)

## Tool Validation

The CLI validates the following development tools:

### Required Tools

- **Go**: Go programming language (1.22+)
- **Docker**: Container runtime
- **Task**: Cross-platform task runner (Taskfile.yml)

### Development Tools

- **golangci-lint**: Go linting tool
- **gosec**: Go security analyzer
- **gitleaks**: Secret detection tool
- **pre-commit**: Pre-commit hooks framework

### Database Tools

- **psql**: PostgreSQL client
- **goose**: Database migration tool
- **sqlc**: Type-safe SQL code generator

### Message Bus Tools

- **nats**: NATS CLI for message bus operations

## Service Connectivity

The CLI tests connectivity to required services:

- **PostgreSQL**: Database connectivity (localhost:5432)
- **NATS**: Message bus connectivity (localhost:4222)

## Status Codes

### Tool Status

- **ok**: Tool is installed and functional
- **warning**: Tool is missing or has issues (non-critical)
- **error**: Tool is missing and required for operation

### Service Status

- **available**: Service is running and accessible
- **unavailable**: Service is not accessible
- **unknown**: Service status could not be determined

## Exit Codes

- **0**: Validation completed successfully (warnings allowed)
- **1**: Validation failed with critical errors

## Development Environment Setup

### Recommended: VS Code DevContainer

For the most consistent development experience, use the VS Code devcontainer:

1. Open the project in VS Code
2. Install the "Dev Containers" extension
3. Press `Ctrl+Shift+P` (or `Cmd+Shift+P` on macOS)
4. Select "Dev Containers: Reopen in Container"
5. Wait for the container to build and start
6. Run `af validate` to verify the environment

### Host System Setup

If you prefer to work on your host system, ensure the following tools are installed:

#### Windows

```powershell
# Install Go
winget install GoLang.Go

# Install Docker Desktop
winget install Docker.DockerDesktop

# Install Task
winget install Task.Task

# Install other tools via Go
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
go install github.com/zricethezav/gitleaks/v8@latest
go install github.com/pressly/goose/v3/cmd/goose@latest
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

# Install pre-commit (requires Python)
pip install pre-commit
```

#### Linux/macOS

```bash
# Install Go (version 1.22+)
# Follow instructions at https://golang.org/doc/install

# Install Docker
# Follow instructions at https://docs.docker.com/get-docker/

# Install Task
sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d

# Install other tools via Go
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install github.com/securecodewarrior/gosec/v2/cmd/gosec@latest
go install github.com/zricethezav/gitleaks/v8@latest
go install github.com/pressly/goose/v3/cmd/goose@latest
go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

# Install pre-commit
pip install pre-commit
# or on macOS with Homebrew:
brew install pre-commit
```

## Troubleshooting

### Common Issues

#### "Command not found" errors

Ensure your `$PATH` (or `%PATH%` on Windows) includes:
- Go binary directory (`$GOPATH/bin` or `%GOPATH%\bin`)
- Local binary installations

#### Service connectivity failures

1. Ensure Docker is running
2. Start required services:
   ```bash
   # Using docker-compose (if available)
   docker-compose up -d postgres nats
   
   # Or start services individually
   docker run -d --name postgres -p 5432:5432 -e POSTGRES_PASSWORD=password postgres:15
   docker run -d --name nats -p 4222:4222 nats:latest
   ```

#### Windows path issues

The CLI handles Windows paths automatically, but ensure:
- Use PowerShell or Command Prompt (not Git Bash for some operations)
- Paths with spaces are properly quoted
- Use forward slashes in configuration files when possible

### Getting Help

- Check the validation output for specific error messages
- Review the warnings for non-critical issues
- Ensure all required tools are in your PATH
- Consider using the devcontainer for a pre-configured environment

## Integration with Development Workflow

### Pre-commit Hooks

Add validation to your pre-commit hooks:

```yaml
# .pre-commit-config.yaml
repos:
  - repo: local
    hooks:
      - id: af-validate
        name: AgentFlow Environment Validation
        entry: af validate
        language: system
        pass_filenames: false
        always_run: true
```

### CI/CD Integration

Use in CI/CD pipelines to validate the build environment:

```yaml
# GitHub Actions example
- name: Validate Environment
  run: |
    cd cmd/af
    go run . validate
    if [ $? -ne 0 ]; then
      echo "Environment validation failed"
      exit 1
    fi
```

### IDE Integration

Configure your IDE to run validation:

#### VS Code Tasks

```json
{
  "version": "2.0.0",
  "tasks": [
    {
      "label": "Validate Environment",
      "type": "shell",
      "command": "cd cmd/af && go run . validate",
      "group": "build",
      "presentation": {
        "echo": true,
        "reveal": "always",
        "focus": false,
        "panel": "shared"
      }
    }
  ]
}
```

## Future Commands

The CLI is designed to be extensible. Future versions will include:

- `af init`: Initialize new AgentFlow projects
- `af deploy`: Deploy AgentFlow services
- `af config`: Manage configuration
- `af logs`: View service logs
- `af status`: Check service status
- `af migrate`: Run database migrations