package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/fatih/color"
	"github.com/olekukonko/tablewriter"
	"github.com/protocol-server-manager/internal/docker"
	"github.com/protocol-server-manager/internal/kubernetes"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List all MCP servers",
	Long:  `List all MCP servers in the current deployment mode with their status and health information.`,
	RunE:  runList,
}

func init() {
	rootCmd.AddCommand(listCmd)

	// Add flags specific to list command
	listCmd.Flags().Bool("health", false, "Include health check information")
	listCmd.Flags().String("namespace", "", "Kubernetes namespace to list from (overrides config)")
}

func runList(cmd *cobra.Command, args []string) error {
	mode := GetMode()
	includeHealth, _ := cmd.Flags().GetBool("health")
	namespace, _ := cmd.Flags().GetString("namespace")

	if namespace == "" {
		namespace = GetConfig().KubeNamespace
	}

	color.New(color.FgCyan, color.Bold).Printf("📋 Listing MCP servers (mode: %s)\n\n", mode)

	switch mode {
	case "kubernetes":
		return listKubernetesServers(namespace, includeHealth)
	case "docker":
		return listDockerServers(includeHealth)
	default:
		return fmt.Errorf("unsupported mode: %s. Use 'kubernetes' or 'docker'", mode)
	}
}

func listKubernetesServers(namespace string, includeHealth bool) error {
	client, err := kubernetes.NewClient(GetConfig().KubeConfig)
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	servers, err := client.ListMCPServers(namespace)
	if err != nil {
		return fmt.Errorf("failed to list MCP servers: %w", err)
	}

	if len(servers) == 0 {
		color.Yellow("No MCP servers found in namespace '%s'", namespace)
		return nil
	}

	// Create table
	table := tablewriter.NewWriter(os.Stdout)
	headers := []string{"Name", "Image", "Status", "Replicas", "Endpoint", "Age"}
	if includeHealth {
		headers = append(headers, "Health")
	}
	table.SetHeader(headers)
	table.SetBorder(false)

	for _, server := range servers {
		row := []string{
			server.Name,
			server.Image,
			getStatusColor(server.Status),
			fmt.Sprintf("%d/%d", server.ReadyReplicas, server.Replicas),
			server.Endpoint, // Show ingress endpoint
			server.Age,
		}

		if includeHealth {
			healthStatus := "Unknown"
			health, err := client.CheckServerHealth(server.Name, namespace, "/health", 10*time.Second)
			if err == nil {
				healthStatus = getHealthColor(health.Status)
			}
			row = append(row, healthStatus)
		}

		table.Append(row)
	}

	table.Render()

	color.New(color.FgGreen).Printf("\n✅ Found %d MCP server(s) in namespace '%s'\n", len(servers), namespace)
	return nil
}

func listDockerServers(includeHealth bool) error {
	client, err := docker.NewClient(GetConfig().DockerHost)
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}

	servers, err := client.ListMCPServers()
	if err != nil {
		return fmt.Errorf("failed to list MCP servers: %w", err)
	}

	if len(servers) == 0 {
		color.Yellow("No MCP servers found")
		return nil
	}

	// Create table
	table := tablewriter.NewWriter(os.Stdout)
	headers := []string{"Name", "Image", "Status", "Endpoint", "Age"}
	if includeHealth {
		headers = append(headers, "Health")
	}
	table.SetHeader(headers)
	table.SetBorder(false)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)

	for _, server := range servers {
		row := []string{
			server.Name,
			server.Image,
			getStatusColor(server.Status),
			server.Endpoint, // Show Docker port mapping
			server.Age,
		}

		if includeHealth {
			healthStatus := "Unknown"
			health, err := client.CheckServerHealth(server.Name, "/health", 10*time.Second)
			if err == nil {
				healthStatus = getHealthColor(health.Status)
			}
			row = append(row, healthStatus)
		}

		table.Append(row)
	}

	table.Render()

	color.New(color.FgGreen).Printf("\n✅ Found %d MCP server(s)\n", len(servers))
	return nil
}

func getStatusColor(status string) string {
	switch status {
	case "Running", "Available":
		return color.New(color.FgGreen).Sprint(status)
	case "Pending", "Creating":
		return color.New(color.FgYellow).Sprint(status)
	case "Failed", "Error":
		return color.New(color.FgRed).Sprint(status)
	default:
		return status
	}
}

func getHealthColor(health string) string {
	switch health {
	case "Healthy":
		return color.New(color.FgGreen).Sprint("✅ Healthy")
	case "Unhealthy":
		return color.New(color.FgRed).Sprint("❌ Unhealthy")
	default:
		return color.New(color.FgYellow).Sprint("⚠️  Unknown")
	}
}
