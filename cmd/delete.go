package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
	"github.com/protocol-server-manager/internal/kubernetes"
	"github.com/protocol-server-manager/internal/docker"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete [server-name]",
	Short: "Delete an MCP server",
	Long: `Delete an MCP server and all its associated resources.
	
For Kubernetes: removes deployment, service, and ingress
For Docker: stops and removes container

Examples:
  # Delete a server (with confirmation prompt)
  mcp-manager delete my-server

  # Delete without confirmation
  mcp-manager delete my-server --force

  # Delete from specific namespace
  mcp-manager delete my-server --namespace my-namespace`,
	Args: cobra.ExactArgs(1),
	RunE: runDelete,
}

func init() {
	rootCmd.AddCommand(deleteCmd)

	deleteCmd.Flags().Bool("force", false, "Delete without confirmation prompt")
	deleteCmd.Flags().String("namespace", "", "Kubernetes namespace (overrides config)")
	deleteCmd.Flags().Bool("keep-volumes", false, "Keep persistent volumes (Kubernetes only)")
}

func runDelete(cmd *cobra.Command, args []string) error {
	serverName := args[0]
	mode := GetMode()
	force, _ := cmd.Flags().GetBool("force")

	color.New(color.FgRed, color.Bold).Printf("🗑️  Deleting MCP server '%s' (mode: %s)\n\n", serverName, mode)

	// Confirmation prompt unless --force is used
	if !force {
		if !confirmDeletion(serverName, mode) {
			color.Yellow("❌ Deletion cancelled")
			return nil
		}
	}

	switch mode {
	case "kubernetes":
		return deleteFromKubernetes(cmd, serverName)
	case "docker":
		return deleteFromDocker(cmd, serverName)
	default:
		return fmt.Errorf("unsupported mode: %s", mode)
	}
}

func confirmDeletion(serverName, mode string) bool {
	color.New(color.FgYellow, color.Bold).Printf("⚠️  This will permanently delete the MCP server '%s' and all its resources.\n", serverName)
	
	if mode == "kubernetes" {
		fmt.Println("   - Kubernetes Deployment")
		fmt.Println("   - Kubernetes Service")
		fmt.Println("   - Kubernetes Ingress (if exists)")
		fmt.Println("   - ConfigMaps and Secrets (if exists)")
	} else {
		fmt.Println("   - Docker Container")
		fmt.Println("   - Associated volumes (unless --keep-volumes is used)")
	}
	
	fmt.Print("\nAre you sure you want to continue? (y/N): ")
	
	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}
	
	response = strings.TrimSpace(strings.ToLower(response))
	return response == "y" || response == "yes"
}

func deleteFromKubernetes(cmd *cobra.Command, serverName string) error {
	namespace := getStringFlagOrConfig(cmd, "namespace", "kube_namespace", "default")
	keepVolumes, _ := cmd.Flags().GetBool("keep-volumes")

	client, err := kubernetes.NewClient(GetConfig().KubeConfig)
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	color.New(color.FgBlue).Println("🔄 Deleting from Kubernetes...")

	// Check if server exists
	exists, err := client.ServerExists(serverName, namespace)
	if err != nil {
		return fmt.Errorf("failed to check if server exists: %w", err)
	}

	if !exists {
		color.Yellow("⚠️  Server '%s' not found in namespace '%s'", serverName, namespace)
		return nil
	}

	// Delete resources
	deleteOptions := &kubernetes.DeleteOptions{
		Namespace:   namespace,
		KeepVolumes: keepVolumes,
	}

	if err := client.DeleteMCPServer(serverName, deleteOptions); err != nil {
		return fmt.Errorf("failed to delete from Kubernetes: %w", err)
	}

	color.New(color.FgGreen).Printf("✅ Successfully deleted '%s' from Kubernetes namespace '%s'\n", serverName, namespace)
	return nil
}

func deleteFromDocker(cmd *cobra.Command, serverName string) error {
	keepVolumes, _ := cmd.Flags().GetBool("keep-volumes")

	client, err := docker.NewClient(GetConfig().DockerHost)
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}

	color.New(color.FgBlue).Println("🔄 Deleting from Docker...")

	// Check if container exists
	exists, err := client.ContainerExists(serverName)
	if err != nil {
		return fmt.Errorf("failed to check if container exists: %w", err)
	}

	if !exists {
		color.Yellow("⚠️  Container '%s' not found", serverName)
		return nil
	}

	// Delete container
	deleteOptions := &docker.DeleteOptions{
		KeepVolumes: keepVolumes,
		Force:       true, // Force stop if running
	}

	if err := client.DeleteMCPServer(serverName, deleteOptions); err != nil {
		return fmt.Errorf("failed to delete from Docker: %w", err)
	}

	color.New(color.FgGreen).Printf("✅ Successfully deleted '%s' from Docker\n", serverName)
	return nil
}
