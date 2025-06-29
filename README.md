# MCP Manager CLI

[![CI](https://github.com/dhrumil2919/mcp-manager/workflows/CI/badge.svg)](https://github.com/dhrumil2919/mcp-manager/actions/workflows/ci.yml)
[![Release](https://github.com/dhrumil2919/mcp-manager/workflows/Release/badge.svg)](https://github.com/dhrumil2919/mcp-manager/actions/workflows/release.yml)
[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)

A powerful command-line tool for managing Model Context Protocol (MCP) servers across Kubernetes and Docker environments. Built in Go for high performance and simplicity.

> **🚀 New to MCP Manager?** Check out our [Getting Started Guide](GETTING_STARTED.md) for a quick 5-minute setup!

## Features

### 🚀 **Dual Deployment Support**
- **Kubernetes**: Deploy MCP servers as Deployments with Services and Ingress
- **Docker**: Deploy MCP servers as containers with networking
- **Unified CLI**: Single binary for managing both deployment modes

### ⚙️ **Smart Configuration**
- **Configuration File**: YAML-based configuration with sensible defaults
- **Override Flags**: Command-line flags override configuration defaults
- **Environment Support**: Different configs for different environments

### 📊 **Real-time Operations**
- **No Database**: All data fetched real-time from Kubernetes/Docker APIs
- **Live Status**: Real-time server status and health information
- **Endpoint Discovery**: Shows actual endpoints from ingress/port mappings

### 🔧 **Developer Friendly**
- **Single Binary**: Easy installation and distribution
- **Colored Output**: Beautiful, readable command output
- **Watch Mode**: Real-time monitoring with auto-refresh

## Installation

### Quick Install with Makefile
```bash
# Install dependencies and build
make install

# Build only
make build

# Install to system PATH
make install-system
```

### Manual Installation
```bash
# Clone repository
git clone https://github.com/your-org/mcp-manager.git
cd mcp-manager

# Install dependencies
go mod download

# Build binary
go build -o mcp-manager main.go
```

## Quick Start

> **📖 Detailed Guide**: See [GETTING_STARTED.md](GETTING_STARTED.md) for step-by-step instructions

### 1. Install and Build
```bash
# Clone and install
git clone https://github.com/your-org/mcp-manager.git
cd mcp-manager
make install

# Verify installation
./mcp-manager --help
```

### 2. Deploy Your First MCP Server
```bash
# Deploy to Docker (easiest to get started)
./mcp-manager deploy test-server --image nginx:alpine --mode docker --port 80

# Deploy to Kubernetes
./mcp-manager deploy test-server --image nginx:alpine --mode kubernetes --port 80
```

### 3. Manage and Monitor
```bash
# List all servers
./mcp-manager list

# Check health
./mcp-manager health test-server

# View logs
./mcp-manager logs test-server --follow

# Clean up
./mcp-manager delete test-server
```

## Quick Reference

| Command | Description | Example |
|---------|-------------|---------|
| `deploy` | Deploy MCP server | `mcp-manager deploy my-app --image my-app:latest` |
| `list` | List all servers | `mcp-manager list --health` |
| `health` | Check server health | `mcp-manager health my-app` |
| `logs` | View server logs | `mcp-manager logs my-app --follow` |
| `delete` | Delete server | `mcp-manager delete my-app` |

### Common Flags
- `--mode docker|kubernetes` - Choose deployment mode
- `--namespace <name>` - Kubernetes namespace
- `--port <port>` - Container port
- `--replicas <count>` - Number of replicas (Kubernetes)
- `--health` - Include health information
- `--follow` - Follow logs in real-time

> **💡 More Examples**: Check out [EXAMPLES.md](EXAMPLES.md) for detailed usage examples and best practices

## Commands

### `mcp-manager deploy`
Deploy an MCP server with smart defaults from configuration.

```bash
# Basic deployment
mcp-manager deploy my-server --image my-app:latest

# With custom settings
mcp-manager deploy my-server \
  --image my-app:latest \
  --port 9000 \
  --replicas 3 \
  --cpu-limit 1000m \
  --memory-limit 1Gi \
  --health-path /api/health

# Deploy to specific mode
mcp-manager deploy my-server --image my-app:latest --mode docker

# Dry run (show what would be deployed)
mcp-manager deploy my-server --image my-app:latest --dry-run
```

### `mcp-manager list`
List all MCP servers with their endpoints and status.

```bash
# List all servers
mcp-manager list

# Include health check information
mcp-manager list --health

# List from specific namespace (Kubernetes)
mcp-manager list --namespace production
```

### `mcp-manager health`
Check health status of MCP servers.

```bash
# Check specific server
mcp-manager health my-server

# Check all servers
mcp-manager health

# Watch health status (refresh every 5 seconds)
mcp-manager health my-server --watch

# Custom health endpoint
mcp-manager health my-server --endpoint /api/health

# Custom timeout
mcp-manager health my-server --timeout 30s
```

### `mcp-manager logs`
Fetch and display logs from MCP servers.

```bash
# Get recent logs
mcp-manager logs my-server

# Follow logs (tail -f style)
mcp-manager logs my-server --follow

# Get last 100 lines
mcp-manager logs my-server --tail 100

# Get logs since 1 hour ago
mcp-manager logs my-server --since 1h

# Get logs with timestamps
mcp-manager logs my-server --timestamps
```

### `mcp-manager delete`
Delete MCP servers and their resources.

```bash
# Delete with confirmation prompt
mcp-manager delete my-server

# Delete without confirmation
mcp-manager delete my-server --force

# Delete from specific namespace
mcp-manager delete my-server --namespace production

# Keep persistent volumes (Kubernetes)
mcp-manager delete my-server --keep-volumes
```

## Configuration

### Configuration File
Create `~/.mcp-manager.yaml` or `./.mcp-manager.yaml`:

```yaml
# Default deployment mode
default_mode: kubernetes

# Kubernetes configuration
kube_config: ~/.kube/config
kube_namespace: default

# Docker configuration  
docker_host: unix:///var/run/docker.sock

# Default configurations
defaults:
  kubernetes:
    image: ""                    # Default image (override with --image)
    port: 8000                   # Default container port
    health_path: /health         # Default health check path
    replicas: 1                  # Default replica count
    cpu_request: 100m            # Default CPU request
    cpu_limit: 500m              # Default CPU limit
    memory_request: 128Mi        # Default memory request
    memory_limit: 512Mi          # Default memory limit
    ingress_class: nginx         # Default ingress class
    
    # Default labels (added to standard Kubernetes labels)
    labels:
      environment: production
      team: platform

  docker:
    image: ""                    # Default image (override with --image)
    port: 8000                   # Default container port
    health_path: /health         # Default health check path
    network: mcp-network         # Default Docker network
    restart_policy: unless-stopped
    expose_ports: true           # Auto-expose ports to host
    
    # Default labels (added to standard Docker labels)
    labels:
      environment: production
      team: platform
```

## Development

### Make Commands
```bash
make help          # Show all available commands
make deps          # Install dependencies
make build         # Build binary
make test          # Run tests
make clean         # Clean build artifacts
make install       # Install dependencies and build
make install-system # Install to system PATH
make cross-compile # Build for multiple platforms
```

### Project Structure
```
├── main.go                 # Entry point
├── cmd/                    # CLI commands
│   ├── root.go            # Root command and config
│   ├── deploy.go          # Deploy command
│   ├── list.go            # List command
│   ├── health.go          # Health command
│   ├── logs.go            # Logs command
│   └── delete.go          # Delete command
├── internal/
│   ├── kubernetes/        # Kubernetes client
│   │   └── client.go
│   └── docker/            # Docker client
│       └── client.go
├── .mcp-manager.yaml.example
├── Makefile
└── README.md
```

## Documentation

| Document | Description |
|----------|-------------|
| **📖 [Getting Started Guide](GETTING_STARTED.md)** | Quick 5-minute setup and first deployment |
| **💡 [Usage Examples](EXAMPLES.md)** | Practical examples and best practices |
| **🔧 [Troubleshooting Guide](TROUBLESHOOTING.md)** | Common issues and solutions |
| **⚙️ [Configuration Reference](.mcp-manager.yaml.example)** | Complete configuration options |
| **📋 [Makefile](Makefile)** | Build and installation commands |

### Quick Help
```bash
# General help
mcp-manager --help

# Command-specific help
mcp-manager deploy --help
mcp-manager list --help
mcp-manager health --help
```

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.
