# MCP Manager Usage Examples

This document provides practical examples for common MCP Manager use cases.

## Basic Operations

### Deploy a Simple Web Server
```bash
# Deploy nginx for testing
mcp-manager deploy web-server \
  --image nginx:alpine \
  --port 80 \
  --mode docker

# Check if it's running
mcp-manager list --mode docker
```

### Deploy with Environment Variables
```bash
# Deploy with custom environment
mcp-manager deploy api-server \
  --image my-api:latest \
  --port 8080 \
  --env "DATABASE_URL=postgres://localhost/mydb" \
  --env "API_KEY=secret123" \
  --mode docker
```

### Deploy to Kubernetes with Resources
```bash
# Deploy with resource limits
mcp-manager deploy production-app \
  --image my-app:v1.2.3 \
  --port 8080 \
  --replicas 3 \
  --cpu-request 200m \
  --cpu-limit 1000m \
  --memory-request 256Mi \
  --memory-limit 1Gi \
  --namespace production
```

## Advanced Deployments

### Microservices Architecture
```bash
# Deploy API Gateway
mcp-manager deploy api-gateway \
  --image my-gateway:latest \
  --port 8080 \
  --replicas 2 \
  --namespace production

# Deploy User Service
mcp-manager deploy user-service \
  --image my-user-service:latest \
  --port 8081 \
  --replicas 3 \
  --namespace production

# Deploy Order Service
mcp-manager deploy order-service \
  --image my-order-service:latest \
  --port 8082 \
  --replicas 2 \
  --namespace production
```

### Development vs Production
```bash
# Development deployment (single replica, lower resources)
mcp-manager deploy my-app-dev \
  --image my-app:dev \
  --port 8080 \
  --replicas 1 \
  --cpu-limit 200m \
  --memory-limit 256Mi \
  --namespace development

# Production deployment (multiple replicas, higher resources)
mcp-manager deploy my-app-prod \
  --image my-app:v1.0.0 \
  --port 8080 \
  --replicas 5 \
  --cpu-request 500m \
  --cpu-limit 2000m \
  --memory-request 512Mi \
  --memory-limit 2Gi \
  --namespace production
```

## Monitoring and Debugging

### Health Monitoring
```bash
# Check health of all servers
mcp-manager health

# Check specific server with custom endpoint
mcp-manager health api-server --endpoint /api/health --timeout 30s

# Continuous health monitoring
mcp-manager health api-server --watch
```

### Log Analysis
```bash
# Get recent logs
mcp-manager logs api-server --tail 100

# Follow logs in real-time
mcp-manager logs api-server --follow

# Get logs from specific time
mcp-manager logs api-server --since 1h

# Get logs with timestamps
mcp-manager logs api-server --timestamps --since 30m
```

### Troubleshooting
```bash
# List all servers with health status
mcp-manager list --health

# Check specific namespace
mcp-manager list --namespace production --health

# Get detailed information about a deployment
kubectl describe deployment api-server  # For Kubernetes
docker inspect api-server              # For Docker
```

## Docker-Specific Examples

### Custom Networks
```bash
# Deploy with custom Docker network
mcp-manager deploy db-server \
  --image postgres:13 \
  --port 5432 \
  --network my-app-network \
  --env "POSTGRES_DB=myapp" \
  --env "POSTGRES_USER=user" \
  --env "POSTGRES_PASSWORD=password" \
  --mode docker

# Deploy app connected to same network
mcp-manager deploy app-server \
  --image my-app:latest \
  --port 8080 \
  --network my-app-network \
  --env "DATABASE_URL=postgres://user:password@db-server:5432/myapp" \
  --mode docker
```

### Port Mapping
```bash
# Deploy with custom port mapping
mcp-manager deploy web-app \
  --image my-web-app:latest \
  --port 3000 \
  --port-mapping "8080:3000" \
  --mode docker

# Access the app at http://localhost:8080
```

## Kubernetes-Specific Examples

### Ingress Configuration
```bash
# Deploy with ingress
mcp-manager deploy api-server \
  --image my-api:latest \
  --port 8080 \
  --ingress-host api.example.com \
  --ingress-class nginx \
  --namespace production
```

### Multiple Namespaces
```bash
# Deploy to different environments
mcp-manager deploy my-app --image my-app:v1.0.0 --namespace production
mcp-manager deploy my-app --image my-app:v1.1.0-rc1 --namespace staging
mcp-manager deploy my-app --image my-app:dev --namespace development

# List servers from specific namespace
mcp-manager list --namespace production
mcp-manager list --namespace staging
```

## CI/CD Integration

### Deployment Script
```bash
#!/bin/bash
# deploy.sh - Example deployment script

APP_NAME="my-application"
IMAGE_TAG=${1:-latest}
NAMESPACE=${2:-default}

echo "Deploying $APP_NAME:$IMAGE_TAG to $NAMESPACE..."

# Deploy the application
mcp-manager deploy $APP_NAME \
  --image "my-registry/$APP_NAME:$IMAGE_TAG" \
  --port 8080 \
  --replicas 3 \
  --cpu-request 200m \
  --cpu-limit 1000m \
  --memory-request 256Mi \
  --memory-limit 1Gi \
  --namespace $NAMESPACE \
  --health-path /health

# Wait for deployment to be ready
echo "Waiting for deployment to be ready..."
sleep 30

# Check health
mcp-manager health $APP_NAME --namespace $NAMESPACE

echo "Deployment completed!"
```

### Rolling Updates
```bash
# Update to new version
mcp-manager deploy my-app \
  --image my-app:v1.1.0 \
  --namespace production

# Check rollout status (Kubernetes)
kubectl rollout status deployment/my-app -n production
```

## Configuration Management

### Using Configuration File
```yaml
# ~/.mcp-manager.yaml
default_mode: kubernetes
kube_namespace: production

defaults:
  kubernetes:
    replicas: 3
    cpu_request: 200m
    cpu_limit: 1000m
    memory_request: 256Mi
    memory_limit: 1Gi
    ingress_class: nginx
```

```bash
# Deploy using defaults from config
mcp-manager deploy my-app --image my-app:latest
# This will use the defaults from the config file
```

### Environment-Specific Configs
```bash
# Use different config files for different environments
mcp-manager --config .mcp-manager-prod.yaml deploy my-app --image my-app:v1.0.0
mcp-manager --config .mcp-manager-staging.yaml deploy my-app --image my-app:v1.1.0-rc1
```

## Cleanup and Maintenance

### Bulk Operations
```bash
# Delete multiple servers
mcp-manager delete api-server user-service order-service

# Delete all servers in a namespace (be careful!)
mcp-manager list --namespace staging | grep -v "NAME" | awk '{print $1}' | xargs -I {} mcp-manager delete {} --namespace staging
```

### Maintenance Mode
```bash
# Scale down for maintenance
mcp-manager deploy my-app --image my-app:latest --replicas 0 --namespace production

# Scale back up
mcp-manager deploy my-app --image my-app:latest --replicas 3 --namespace production
```

## Best Practices

### Resource Management
```bash
# Always set resource limits in production
mcp-manager deploy prod-app \
  --image my-app:latest \
  --cpu-request 100m \
  --cpu-limit 500m \
  --memory-request 128Mi \
  --memory-limit 512Mi \
  --namespace production
```

### Health Checks
```bash
# Always configure health checks
mcp-manager deploy my-app \
  --image my-app:latest \
  --health-path /api/health \
  --port 8080
```

### Labeling and Organization
```bash
# Use consistent naming and labeling
mcp-manager deploy frontend-web \
  --image frontend:v1.0.0 \
  --label "tier=frontend" \
  --label "version=v1.0.0" \
  --namespace production
```

These examples should cover most common use cases. For more advanced scenarios, refer to the full documentation in [README.md](README.md).
