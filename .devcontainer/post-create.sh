#!/bin/bash
set -e

echo "🚀 Setting up AgentFlow development environment..."

# Update package lists
sudo apt-get update

# Install PostgreSQL client (pinned version)
echo "📦 Installing PostgreSQL client..."
sudo apt-get install -y postgresql-client-13

# Install NATS CLI (pinned version)
echo "📦 Installing NATS CLI..."
NATS_VERSION="0.1.4"
curl -L https://github.com/nats-io/natscli/releases/download/v${NATS_VERSION}/nats-${NATS_VERSION}-linux-amd64.tar.gz | sudo tar -xz -C /usr/local/bin --strip-components=1 nats-${NATS_VERSION}-linux-amd64/nats
sudo chmod +x /usr/local/bin/nats

# Install additional development tools
echo "📦 Installing development tools..."
sudo apt-get install -y \
    make \
    curl \
    wget \
    jq \
    git \
    ca-certificates

# Install Task (Taskfile runner)
echo "📦 Installing Task runner..."
TASK_VERSION="3.35.1"
curl -sL https://github.com/go-task/task/releases/download/v${TASK_VERSION}/task_linux_amd64.tar.gz | sudo tar -xz -C /usr/local/bin task

# Install golangci-lint (pinned version)
echo "📦 Installing golangci-lint..."
GOLANGCI_VERSION="1.55.2"
curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sudo sh -s -- -b /usr/local/bin v${GOLANGCI_VERSION}

# Install goose (database migration tool)
echo "📦 Installing goose..."
GOOSE_VERSION="3.18.0"
go install github.com/pressly/goose/v3/cmd/goose@v${GOOSE_VERSION}

# Install sqlc (SQL code generator)
echo "📦 Installing sqlc..."
SQLC_VERSION="1.25.0"
go install github.com/sqlc-dev/sqlc/cmd/sqlc@v${SQLC_VERSION}

# Install security tools
echo "📦 Installing security tools..."
# gosec
GOSEC_VERSION="2.19.0"
go install github.com/securecodewarrior/gosec/v2/cmd/gosec@v${GOSEC_VERSION}

# gitleaks
GITLEAKS_VERSION="8.18.1"
curl -sL https://github.com/gitleaks/gitleaks/releases/download/v${GITLEAKS_VERSION}/gitleaks_${GITLEAKS_VERSION}_linux_x64.tar.gz | sudo tar -xz -C /usr/local/bin gitleaks

# Install pre-commit
echo "📦 Installing pre-commit..."
pip3 install pre-commit==3.6.0

# Setup pre-commit hooks
echo "🔧 Setting up pre-commit hooks..."
if [ -f ".pre-commit-config.yaml" ]; then
    pre-commit install
    echo "✅ Pre-commit hooks installed"
else
    echo "⚠️  .pre-commit-config.yaml not found, skipping pre-commit setup"
fi

# Verify Go installation and modules
echo "🔧 Setting up Go environment..."
go version
go env GOPATH
go env GOROOT

# Download Go dependencies if go.mod exists
if [ -f "go.mod" ]; then
    echo "📦 Downloading Go dependencies..."
    go mod download
    go mod tidy
fi

# Create validation script
echo "📝 Creating environment validation script..."
sudo tee /usr/local/bin/validate-env > /dev/null << 'EOF'
#!/bin/bash
# Environment validation script for AgentFlow development

echo "🔍 Validating AgentFlow development environment..."

# Check Go version
echo -n "Go: "
if command -v go &> /dev/null; then
    go version | awk '{print $3}'
else
    echo "❌ Not installed"
    exit 1
fi

# Check Task
echo -n "Task: "
if command -v task &> /dev/null; then
    task --version
else
    echo "❌ Not installed"
    exit 1
fi

# Check PostgreSQL client
echo -n "PostgreSQL client: "
if command -v psql &> /dev/null; then
    psql --version | awk '{print $3}'
else
    echo "❌ Not installed"
    exit 1
fi

# Check NATS CLI
echo -n "NATS CLI: "
if command -v nats &> /dev/null; then
    nats --version 2>/dev/null || echo "installed"
else
    echo "❌ Not installed"
    exit 1
fi

# Check golangci-lint
echo -n "golangci-lint: "
if command -v golangci-lint &> /dev/null; then
    golangci-lint version | head -1 | awk '{print $4}'
else
    echo "❌ Not installed"
    exit 1
fi

# Check goose
echo -n "goose: "
if command -v goose &> /dev/null; then
    goose -version 2>/dev/null | awk '{print $3}' || echo "installed"
else
    echo "❌ Not installed"
    exit 1
fi

# Check sqlc
echo -n "sqlc: "
if command -v sqlc &> /dev/null; then
    sqlc version
else
    echo "❌ Not installed"
    exit 1
fi

# Check gosec
echo -n "gosec: "
if command -v gosec &> /dev/null; then
    gosec -version 2>/dev/null | awk '{print $2}' || echo "installed"
else
    echo "❌ Not installed"
    exit 1
fi

# Check gitleaks
echo -n "gitleaks: "
if command -v gitleaks &> /dev/null; then
    gitleaks version | awk '{print $2}'
else
    echo "❌ Not installed"
    exit 1
fi

# Check pre-commit
echo -n "pre-commit: "
if command -v pre-commit &> /dev/null; then
    pre-commit --version | awk '{print $2}'
else
    echo "❌ Not installed"
    exit 1
fi

echo "✅ All tools validated successfully!"
EOF

sudo chmod +x /usr/local/bin/validate-env

echo "✅ AgentFlow development environment setup complete!"
echo "🔧 Run 'validate-env' to verify all tools are properly installed"
echo "🚀 You can now start developing AgentFlow!"