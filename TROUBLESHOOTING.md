# Troubleshooting Guide

This guide helps you resolve common issues with MCP Manager.

## Installation Issues

### Build Failures

**Problem**: `go build` fails with dependency errors
```
Solution:
1. Ensure Go 1.21+ is installed: `go version`
2. Clean module cache: `go clean -modcache`
3. Download dependencies: `go mod download`
4. Try building again: `make build`
```

**Problem**: Permission denied when installing to system PATH
```
Solution:
Use sudo for system installation:
sudo make install-system
```

### Docker Issues

**Problem**: "Cannot connect to the Docker daemon"
```bash
Error: failed to list containers: Cannot connect to the Docker daemon at unix:///var/run/docker.sock. Is the docker daemon running?
```
```
Solutions:
1. Start Docker daemon: `sudo systemctl start docker`
2. Check Docker status: `docker info`
3. Add user to docker group: `sudo usermod -aG docker $USER`
4. Logout and login again
5. Check socket permissions: `ls -la /var/run/docker.sock`
```

**Problem**: Docker containers not starting
```
Solutions:
1. Check Docker logs: `docker logs <container-name>`
2. Verify image exists: `docker images`
3. Check port conflicts: `docker ps -a`
4. Try with different port: `--port 8081`
```

### Kubernetes Issues

**Problem**: "Unable to connect to Kubernetes cluster"
```
Solutions:
1. Check kubectl config: `kubectl cluster-info`
2. Verify current context: `kubectl config current-context`
3. Test cluster access: `kubectl get nodes`
4. Check kubeconfig path: `--kubeconfig ~/.kube/config`
```

**Problem**: "Forbidden" or permission errors
```
Solutions:
1. Check RBAC permissions: `kubectl auth can-i create deployments`
2. Verify namespace access: `kubectl get namespaces`
3. Use correct service account
4. Check cluster admin permissions
```

## Deployment Issues

### Image Pull Errors

**Problem**: "ImagePullBackOff" or "ErrImagePull"
```
Solutions:
1. Verify image name and tag: `docker pull <image>`
2. Check image registry access
3. Use public images for testing: `nginx:alpine`
4. Configure image pull secrets if needed
```

### Port Conflicts

**Problem**: Port already in use
```
Solutions:
1. Check what's using the port: `lsof -i :8080`
2. Use different port: `--port 8081`
3. Stop conflicting service
4. Use port mapping: `--port-mapping "8081:8080"`
```

### Resource Issues

**Problem**: Pods stuck in "Pending" state
```
Solutions:
1. Check node resources: `kubectl describe nodes`
2. Reduce resource requests: `--cpu-request 50m --memory-request 64Mi`
3. Check resource quotas: `kubectl describe quota`
4. Scale down other deployments if needed
```

## Health Check Issues

### Health Check Failures

**Problem**: Health checks always fail
```
Solutions:
1. Verify health endpoint: `curl http://localhost:8080/health`
2. Check correct path: `--health-path /api/health`
3. Increase timeout: `--timeout 30s`
4. Check if service is actually running
```

**Problem**: "Connection refused" errors
```
Solutions:
1. Verify service is listening on correct port
2. Check firewall settings
3. Ensure service binds to 0.0.0.0, not 127.0.0.1
4. Check network policies (Kubernetes)
```

## Configuration Issues

### Config File Problems

**Problem**: Configuration not being loaded
```
Solutions:
1. Check file location: `~/.mcp-manager.yaml`
2. Verify YAML syntax: `yamllint ~/.mcp-manager.yaml`
3. Use explicit config: `--config /path/to/config.yaml`
4. Check file permissions: `ls -la ~/.mcp-manager.yaml`
```

**Problem**: Environment variables not working
```
Solutions:
1. Use proper format: `--env "KEY=value"`
2. Quote values with spaces: `--env "MESSAGE=hello world"`
3. Check container logs for environment variables
4. Verify environment in running container
```

## Networking Issues

### Service Discovery

**Problem**: Services can't communicate
```
Solutions (Kubernetes):
1. Use service names: `http://api-server:8080`
2. Check service endpoints: `kubectl get endpoints`
3. Verify network policies
4. Test with port-forward: `kubectl port-forward pod/api-server 8080:8080`

Solutions (Docker):
1. Use same network: `--network my-network`
2. Use container names: `http://api-server:8080`
3. Check network connectivity: `docker network ls`
4. Create custom network: `docker network create my-network`
```

### Ingress Issues

**Problem**: Ingress not working (Kubernetes)
```
Solutions:
1. Check ingress controller: `kubectl get pods -n ingress-nginx`
2. Verify ingress class: `--ingress-class nginx`
3. Check DNS resolution
4. Verify ingress resource: `kubectl describe ingress`
```

## Performance Issues

### Slow Deployments

**Problem**: Deployments take too long
```
Solutions:
1. Use smaller base images
2. Pre-pull images on nodes
3. Increase resource limits
4. Check network connectivity to registry
```

### High Resource Usage

**Problem**: Containers using too much CPU/memory
```
Solutions:
1. Set resource limits: `--cpu-limit 500m --memory-limit 512Mi`
2. Profile application performance
3. Check for memory leaks
4. Scale horizontally: `--replicas 3`
```

## Debugging Commands

### General Debugging
```bash
# Check MCP Manager version
mcp-manager version

# List all servers with health
mcp-manager list --health

# Get detailed logs
mcp-manager logs <server-name> --timestamps --tail 200
```

### Docker Debugging
```bash
# Check Docker system info
docker system info

# List all containers
docker ps -a

# Inspect container
docker inspect <container-name>

# Check container logs
docker logs <container-name>

# Execute into container
docker exec -it <container-name> /bin/sh
```

### Kubernetes Debugging
```bash
# Check cluster info
kubectl cluster-info

# Get all resources
kubectl get all

# Describe deployment
kubectl describe deployment <deployment-name>

# Check pod logs
kubectl logs <pod-name>

# Execute into pod
kubectl exec -it <pod-name> -- /bin/sh

# Check events
kubectl get events --sort-by=.metadata.creationTimestamp
```

## Common Error Messages

### "server not found"
```
Cause: Server name doesn't exist or wrong namespace
Solution: Check server name with `mcp-manager list`
```

### "context deadline exceeded"
```
Cause: Operation timeout
Solution: Increase timeout or check network connectivity
```

### "resource already exists"
```
Cause: Trying to create existing resource
Solution: Use different name or delete existing resource first
```

### "insufficient resources"
```
Cause: Not enough CPU/memory available
Solution: Reduce resource requests or add more nodes
```

## Getting Help

### Enable Debug Mode
```bash
# Add verbose logging (if implemented)
mcp-manager --verbose deploy my-app --image my-app:latest

# Check configuration
mcp-manager --config ~/.mcp-manager.yaml list
```

### Collect Information
When reporting issues, include:
1. MCP Manager version: `mcp-manager version`
2. Operating system and version
3. Docker/Kubernetes version
4. Complete error message
5. Configuration file (remove sensitive data)
6. Steps to reproduce

### Community Support
- Check [GitHub Issues](https://github.com/your-org/mcp-manager/issues)
- Read [README.md](README.md) and [GETTING_STARTED.md](GETTING_STARTED.md)
- Look at [EXAMPLES.md](EXAMPLES.md) for similar use cases

### Emergency Recovery
```bash
# Stop all MCP Manager containers (Docker)
docker ps | grep mcp-manager | awk '{print $1}' | xargs docker stop

# Delete all MCP Manager deployments (Kubernetes)
kubectl get deployments -l app.kubernetes.io/managed-by=mcp-manager -o name | xargs kubectl delete
```

Remember: When in doubt, start with the basics - check connectivity, permissions, and resource availability first!
