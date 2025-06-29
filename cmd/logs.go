package cmd

import (
	"fmt"

	"github.com/fatih/color"
	"github.com/protocol-server-manager/internal/docker"
	"github.com/protocol-server-manager/internal/kubernetes"
	"github.com/spf13/cobra"
)

var logsCmd = &cobra.Command{
	Use:   "logs [server-name]",
	Short: "Fetch logs from MCP server",
	Long: `Fetch and display logs from an MCP server.
	
For Kubernetes: fetches logs from pods
For Docker: fetches logs from containers

Examples:
  # Get recent logs
  mcp-manager logs my-server

  # Follow logs (tail -f style)
  mcp-manager logs my-server --follow

  # Get last 100 lines
  mcp-manager logs my-server --tail 100

  # Get logs since 1 hour ago
  mcp-manager logs my-server --since 1h

  # Get logs with timestamps
  mcp-manager logs my-server --timestamps`,
	Args: cobra.ExactArgs(1),
	RunE: runLogs,
}

func init() {
	rootCmd.AddCommand(logsCmd)

	logsCmd.Flags().BoolP("follow", "f", false, "Follow log output (like tail -f)")
	logsCmd.Flags().Int("tail", 50, "Number of lines to show from the end of the logs")
	logsCmd.Flags().String("since", "", "Show logs since timestamp (e.g. 2013-01-02T13:23:37Z) or relative (e.g. 42m for 42 minutes)")
	logsCmd.Flags().Bool("timestamps", false, "Include timestamps in output")
	logsCmd.Flags().String("namespace", "", "Kubernetes namespace (overrides config)")
	logsCmd.Flags().String("container", "", "Container name (for multi-container pods)")
	logsCmd.Flags().Bool("previous", false, "Show logs from previous container instance")
}

func runLogs(cmd *cobra.Command, args []string) error {
	serverName := args[0]
	mode := GetMode()

	color.New(color.FgCyan, color.Bold).Printf("📜 Fetching logs for '%s' (mode: %s)\n\n", serverName, mode)

	switch mode {
	case "kubernetes":
		return fetchKubernetesLogs(cmd, serverName)
	case "docker":
		return fetchDockerLogs(cmd, serverName)
	default:
		return fmt.Errorf("unsupported mode: %s", mode)
	}
}

func fetchKubernetesLogs(cmd *cobra.Command, serverName string) error {
	namespace := getStringFlagOrConfig(cmd, "namespace", "kube_namespace", "default")
	follow, _ := cmd.Flags().GetBool("follow")
	tail, _ := cmd.Flags().GetInt("tail")
	since, _ := cmd.Flags().GetString("since")
	timestamps, _ := cmd.Flags().GetBool("timestamps")
	container, _ := cmd.Flags().GetString("container")
	previous, _ := cmd.Flags().GetBool("previous")

	client, err := kubernetes.NewClient(GetConfig().KubeConfig)
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	logOptions := &kubernetes.LogOptions{
		Follow:     follow,
		Tail:       tail,
		Since:      since,
		Timestamps: timestamps,
		Container:  container,
		Previous:   previous,
	}

	if follow {
		color.Yellow("Following logs for '%s' - Press Ctrl+C to stop\n", serverName)
	}

	return client.GetServerLogs(serverName, namespace, logOptions)
}

func fetchDockerLogs(cmd *cobra.Command, serverName string) error {
	follow, _ := cmd.Flags().GetBool("follow")
	tail, _ := cmd.Flags().GetInt("tail")
	since, _ := cmd.Flags().GetString("since")
	timestamps, _ := cmd.Flags().GetBool("timestamps")

	client, err := docker.NewClient(GetConfig().DockerHost)
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}

	logOptions := &docker.LogOptions{
		Follow:     follow,
		Tail:       tail,
		Since:      since,
		Timestamps: timestamps,
	}

	if follow {
		color.Yellow("Following logs for '%s' - Press Ctrl+C to stop\n", serverName)
	}

	return client.GetServerLogs(serverName, logOptions)
}
