# Devcontainer Adoption Guide

## Overview

AgentFlow uses VS Code devcontainers to provide a consistent, standardized development environment across all platforms. This guide explains how to adopt and use the devcontainer for AgentFlow development.

## Why Use Devcontainers?

### Benefits
- **Consistent Environment**: Same tools, versions, and configuration across all developers
- **Quick Setup**: Get started in minutes without manual tool installation
- **Cross-Platform**: Works identically on Windows, macOS, and Linux
- **Isolated Dependencies**: No conflicts with your host system tools
- **Pre-configured Tools**: All required development tools pre-installed and configured

### What's Included
- Go 1.22+ with module support
- NATS client tools for message bus testing
- PostgreSQL client for database operations
- Docker and Docker Compose
- Security tools (gosec, gitleaks, golangci-lint)
- Database tools (goose, sqlc)
- Pre-commit hooks for code quality
- Task runner for cross-platform builds

## Getting Started

### Prerequisites
- [VS Code](https://code.visualstudio.com/) installed
- [Docker Desktop](https://www.docker.com/products/docker-desktop/) installed and running
- [Dev Containers extension](https://marketplace.visualstudio.com/items?itemName=ms-vscode-remote.remote-containers) for VS Code

### Opening in Devcontainer

1. **Clone the Repository**
   ```bash
   git clone https://github.com/agentflow/agentflow.git
   cd agentflow
   ```

2. **Open in VS Code**
   ```bash
   code .
   ```

3. **Reopen in Container**
   - VS Code should automatically detect the devcontainer configuration
   - Click "Reopen in Container" when prompted
   - Or use Command Palette (Ctrl+Shift+P): "Dev Containers: Reopen in Container"

4. **Wait for Setup**
   - First-time setup takes 2-5 minutes to download and build the container
   - Subsequent starts are much faster (30-60 seconds)

### Verification

Once the devcontainer is running, verify your environment:

```bash
af validate
```

You should see:
- `"container": "devcontainer"` in the environment section
- No warnings about using devcontainer
- All required tools showing as "ok" status

## Environment Detection

The `af validate` command automatically detects your environment:

### Container Types
- **`host`**: Running directly on your machine (will show warnings)
- **`devcontainer`**: Running in VS Code devcontainer (recommended)
- **`codespaces`**: Running in GitHub Codespaces
- **`docker`**: Running in a generic Docker container

### Warning System
When running on the host system, `af validate` will display warnings:
- Recommendation to use devcontainer
- Benefits of standardized environment
- Instructions for switching to devcontainer

## Troubleshooting

### Common Issues

#### Docker Not Running
**Error**: "Cannot connect to the Docker daemon"
**Solution**: Start Docker Desktop and ensure it's running

#### Permission Issues (Linux/WSL)
**Error**: Permission denied accessing Docker socket
**Solution**: 
```bash
sudo usermod -aG docker $USER
# Log out and back in, or restart your session
```

#### Slow Container Startup
**Issue**: Container takes a long time to start
**Solutions**:
- Ensure Docker Desktop has sufficient resources allocated
- Close other resource-intensive applications
- Consider using Docker Desktop's WSL2 backend on Windows

#### Port Conflicts
**Issue**: Services can't bind to ports (5432, 4222, etc.)
**Solution**: Stop conflicting services on your host system:
```bash
# Check what's using the port
netstat -tulpn | grep :5432
# Stop the conflicting service
sudo systemctl stop postgresql  # Example for PostgreSQL
```

### Windows-Specific Issues

#### WSL2 Integration
Ensure Docker Desktop has WSL2 integration enabled:
1. Open Docker Desktop Settings
2. Go to Resources → WSL Integration
3. Enable integration with your WSL2 distribution

#### Path Issues
If you encounter path-related errors:
- Ensure you're using forward slashes in paths within the container
- Use the integrated terminal in VS Code (not external terminals)

### Performance Optimization

#### Resource Allocation
Allocate sufficient resources to Docker Desktop:
- **Memory**: At least 4GB (8GB recommended)
- **CPU**: At least 2 cores (4 cores recommended)
- **Disk**: At least 20GB free space

#### Volume Mounting
The devcontainer uses bind mounts for optimal performance:
- Source code is mounted from your host system
- Dependencies are cached in named volumes
- This provides fast file access while maintaining isolation

## Advanced Usage

### Customizing the Environment

#### Personal Settings
Your VS Code settings and extensions are automatically synced:
- Settings sync works across devcontainer sessions
- Extensions are automatically installed in the container
- Git configuration is inherited from your host system

#### Additional Tools
To add tools to your personal devcontainer:
1. Create `.devcontainer/devcontainer.local.json` (gitignored)
2. Extend the base configuration:
   ```json
   {
     "name": "AgentFlow Dev (Personal)",
     "dockerComposeFile": ["docker-compose.yml"],
     "service": "devcontainer",
     "workspaceFolder": "/workspace",
     "postCreateCommand": "your-custom-setup.sh"
   }
   ```

### Multiple Containers
The devcontainer includes supporting services:
- **PostgreSQL**: Database for development and testing
- **NATS**: Message bus for inter-service communication
- **Redis**: Caching and session storage

Access these services at:
- PostgreSQL: `localhost:5432`
- NATS: `localhost:4222`
- Redis: `localhost:6379`

## Best Practices

### Development Workflow
1. Always use `af validate` to check your environment
2. Run tests frequently: `make test` or `task test`
3. Use pre-commit hooks: `pre-commit install`
4. Build regularly: `make build` or `task build`

### Code Quality
The devcontainer includes pre-configured quality tools:
- **golangci-lint**: Code linting and style checking
- **gosec**: Security vulnerability scanning
- **gitleaks**: Secret detection
- **pre-commit**: Automated quality checks

### Database Development
- Use `goose` for database migrations
- Use `sqlc` for type-safe SQL code generation
- Test migrations with both up and down operations

## Migration from Host Development

### Backing Up Host Configuration
Before switching to devcontainer:
1. Export your VS Code settings
2. Note any custom tool configurations
3. Document any host-specific setup steps

### Gradual Migration
You can use both host and devcontainer development:
1. Start with devcontainer for new features
2. Gradually migrate existing workflows
3. Use `af validate` to compare environments

### Team Adoption
For team-wide adoption:
1. Start with optional devcontainer usage
2. Document team-specific customizations
3. Gradually make devcontainer the standard
4. Update CI/CD to match devcontainer environment

## Support and Resources

### Getting Help
- Check the [troubleshooting section](#troubleshooting) above
- Review VS Code devcontainer documentation
- Ask in team channels or create an issue

### Additional Resources
- [VS Code Dev Containers Documentation](https://code.visualstudio.com/docs/devcontainers/containers)
- [Docker Desktop Documentation](https://docs.docker.com/desktop/)
- [AgentFlow Development Guide](dev-environment.md)

## Validation Checklist

Use this checklist to ensure proper devcontainer adoption:

- [ ] Docker Desktop installed and running
- [ ] VS Code with Dev Containers extension installed
- [ ] Project opened in devcontainer successfully
- [ ] `af validate` shows `"container": "devcontainer"`
- [ ] No devcontainer warnings in `af validate` output
- [ ] All required tools show "ok" status
- [ ] Can build project: `make build` or `task build`
- [ ] Can run tests: `make test` or `task test`
- [ ] Database connection works (if applicable)
- [ ] NATS connection works (if applicable)
- [ ] Pre-commit hooks installed and working

## Conclusion

The devcontainer provides a robust, consistent development environment that eliminates "works on my machine" issues and accelerates developer onboarding. By following this guide, you'll have a fully functional AgentFlow development environment in minutes.

For questions or issues not covered in this guide, please refer to the project documentation or reach out to the development team.