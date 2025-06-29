package docker

import (
	"context"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/network"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

// Client wraps the Docker client
type Client struct {
	client *client.Client
	host   string
}

// NewClient creates a new Docker client
func NewClient(host string) (*Client, error) {
	var cli *client.Client
	var err error

	if host == "" {
		// Use default Docker host
		cli, err = client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	} else {
		// Use specified host
		cli, err = client.NewClientWithOpts(
			client.WithHost(host),
			client.WithAPIVersionNegotiation(),
		)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	return &Client{
		client: cli,
		host:   host,
	}, nil
}

// MCPServerInfo represents information about an MCP server
type MCPServerInfo struct {
	Name     string
	Image    string
	Status   string
	Endpoint string
	Age      string
}

// ListMCPServers lists all MCP servers (Docker containers)
func (c *Client) ListMCPServers() ([]*MCPServerInfo, error) {
	containers, err := c.client.ContainerList(context.Background(), types.ContainerListOptions{
		All: true, // Include stopped containers
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	var servers []*MCPServerInfo
	for _, container := range containers {
		// Filter for MCP servers by checking labels
		if isMCPServer(container.Labels) {
			server := &MCPServerInfo{
				Name:     getContainerName(container.Names),
				Image:    container.Image,
				Status:   container.Status,
				Endpoint: getContainerEndpoint(container),
				Age:      formatAge(container.Created),
			}
			servers = append(servers, server)
		}
	}

	return servers, nil
}

// Helper functions for ListMCPServers
func isMCPServer(labels map[string]string) bool {
	// Check if container has MCP manager labels
	if managedBy, exists := labels["mcp-manager.managed-by"]; exists && managedBy == "mcp-manager" {
		return true
	}
	if managedBy, exists := labels["app.kubernetes.io/managed-by"]; exists && managedBy == "mcp-manager" {
		return true
	}
	return false
}

func getContainerName(names []string) string {
	if len(names) == 0 {
		return "unknown"
	}
	// Remove leading slash from container name
	name := names[0]
	if strings.HasPrefix(name, "/") {
		return name[1:]
	}
	return name
}

func getContainerEndpoint(container types.Container) string {
	// Try to get endpoint from port mappings
	for _, port := range container.Ports {
		if port.PublicPort > 0 {
			return fmt.Sprintf("localhost:%d", port.PublicPort)
		}
	}
	return "not exposed"
}

func formatAge(created int64) string {
	createdTime := time.Unix(created, 0)
	duration := time.Since(createdTime)

	if duration < time.Minute {
		return fmt.Sprintf("%ds", int(duration.Seconds()))
	} else if duration < time.Hour {
		return fmt.Sprintf("%dm", int(duration.Minutes()))
	} else if duration < 24*time.Hour {
		return fmt.Sprintf("%dh", int(duration.Hours()))
	} else {
		return fmt.Sprintf("%dd", int(duration.Hours()/24))
	}
}

// DeploymentConfig represents deployment configuration
type DeploymentConfig struct {
	Name          string
	Image         string
	Port          int
	HealthPath    string
	Command       []string
	Args          []string
	Environment   map[string]string
	Network       string
	RestartPolicy string
	ExposePorts   bool
	PortMappings  []string
	Labels        map[string]string
}

// DeployMCPServer deploys an MCP server to Docker
func (c *Client) DeployMCPServer(config *DeploymentConfig) error {
	ctx := context.Background()

	// Check if container already exists
	exists, err := c.ContainerExists(config.Name)
	if err != nil {
		return fmt.Errorf("failed to check if container exists: %w", err)
	}

	if exists {
		return fmt.Errorf("container '%s' already exists", config.Name)
	}

	// Create network if it doesn't exist
	if config.Network != "" {
		if err := c.ensureNetwork(config.Network); err != nil {
			return fmt.Errorf("failed to ensure network: %w", err)
		}
	}

	// Prepare container configuration
	containerConfig := &container.Config{
		Image:  config.Image,
		Labels: c.buildLabels(config),
		Env:    c.buildEnvVars(config.Environment),
	}

	// Add command and args if specified
	if len(config.Command) > 0 {
		containerConfig.Cmd = config.Command
	}
	if len(config.Args) > 0 {
		if len(containerConfig.Cmd) == 0 {
			containerConfig.Cmd = config.Args
		} else {
			containerConfig.Cmd = append(containerConfig.Cmd, config.Args...)
		}
	}

	// Expose port if specified
	if config.Port > 0 {
		containerConfig.ExposedPorts = nat.PortSet{
			nat.Port(fmt.Sprintf("%d/tcp", config.Port)): struct{}{},
		}
	}

	// Prepare host configuration
	hostConfig := &container.HostConfig{
		RestartPolicy: container.RestartPolicy{
			Name: config.RestartPolicy,
		},
	}

	// Configure port mappings if ExposePorts is true
	if config.ExposePorts && config.Port > 0 {
		hostConfig.PortBindings = nat.PortMap{
			nat.Port(fmt.Sprintf("%d/tcp", config.Port)): []nat.PortBinding{
				{
					HostIP:   "0.0.0.0",
					HostPort: strconv.Itoa(config.Port),
				},
			},
		}
	}

	// Add custom port mappings
	for _, mapping := range config.PortMappings {
		if err := c.addPortMapping(hostConfig, mapping); err != nil {
			return fmt.Errorf("failed to add port mapping '%s': %w", mapping, err)
		}
	}

	// Prepare network configuration
	networkConfig := &network.NetworkingConfig{}
	if config.Network != "" {
		networkConfig.EndpointsConfig = map[string]*network.EndpointSettings{
			config.Network: {},
		}
	}

	// Create container
	resp, err := c.client.ContainerCreate(ctx, containerConfig, hostConfig, networkConfig, nil, config.Name)
	if err != nil {
		return fmt.Errorf("failed to create container: %w", err)
	}

	// Start container
	if err := c.client.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{}); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}

	fmt.Printf("✅ Successfully deployed Docker container '%s' (ID: %s)\n", config.Name, resp.ID[:12])
	return nil
}

// Helper functions for DeployMCPServer
func (c *Client) ensureNetwork(networkName string) error {
	ctx := context.Background()

	// Check if network exists
	networks, err := c.client.NetworkList(ctx, types.NetworkListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list networks: %w", err)
	}

	for _, net := range networks {
		if net.Name == networkName {
			return nil // Network already exists
		}
	}

	// Create network
	_, err = c.client.NetworkCreate(ctx, networkName, types.NetworkCreate{
		Driver: "bridge",
	})
	if err != nil {
		return fmt.Errorf("failed to create network: %w", err)
	}

	fmt.Printf("📡 Created Docker network '%s'\n", networkName)
	return nil
}

func (c *Client) buildLabels(config *DeploymentConfig) map[string]string {
	labels := make(map[string]string)

	// Add default MCP manager labels
	labels["mcp-manager.managed-by"] = "mcp-manager"
	labels["mcp-manager.component"] = "mcp-server"
	labels["mcp-manager.server-name"] = config.Name

	// Add custom labels
	for k, v := range config.Labels {
		labels[k] = v
	}

	return labels
}

func (c *Client) buildEnvVars(environment map[string]string) []string {
	var envVars []string
	for k, v := range environment {
		envVars = append(envVars, fmt.Sprintf("%s=%s", k, v))
	}
	return envVars
}

func (c *Client) addPortMapping(hostConfig *container.HostConfig, mapping string) error {
	// Parse port mapping in format "host:container" or "host:container/protocol"
	parts := strings.Split(mapping, ":")
	if len(parts) != 2 {
		return fmt.Errorf("invalid port mapping format, expected 'host:container'")
	}

	hostPort := parts[0]
	containerPortStr := parts[1]

	// Default protocol is tcp
	protocol := "tcp"
	if strings.Contains(containerPortStr, "/") {
		portParts := strings.Split(containerPortStr, "/")
		containerPortStr = portParts[0]
		protocol = portParts[1]
	}

	containerPort := nat.Port(fmt.Sprintf("%s/%s", containerPortStr, protocol))

	if hostConfig.PortBindings == nil {
		hostConfig.PortBindings = make(nat.PortMap)
	}

	hostConfig.PortBindings[containerPort] = []nat.PortBinding{
		{
			HostIP:   "0.0.0.0",
			HostPort: hostPort,
		},
	}

	return nil
}

// HealthResult represents health check result
type HealthResult struct {
	Status       string        `json:"status"`
	Endpoint     string        `json:"endpoint"`
	ResponseTime time.Duration `json:"response_time"`
	Message      string        `json:"message"`
	Details      interface{}   `json:"details,omitempty"`
}

// CheckServerHealth checks the health of a server
func (c *Client) CheckServerHealth(serverName, endpoint string, timeout time.Duration) (*HealthResult, error) {
	ctx := context.Background()

	// Get container info
	container, err := c.getContainerByName(serverName)
	if err != nil {
		return &HealthResult{
			Status:   "unhealthy",
			Endpoint: "unknown",
			Message:  fmt.Sprintf("Container not found: %v", err),
		}, nil
	}

	// Check if container is running
	if container.State != "running" {
		return &HealthResult{
			Status:   "unhealthy",
			Endpoint: getContainerEndpoint(*container),
			Message:  fmt.Sprintf("Container is %s", container.State),
		}, nil
	}

	// Get container details for port information
	containerJSON, err := c.client.ContainerInspect(ctx, container.ID)
	if err != nil {
		return &HealthResult{
			Status:   "unhealthy",
			Endpoint: getContainerEndpoint(*container),
			Message:  fmt.Sprintf("Failed to inspect container: %v", err),
		}, nil
	}

	// Build endpoint URL
	endpointURL := c.buildHealthEndpoint(containerJSON, endpoint)

	// For now, if container is running, consider it healthy
	// In a full implementation, you would make an HTTP request to the health endpoint
	return &HealthResult{
		Status:   "healthy",
		Endpoint: endpointURL,
		Message:  "Container is running",
	}, nil
}

func (c *Client) buildHealthEndpoint(containerJSON types.ContainerJSON, healthPath string) string {
	// Try to find exposed port
	for _, bindings := range containerJSON.NetworkSettings.Ports {
		if len(bindings) > 0 && bindings[0].HostPort != "" {
			return fmt.Sprintf("http://localhost:%s%s", bindings[0].HostPort, healthPath)
		}
	}

	// Fallback to container IP if no port binding
	if containerJSON.NetworkSettings.IPAddress != "" {
		return fmt.Sprintf("http://%s%s", containerJSON.NetworkSettings.IPAddress, healthPath)
	}

	return fmt.Sprintf("container:%s%s", containerJSON.Name, healthPath)
}

// LogOptions represents log fetching options
type LogOptions struct {
	Follow     bool
	Tail       int
	Since      string
	Timestamps bool
}

// GetServerLogs gets logs from a server
func (c *Client) GetServerLogs(serverName string, options *LogOptions) error {
	ctx := context.Background()

	// Get container
	container, err := c.getContainerByName(serverName)
	if err != nil {
		return fmt.Errorf("container not found: %w", err)
	}

	// Build log options
	logOptions := types.ContainerLogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Follow:     options.Follow,
		Timestamps: options.Timestamps,
	}

	if options.Tail > 0 {
		tail := strconv.Itoa(options.Tail)
		logOptions.Tail = tail
	}

	if options.Since != "" {
		logOptions.Since = options.Since
	}

	// Get log stream
	reader, err := c.client.ContainerLogs(ctx, container.ID, logOptions)
	if err != nil {
		return fmt.Errorf("failed to get container logs: %w", err)
	}
	defer reader.Close()

	// Copy logs to stdout
	_, err = io.Copy(os.Stdout, reader)
	return err
}

// DeleteOptions represents deletion options
type DeleteOptions struct {
	KeepVolumes bool
	Force       bool
}

// ContainerExists checks if a container exists
func (c *Client) ContainerExists(name string) (bool, error) {
	_, err := c.getContainerByName(name)
	if err != nil {
		return false, nil // Container doesn't exist
	}
	return true, nil
}

// DeleteMCPServer deletes an MCP server container
func (c *Client) DeleteMCPServer(serverName string, options *DeleteOptions) error {
	ctx := context.Background()

	// Get container
	container, err := c.getContainerByName(serverName)
	if err != nil {
		return fmt.Errorf("container not found: %w", err)
	}

	// Stop container if it's running
	if container.State == "running" {
		fmt.Printf("🛑 Stopping container '%s'...\n", serverName)
		timeout := time.Duration(10) * time.Second
		if err := c.client.ContainerStop(ctx, container.ID, &timeout); err != nil {
			if options.Force {
				fmt.Printf("⚠️  Failed to stop gracefully, forcing kill...\n")
				if err := c.client.ContainerKill(ctx, container.ID, "SIGKILL"); err != nil {
					return fmt.Errorf("failed to kill container: %w", err)
				}
			} else {
				return fmt.Errorf("failed to stop container: %w", err)
			}
		}
	}

	// Remove container
	fmt.Printf("🗑️  Removing container '%s'...\n", serverName)
	removeOptions := types.ContainerRemoveOptions{
		Force:         options.Force,
		RemoveVolumes: !options.KeepVolumes,
	}

	if err := c.client.ContainerRemove(ctx, container.ID, removeOptions); err != nil {
		return fmt.Errorf("failed to remove container: %w", err)
	}

	fmt.Printf("✅ Successfully deleted container '%s'\n", serverName)
	return nil
}

// getContainerByName gets a container by name
func (c *Client) getContainerByName(name string) (*types.Container, error) {
	containers, err := c.client.ContainerList(context.Background(), types.ContainerListOptions{
		All: true,
	})
	if err != nil {
		return nil, err
	}

	for _, container := range containers {
		for _, containerName := range container.Names {
			if strings.TrimPrefix(containerName, "/") == name {
				return &container, nil
			}
		}
	}

	return nil, fmt.Errorf("container %s not found", name)
}
