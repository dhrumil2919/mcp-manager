# Getting Started with MCP Manager

This guide will help you get up and running with MCP Manager in just a few minutes.

## Prerequisites

- **Go 1.21+** (for building from source)
- **Docker** (for Docker mode) or **Kubernetes** (for Kubernetes mode)
- **Git** (for cloning the repository)

## Quick Installation

### Option 1: Using Makefile (Recommended)
```bash
# Clone the repository
git clone https://github.com/your-org/mcp-manager.git
cd mcp-manager

# Install dependencies and build
make install

# Verify installation
./mcp-manager --help
```

### Option 2: Install to System PATH
```bash
# Install to /usr/local/bin (requires sudo)
make install-system

# Now you can use it from anywhere
mcp-manager --help
```

### Option 3: Manual Build
```bash
# Clone and build manually
git clone https://github.com/your-org/mcp-manager.git
cd mcp-manager
go mod download
go build -o mcp-manager main.go
```

## Initial Setup

### 1. Create Configuration File
```bash
# Copy the example configuration
cp .mcp-manager.yaml.example ~/.mcp-manager.yaml

# Edit the configuration (optional - defaults work for most cases)
vim ~/.mcp-manager.yaml
```

### 2. Choose Your Deployment Mode

#### For Docker Mode:
- Ensure Docker is running: `docker info`
- The tool will connect to Docker via `/var/run/docker.sock`

#### For Kubernetes Mode:
- Ensure kubectl is configured: `kubectl cluster-info`
- The tool will use your current kubeconfig context

## Your First MCP Server

### Deploy a Test Server

#### Using Docker:
```bash
# Deploy a simple web server for testing
./mcp-manager deploy test-server \
  --image nginx:alpine \
  --mode docker \
  --port 80

# Check if it's running
./mcp-manager list --mode docker

# Check health
./mcp-manager health test-server --mode docker
```

#### Using Kubernetes:
```bash
# Deploy a simple web server for testing
./mcp-manager deploy test-server \
  --image nginx:alpine \
  --mode kubernetes \
  --port 80

# Check if it's running
./mcp-manager list --mode kubernetes

# Check health
./mcp-manager health test-server --mode kubernetes
```

### View Logs
```bash
# View recent logs
./mcp-manager logs test-server

# Follow logs in real-time
./mcp-manager logs test-server --follow
```

### Clean Up
```bash
# Delete the test server
./mcp-manager delete test-server
```

## Common Use Cases

### 1. Deploy with Custom Configuration
```bash
./mcp-manager deploy my-app \
  --image my-registry/my-app:v1.0.0 \
  --port 8080 \
  --replicas 3 \
  --cpu-limit 500m \
  --memory-limit 1Gi \
  --health-path /api/health
```

### 2. Monitor Multiple Servers
```bash
# List all servers with health status
./mcp-manager list --health

# Check health of all servers
./mcp-manager health
```

### 3. Environment-Specific Deployments
```bash
# Deploy to production namespace
./mcp-manager deploy prod-app \
  --image my-app:v2.0.0 \
  --namespace production \
  --replicas 5

# Deploy to staging with different resources
./mcp-manager deploy staging-app \
  --image my-app:v2.0.0-rc1 \
  --namespace staging \
  --replicas 2 \
  --cpu-limit 200m
```

## Configuration Tips

### Default Configuration
The tool works with sensible defaults, but you can customize everything:

```yaml
# ~/.mcp-manager.yaml
default_mode: kubernetes  # or "docker"

defaults:
  kubernetes:
    port: 8000
    replicas: 1
    cpu_request: 100m
    cpu_limit: 500m
    memory_request: 128Mi
    memory_limit: 512Mi
    health_path: /health
    
  docker:
    port: 8000
    network: mcp-network
    restart_policy: unless-stopped
    expose_ports: true
    health_path: /health
```

### Override Defaults
Command-line flags always override configuration file defaults:

```bash
# This overrides the default port from config
./mcp-manager deploy my-app --image my-app:latest --port 9000
```

## Troubleshooting

### Common Issues

1. **Docker connection failed**
   ```bash
   # Check if Docker is running
   docker info
   
   # Check Docker socket permissions
   ls -la /var/run/docker.sock
   ```

2. **Kubernetes connection failed**
   ```bash
   # Check kubectl configuration
   kubectl cluster-info
   
   # Check current context
   kubectl config current-context
   ```

3. **Permission denied**
   ```bash
   # For Docker: Add user to docker group
   sudo usermod -aG docker $USER
   # Then logout and login again
   
   # For system installation: Use sudo
   sudo make install-system
   ```

### Getting Help

```bash
# General help
./mcp-manager --help

# Command-specific help
./mcp-manager deploy --help
./mcp-manager list --help
./mcp-manager health --help

# Show version information
./mcp-manager version
```

## Next Steps

1. **Read the full README.md** for detailed command reference
2. **Customize your configuration** in `~/.mcp-manager.yaml`
3. **Set up CI/CD integration** using the CLI in your deployment pipelines
4. **Explore advanced features** like custom labels, annotations, and resource limits

## Need Help?

- Check the [README.md](README.md) for detailed documentation
- Look at the [example configuration](.mcp-manager.yaml.example)
- Run `./mcp-manager [command] --help` for command-specific help
- Check the [project issues](https://github.com/your-org/mcp-manager/issues) for known problems

Happy deploying! 🚀
