# Local Services Setup Guide

This guide explains how to set up and manage local development services for AgentFlow, including PostgreSQL, Redis, Qdrant vector database, and NATS.

## Overview

AgentFlow requires several services for development and testing:

- **PostgreSQL**: Primary database for storing tenants, users, agents, workflows, plans, messages, tools, audits, budgets, and RBAC data
- **Redis**: Caching layer for short-term memory and response caching
- **Qdrant**: Vector database for embeddings and similarity search
- **NATS**: Message bus for inter-service communication

## Quick Start

### Using Docker Compose (Recommended)

1. **Start all services:**
   ```bash
   docker-compose up -d
   ```

2. **Verify services are running:**
   ```bash
   af validate
   ```

3. **Stop services:**
   ```bash
   docker-compose down
   ```

### Individual Service Management

Start specific services:
```bash
# Start only PostgreSQL
docker-compose up -d postgres

# Start only Redis
docker-compose up -d redis

# Start only Qdrant
docker-compose up -d qdrant

# Start only NATS
docker-compose up -d nats
```

## Service Configuration

### PostgreSQL
- **Port**: 5432
- **Database**: agentflow_dev
- **Username**: agentflow
- **Password**: agentflow_dev_password
- **Connection String**: `postgresql://agentflow:agentflow_dev_password@localhost:5432/agentflow_dev`

### Redis
- **Port**: 6379
- **Connection String**: `redis://localhost:6379`
- **No authentication required in development**

### Qdrant Vector Database
- **HTTP API Port**: 6333
- **gRPC API Port**: 6334
- **Health Check**: `http://localhost:6333/health`
- **Web UI**: `http://localhost:6333/dashboard`

### NATS
- **Client Port**: 4222
- **Monitoring Port**: 8222
- **Connection String**: `nats://localhost:4222`
- **Monitoring UI**: `http://localhost:8222`

## Health Checks

The `af validate` command performs health checks on all services:

```bash
af validate
```

Example output:
```json
{
  "version": "1.0.0",
  "timestamp": "2025-01-25T10:00:00Z",
  "environment": {
    "platform": "windows",
    "architecture": "amd64",
    "container": "host"
  },
  "services": {
    "postgres": {
      "status": "available",
      "connection": "postgresql://agentflow@localhost:5432/agentflow_dev"
    },
    "redis": {
      "status": "available",
      "connection": "redis://localhost:6379"
    },
    "qdrant": {
      "status": "available",
      "connection": "http://localhost:6333"
    },
    "nats": {
      "status": "available",
      "connection": "nats://localhost:4222"
    }
  }
}
```

## Platform-Specific Instructions

### Windows

On Windows, services are typically run via Docker Desktop:

1. **Install Docker Desktop** from https://docker.com/products/docker-desktop
2. **Start Docker Desktop** and ensure it's running
3. **Run services** using docker-compose as shown above

If services are unavailable, `af validate` will provide helpful guidance:
- "Redis service unavailable on Windows. Start with: docker-compose up redis"
- "Qdrant service unavailable on Windows. Start with: docker-compose up qdrant"

### Linux/macOS

Services can be run via Docker or installed natively:

#### Docker (Recommended)
```bash
docker-compose up -d
```

#### Native Installation
- **PostgreSQL**: Install via package manager (apt, brew, etc.)
- **Redis**: Install via package manager
- **Qdrant**: Download from https://qdrant.tech/
- **NATS**: Download from https://nats.io/

### VS Code Devcontainer

The devcontainer automatically includes all required services:

1. **Open project in VS Code**
2. **Select "Reopen in Container"** when prompted
3. **Services start automatically** with the devcontainer

## Troubleshooting

### Common Issues

#### Services Not Starting
```bash
# Check Docker is running
docker --version
docker ps

# Check for port conflicts
netstat -an | grep :5432  # PostgreSQL
netstat -an | grep :6379  # Redis
netstat -an | grep :6333  # Qdrant
netstat -an | grep :4222  # NATS
```

#### Connection Failures
```bash
# Test individual connections
psql -h localhost -p 5432 -U agentflow -d agentflow_dev
redis-cli -h localhost -p 6379 ping
curl http://localhost:6333/health
nats server check --server=localhost:4222
```

#### Permission Issues (Linux/macOS)
```bash
# Fix Docker permissions
sudo usermod -aG docker $USER
# Log out and back in

# Fix file permissions
sudo chown -R $USER:$USER .
```

### Service-Specific Troubleshooting

#### PostgreSQL
- **Issue**: Connection refused
- **Solution**: Ensure PostgreSQL container is running and port 5432 is available
- **Check**: `docker logs agentflow-postgres-1`

#### Redis
- **Issue**: Connection timeout
- **Solution**: Verify Redis container is healthy
- **Check**: `docker logs agentflow-redis-1`

#### Qdrant
- **Issue**: Health check fails
- **Solution**: Wait for Qdrant to fully initialize (can take 30-60 seconds)
- **Check**: `docker logs agentflow-qdrant-1`

#### NATS
- **Issue**: JetStream not available
- **Solution**: Ensure NATS is started with `--jetstream` flag (included in docker-compose)
- **Check**: `docker logs agentflow-nats-1`

### Performance Optimization

#### Resource Allocation
```yaml
# docker-compose.yml - Add resource limits
services:
  postgres:
    deploy:
      resources:
        limits:
          memory: 512M
        reservations:
          memory: 256M
```

#### Data Persistence
All services use Docker volumes for data persistence:
- `postgres_data`: PostgreSQL data
- `redis_data`: Redis data
- `qdrant_data`: Qdrant collections and indexes
- `nats_data`: NATS JetStream data

### Cleanup

#### Remove All Data
```bash
# Stop services and remove volumes
docker-compose down -v

# Remove all AgentFlow containers and volumes
docker system prune -f
docker volume prune -f
```

#### Reset Individual Services
```bash
# Reset PostgreSQL
docker-compose stop postgres
docker volume rm agentflow_postgres_data
docker-compose up -d postgres

# Reset Redis
docker-compose stop redis
docker volume rm agentflow_redis_data
docker-compose up -d redis
```

## Development Workflow

### Typical Development Session
1. **Start services**: `docker-compose up -d`
2. **Validate environment**: `af validate`
3. **Run migrations**: `goose -dir migrations postgres "postgresql://agentflow:agentflow_dev_password@localhost:5432/agentflow_dev" up`
4. **Develop and test**
5. **Stop services**: `docker-compose down` (optional)

### Testing with Services
```bash
# Run tests that require services
go test ./internal/storage/... -v

# Run integration tests
go test ./test/integration/... -v

# Skip service-dependent tests
go test -short ./...
```

## Monitoring and Observability

### Service Monitoring
- **PostgreSQL**: Use `psql` or pgAdmin
- **Redis**: Use `redis-cli` or RedisInsight
- **Qdrant**: Web UI at http://localhost:6333/dashboard
- **NATS**: Monitoring at http://localhost:8222

### Logs
```bash
# View all service logs
docker-compose logs -f

# View specific service logs
docker-compose logs -f postgres
docker-compose logs -f redis
docker-compose logs -f qdrant
docker-compose logs -f nats
```

### Health Monitoring
```bash
# Continuous health monitoring
watch -n 5 'af validate | jq .services'

# Service-specific health checks
curl -s http://localhost:6333/health | jq
curl -s http://localhost:8222/healthz
```

## Security Considerations

### Development Security
- **Default passwords**: Change default passwords for production
- **Network isolation**: Services are only accessible on localhost
- **Data encryption**: Enable TLS for production deployments

### Production Deployment
- Use environment variables for secrets
- Enable authentication on all services
- Configure network security groups
- Enable audit logging
- Use encrypted storage volumes

## Next Steps

After setting up local services:
1. **Run database migrations**: See [Migration Guide](./migration-guide.md)
2. **Configure authentication**: See [Security Baseline](./security-baseline.md)
3. **Set up monitoring**: See [Observability Guide](./observability-guide.md)
4. **Deploy to production**: See [Deployment Guide](./deployment-guide.md)