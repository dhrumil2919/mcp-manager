package cmd

import (
	"fmt"
	"time"

	"github.com/fatih/color"
	"github.com/protocol-server-manager/internal/docker"
	"github.com/protocol-server-manager/internal/kubernetes"
	"github.com/spf13/cobra"
)

var healthCmd = &cobra.Command{
	Use:   "health [server-name]",
	Short: "Check health status of MCP server(s)",
	Long: `Check the health status of one or more MCP servers.
	
Uses the configured health check endpoint to verify server status.
If no server name is provided, checks all servers.

Examples:
  # Check health of specific server
  mcp-manager health my-server

  # Check health of all servers
  mcp-manager health

  # Check health with custom endpoint
  mcp-manager health my-server --endpoint /api/health

  # Watch health status (refresh every 5 seconds)
  mcp-manager health my-server --watch`,
	Args: cobra.MaximumNArgs(1),
	RunE: runHealth,
}

func init() {
	rootCmd.AddCommand(healthCmd)

	healthCmd.Flags().String("endpoint", "", "Custom health check endpoint (overrides config default)")
	healthCmd.Flags().Bool("watch", false, "Watch health status (refresh every 5 seconds)")
	healthCmd.Flags().Duration("timeout", 10*time.Second, "Health check timeout")
	healthCmd.Flags().String("namespace", "", "Kubernetes namespace (overrides config)")
}

func runHealth(cmd *cobra.Command, args []string) error {
	mode := GetMode()
	watch, _ := cmd.Flags().GetBool("watch")

	if len(args) == 1 {
		// Check specific server
		serverName := args[0]
		if watch {
			return watchServerHealth(cmd, serverName, mode)
		}
		return checkServerHealth(cmd, serverName, mode)
	} else {
		// Check all servers
		if watch {
			return watchAllServersHealth(cmd, mode)
		}
		return checkAllServersHealth(cmd, mode)
	}
}

func checkServerHealth(cmd *cobra.Command, serverName, mode string) error {
	color.New(color.FgCyan, color.Bold).Printf("🏥 Checking health of '%s' (mode: %s)\n\n", serverName, mode)

	switch mode {
	case "kubernetes":
		return checkKubernetesServerHealth(cmd, serverName)
	case "docker":
		return checkDockerServerHealth(cmd, serverName)
	default:
		return fmt.Errorf("unsupported mode: %s", mode)
	}
}

func checkAllServersHealth(cmd *cobra.Command, mode string) error {
	color.New(color.FgCyan, color.Bold).Printf("🏥 Checking health of all servers (mode: %s)\n\n", mode)

	switch mode {
	case "kubernetes":
		return checkAllKubernetesServersHealth(cmd)
	case "docker":
		return checkAllDockerServersHealth(cmd)
	default:
		return fmt.Errorf("unsupported mode: %s", mode)
	}
}

func watchServerHealth(cmd *cobra.Command, serverName, mode string) error {
	color.New(color.FgCyan, color.Bold).Printf("👀 Watching health of '%s' (mode: %s) - Press Ctrl+C to stop\n\n", serverName, mode)

	for {
		// Clear screen (simple approach)
		fmt.Print("\033[2J\033[H")

		color.New(color.FgCyan, color.Bold).Printf("👀 Watching health of '%s' (mode: %s) - %s\n\n",
			serverName, mode, time.Now().Format("15:04:05"))

		if err := checkServerHealth(cmd, serverName, mode); err != nil {
			color.Red("❌ Error: %v", err)
		}

		fmt.Println("\nRefreshing in 5 seconds... (Press Ctrl+C to stop)")
		time.Sleep(5 * time.Second)
	}
}

func watchAllServersHealth(cmd *cobra.Command, mode string) error {
	color.New(color.FgCyan, color.Bold).Printf("👀 Watching health of all servers (mode: %s) - Press Ctrl+C to stop\n\n", mode)

	for {
		// Clear screen
		fmt.Print("\033[2J\033[H")

		color.New(color.FgCyan, color.Bold).Printf("👀 Watching health of all servers (mode: %s) - %s\n\n",
			mode, time.Now().Format("15:04:05"))

		if err := checkAllServersHealth(cmd, mode); err != nil {
			color.Red("❌ Error: %v", err)
		}

		fmt.Println("\nRefreshing in 5 seconds... (Press Ctrl+C to stop)")
		time.Sleep(5 * time.Second)
	}
}

func checkKubernetesServerHealth(cmd *cobra.Command, serverName string) error {
	namespace := getStringFlagOrConfig(cmd, "namespace", "kube_namespace", "default")
	endpoint, _ := cmd.Flags().GetString("endpoint")
	timeout, _ := cmd.Flags().GetDuration("timeout")

	client, err := kubernetes.NewClient(GetConfig().KubeConfig)
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	health, err := client.CheckServerHealth(serverName, namespace, endpoint, timeout)
	if err != nil {
		color.Red("❌ Failed to check health: %v", err)
		return err
	}

	// Convert kubernetes.HealthResult to cmd.HealthResult
	cmdHealth := &HealthResult{
		Status:       health.Status,
		Endpoint:     health.Endpoint,
		ResponseTime: health.ResponseTime,
		Message:      health.Message,
		Details:      health.Details,
	}

	displayHealthResult(serverName, cmdHealth)
	return nil
}

func checkDockerServerHealth(cmd *cobra.Command, serverName string) error {
	endpoint, _ := cmd.Flags().GetString("endpoint")
	timeout, _ := cmd.Flags().GetDuration("timeout")

	client, err := docker.NewClient(GetConfig().DockerHost)
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}

	health, err := client.CheckServerHealth(serverName, endpoint, timeout)
	if err != nil {
		color.Red("❌ Failed to check health: %v", err)
		return err
	}

	// Convert docker.HealthResult to cmd.HealthResult
	cmdHealth := &HealthResult{
		Status:       health.Status,
		Endpoint:     health.Endpoint,
		ResponseTime: health.ResponseTime,
		Message:      health.Message,
		Details:      health.Details,
	}

	displayHealthResult(serverName, cmdHealth)
	return nil
}

func checkAllKubernetesServersHealth(cmd *cobra.Command) error {
	namespace := getStringFlagOrConfig(cmd, "namespace", "kube_namespace", "default")
	endpoint, _ := cmd.Flags().GetString("endpoint")
	timeout, _ := cmd.Flags().GetDuration("timeout")

	client, err := kubernetes.NewClient(GetConfig().KubeConfig)
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	servers, err := client.ListMCPServers(namespace)
	if err != nil {
		return fmt.Errorf("failed to list servers: %w", err)
	}

	if len(servers) == 0 {
		color.Yellow("No MCP servers found in namespace '%s'", namespace)
		return nil
	}

	healthyCount := 0
	for _, server := range servers {
		health, err := client.CheckServerHealth(server.Name, namespace, endpoint, timeout)
		if err != nil {
			color.Red("❌ %s: Failed to check health - %v", server.Name, err)
			continue
		}

		// Convert kubernetes.HealthResult to cmd.HealthResult
		cmdHealth := &HealthResult{
			Status:       health.Status,
			Endpoint:     health.Endpoint,
			ResponseTime: health.ResponseTime,
			Message:      health.Message,
			Details:      health.Details,
		}

		displayHealthResult(server.Name, cmdHealth)
		if cmdHealth.Status == "healthy" {
			healthyCount++
		}
		fmt.Println() // Add spacing between servers
	}

	// Summary
	color.New(color.FgBlue, color.Bold).Printf("📊 Summary: %d/%d servers healthy\n", healthyCount, len(servers))
	return nil
}

func checkAllDockerServersHealth(cmd *cobra.Command) error {
	endpoint, _ := cmd.Flags().GetString("endpoint")
	timeout, _ := cmd.Flags().GetDuration("timeout")

	client, err := docker.NewClient(GetConfig().DockerHost)
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}

	servers, err := client.ListMCPServers()
	if err != nil {
		return fmt.Errorf("failed to list servers: %w", err)
	}

	if len(servers) == 0 {
		color.Yellow("No MCP servers found")
		return nil
	}

	healthyCount := 0
	for _, server := range servers {
		health, err := client.CheckServerHealth(server.Name, endpoint, timeout)
		if err != nil {
			color.Red("❌ %s: Failed to check health - %v", server.Name, err)
			continue
		}

		// Convert docker.HealthResult to cmd.HealthResult
		cmdHealth := &HealthResult{
			Status:       health.Status,
			Endpoint:     health.Endpoint,
			ResponseTime: health.ResponseTime,
			Message:      health.Message,
			Details:      health.Details,
		}

		displayHealthResult(server.Name, cmdHealth)
		if cmdHealth.Status == "healthy" {
			healthyCount++
		}
		fmt.Println() // Add spacing between servers
	}

	// Summary
	color.New(color.FgBlue, color.Bold).Printf("📊 Summary: %d/%d servers healthy\n", healthyCount, len(servers))
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

func displayHealthResult(serverName string, health *HealthResult) {
	switch health.Status {
	case "healthy":
		color.Green("✅ %s: Healthy", serverName)
	case "unhealthy":
		color.Red("❌ %s: Unhealthy", serverName)
	case "unknown":
		color.Yellow("⚠️  %s: Unknown", serverName)
	default:
		color.White("❓ %s: %s", serverName, health.Status)
	}

	fmt.Printf("   Endpoint: %s\n", health.Endpoint)
	fmt.Printf("   Response Time: %v\n", health.ResponseTime)

	if health.Message != "" {
		fmt.Printf("   Message: %s\n", health.Message)
	}

	if health.Details != nil {
		fmt.Printf("   Details: %v\n", health.Details)
	}
}
