# Secrets Provider Documentation

## Overview

The AgentFlow secrets provider system provides a secure, flexible interface for managing secrets across different storage backends. It supports environment variables and file-based storage with hot reload capabilities, secure value masking, and comprehensive audit logging.

## Architecture

### Interface Design

The `SecretsProvider` interface defines five core operations:

```go
type SecretsProvider interface {
    GetSecret(ctx context.Context, key string) (string, error)
    SetSecret(ctx context.Context, key, value string) error
    DeleteSecret(ctx context.Context, key string) error
    ListSecrets(ctx context.Context) ([]string, error)
    Rotate(ctx context.Context, key string) error
}
```

### Providers

#### Environment Provider

The `EnvironmentProvider` manages secrets through environment variables with a configurable prefix.

**Features:**
- Configurable prefix (default: `AF_SECRET_`)
- Automatic key transformation (lowercase to uppercase)
- Read-only operations (set/delete affect current process only)
- No rotation support (environment variables are externally managed)

**Usage:**
```go
provider := NewEnvironmentProvider("AF_SECRET_")
value, err := provider.GetSecret(ctx, "api_key") // Reads AF_SECRET_API_KEY
```

#### File Provider

The `FileProvider` manages secrets in a JSON file with hot reload capabilities.

**Features:**
- JSON file storage with atomic writes
- Hot reload - detects external file changes automatically
- Secret rotation with cryptographically secure random values
- File permissions set to 0600 for security
- Concurrent access safety with read-write locks

**Usage:**
```go
provider := NewFileProvider("secrets.json")
err := provider.SetSecret(ctx, "api_key", "secret-value")
err = provider.Rotate(ctx, "api_key") // Generates new random value
```

## Security Features

### Value Masking

All secret values are automatically masked in logs and debug output:

```go
MaskSecret("short") // Returns "****"
MaskSecret("sk-1234567890abcdef") // Returns "sk**************ef"
```

**Masking Rules:**
- Empty strings: return empty
- 1-4 characters: all asterisks
- 5+ characters: show first 2 and last 2, mask middle

### Key Validation

Secret keys must follow strict validation rules:

```go
ValidateKey("valid_key-123") // ✓ Valid
ValidateKey("invalid@key")   // ✗ Invalid characters
ValidateKey("")              // ✗ Empty key
```

**Validation Rules:**
- Alphanumeric characters, underscores, and hyphens only
- Maximum length: 255 characters
- Cannot be empty

### Access Logging

All secret operations are logged with correlation context:

```json
{
  "level": "info",
  "component": "secrets.file",
  "message": "Secret retrieved from file",
  "key": "api_key",
  "path": "/path/to/secrets.json",
  "value_length": 32,
  "trace_id": "abc123",
  "timestamp": "2025-01-15T10:30:00Z"
}
```

## Hot Reload Mechanism

The file provider implements hot reload by checking file modification times:

1. **Modification Detection**: Compare file `ModTime` with cached timestamp
2. **Atomic Loading**: Read and parse entire file if modified
3. **Thread Safety**: Use read-write locks for concurrent access
4. **Error Handling**: Continue with cached data if reload fails

### Hot Reload Example

```bash
# Terminal 1: Start application with file provider
go run main.go

# Terminal 2: Modify secrets file externally
echo '{"new_secret": "external_value"}' > secrets.json

# Terminal 1: Next GetSecret() call automatically loads new values
```

## Usage Patterns

### Basic Operations

```go
// Create provider
provider := NewFileProvider("secrets.json")
ctx := context.Background()

// Store secret
err := provider.SetSecret(ctx, "database_url", "postgres://...")
if err != nil {
    log.Fatal("Failed to store secret:", err)
}

// Retrieve secret
dbURL, err := provider.GetSecret(ctx, "database_url")
if err != nil {
    log.Fatal("Failed to get secret:", err)
}

// List all secrets
keys, err := provider.ListSecrets(ctx)
fmt.Printf("Available secrets: %v\n", keys)

// Rotate secret (generates new random value)
err = provider.Rotate(ctx, "api_key")
if err != nil {
    log.Fatal("Failed to rotate secret:", err)
}

// Delete secret
err = provider.DeleteSecret(ctx, "old_secret")
if err != nil {
    log.Fatal("Failed to delete secret:", err)
}
```

### Environment Configuration

```go
// Use environment variables with custom prefix
provider := NewEnvironmentProvider("MYAPP_")

// This will read MYAPP_DATABASE_URL
dbURL, err := provider.GetSecret(ctx, "database_url")
```

### Error Handling

```go
value, err := provider.GetSecret(ctx, "api_key")
switch {
case errors.Is(err, ErrSecretNotFound):
    // Handle missing secret
    log.Warn("Secret not configured, using default")
    value = "default-value"
case errors.Is(err, ErrPermissionDenied):
    // Handle permission error
    log.Error("Access denied to secret")
    return err
case errors.Is(err, ErrProviderUnavailable):
    // Handle provider failure
    log.Error("Secrets provider unavailable")
    return err
default:
    // Handle other errors
    if err != nil {
        return fmt.Errorf("failed to get secret: %w", err)
    }
}
```

## Integration Examples

### Dependency Injection

```go
type Config struct {
    SecretsProvider secrets.SecretsProvider
}

func NewConfig() *Config {
    var provider secrets.SecretsProvider
    
    if os.Getenv("SECRETS_FILE") != "" {
        provider = secrets.NewFileProvider(os.Getenv("SECRETS_FILE"))
    } else {
        provider = secrets.NewEnvironmentProvider("AF_SECRET_")
    }
    
    return &Config{
        SecretsProvider: provider,
    }
}
```

### Database Connection

```go
func ConnectDatabase(provider secrets.SecretsProvider) (*sql.DB, error) {
    ctx := context.Background()
    
    dbURL, err := provider.GetSecret(ctx, "database_url")
    if err != nil {
        return nil, fmt.Errorf("failed to get database URL: %w", err)
    }
    
    return sql.Open("postgres", dbURL)
}
```

### API Key Management

```go
func GetAPIClient(provider secrets.SecretsProvider) (*APIClient, error) {
    ctx := context.Background()
    
    apiKey, err := provider.GetSecret(ctx, "external_api_key")
    if err != nil {
        return nil, fmt.Errorf("failed to get API key: %w", err)
    }
    
    return NewAPIClient(apiKey), nil
}
```

## Future Provider Expansion

The secrets provider interface is designed for extensibility. Future providers could include:

### HashiCorp Vault Provider

```go
type VaultProvider struct {
    client *vault.Client
    path   string
}

func (p *VaultProvider) GetSecret(ctx context.Context, key string) (string, error) {
    secret, err := p.client.Logical().Read(p.path + "/" + key)
    // Implementation details...
}
```

### AWS Secrets Manager Provider

```go
type AWSSecretsProvider struct {
    client *secretsmanager.SecretsManager
    region string
}

func (p *AWSSecretsProvider) GetSecret(ctx context.Context, key string) (string, error) {
    result, err := p.client.GetSecretValue(&secretsmanager.GetSecretValueInput{
        SecretId: aws.String(key),
    })
    // Implementation details...
}
```

### Kubernetes Secrets Provider

```go
type K8sSecretsProvider struct {
    client    kubernetes.Interface
    namespace string
}

func (p *K8sSecretsProvider) GetSecret(ctx context.Context, key string) (string, error) {
    secret, err := p.client.CoreV1().Secrets(p.namespace).Get(ctx, key, metav1.GetOptions{})
    // Implementation details...
}
```

## Best Practices

### Security

1. **File Permissions**: Ensure secrets files have restrictive permissions (0600)
2. **Key Validation**: Always validate secret keys before operations
3. **Value Masking**: Never log raw secret values
4. **Rotation**: Implement regular secret rotation schedules
5. **Access Logging**: Log all secret access for audit trails

### Performance

1. **Caching**: File provider caches secrets and only reloads on modification
2. **Concurrent Access**: Use appropriate locking for thread safety
3. **Error Handling**: Implement graceful degradation on provider failures
4. **Resource Cleanup**: Properly close resources and clean up temporary files

### Operational

1. **Monitoring**: Monitor secret access patterns and failures
2. **Backup**: Backup secrets files with appropriate encryption
3. **Testing**: Test hot reload and rotation scenarios regularly
4. **Documentation**: Document secret key naming conventions and usage

## Troubleshooting

### Common Issues

**Secret Not Found**
```
Error: secret not found: api_key
```
- Check secret key spelling and case
- Verify secret exists in provider
- Check provider configuration (file path, environment prefix)

**Permission Denied**
```
Error: permission denied
```
- Check file permissions (should be 0600)
- Verify process has read/write access to secrets file
- Check environment variable access

**Hot Reload Not Working**
```
Warning: Failed to reload secrets
```
- Verify file modification time changes
- Check file format (valid JSON for file provider)
- Ensure no file locks preventing access

**Rotation Failures**
```
Error: failed to generate random value
```
- Check system entropy availability
- Verify write permissions to secrets file
- Ensure sufficient disk space

### Debug Mode

Enable debug logging to troubleshoot issues:

```go
logger := logging.NewLogger().WithLevel("debug")
provider := NewFileProvider("secrets.json")
provider.logger = logger
```

This will provide detailed logs of all secret operations, file access, and error conditions.