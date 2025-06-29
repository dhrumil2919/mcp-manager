package cmd

import (
	"fmt"
	"strings"

	"github.com/fatih/color"
	"github.com/protocol-server-manager/internal/docker"
	"github.com/protocol-server-manager/internal/kubernetes"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var deployCmd = &cobra.Command{
	Use:   "deploy [server-name]",
	Short: "Deploy an MCP server",
	Long: `Deploy an MCP server to Kubernetes or Docker with smart defaults.
	
The command uses configuration defaults that can be overridden with flags.
If no image is specified, you must provide one via --image flag.

Examples:
  # Deploy with minimal config (uses defaults from config file)
  mcp-manager deploy my-server --image my-mcp-server:latest

  # Deploy with custom settings
  mcp-manager deploy my-server \
    --image my-mcp-server:latest \
    --port 9000 \
    --replicas 3 \
    --cpu-limit 1000m \
    --memory-limit 1Gi \
    --health-path /api/health

  # Deploy to specific mode
  mcp-manager deploy my-server --image my-app:latest --mode docker`,
	Args: cobra.ExactArgs(1),
	RunE: runDeploy,
}

func init() {
	rootCmd.AddCommand(deployCmd)

	// Required flags
	deployCmd.Flags().String("image", "", "Docker image to deploy (required)")
	deployCmd.MarkFlagRequired("image")

	// Optional override flags
	deployCmd.Flags().Int("port", 0, "Container port (uses config default if not specified)")
	deployCmd.Flags().String("health-path", "", "Health check endpoint path (uses config default if not specified)")
	deployCmd.Flags().StringSlice("command", []string{}, "Container command (overrides config default)")
	deployCmd.Flags().StringSlice("args", []string{}, "Container args (overrides config default)")
	deployCmd.Flags().StringToString("env", map[string]string{}, "Environment variables (key=value)")
	deployCmd.Flags().String("secrets-file", "", "Path to secrets file (JSON or YAML)")

	// Kubernetes specific flags
	deployCmd.Flags().Int("replicas", 0, "Number of replicas (uses config default if not specified)")
	deployCmd.Flags().String("cpu-request", "", "CPU request (uses config default if not specified)")
	deployCmd.Flags().String("cpu-limit", "", "CPU limit (uses config default if not specified)")
	deployCmd.Flags().String("memory-request", "", "Memory request (uses config default if not specified)")
	deployCmd.Flags().String("memory-limit", "", "Memory limit (uses config default if not specified)")
	deployCmd.Flags().String("ingress-host", "", "Ingress host (uses config default if not specified)")
	deployCmd.Flags().String("ingress-class", "", "Ingress class (uses config default if not specified)")
	deployCmd.Flags().String("namespace", "", "Kubernetes namespace (overrides config)")

	// Docker specific flags
	deployCmd.Flags().String("network", "", "Docker network (uses config default if not specified)")
	deployCmd.Flags().String("restart-policy", "", "Restart policy (uses config default if not specified)")
	deployCmd.Flags().Bool("expose-ports", true, "Expose container ports to host")
	deployCmd.Flags().StringSlice("ports", []string{}, "Additional port mappings (host:container)")

	// Common flags
	deployCmd.Flags().StringToString("labels", map[string]string{}, "Additional labels")
	deployCmd.Flags().Bool("dry-run", false, "Show what would be deployed without actually deploying")
}

func runDeploy(cmd *cobra.Command, args []string) error {
	serverName := args[0]
	mode := GetMode()

	// Validate server name
	if !isValidServerName(serverName) {
		return fmt.Errorf("invalid server name '%s'. Must contain only lowercase letters, numbers, and hyphens", serverName)
	}

	color.New(color.FgCyan, color.Bold).Printf("🚀 Deploying MCP server '%s' (mode: %s)\n\n", serverName, mode)

	// Get image
	image, _ := cmd.Flags().GetString("image")
	if image == "" {
		return fmt.Errorf("image is required. Use --image flag to specify Docker image")
	}

	// Build deployment config with smart defaults
	config := buildDeploymentConfig(cmd, serverName, image, mode)

	// Show deployment plan
	if err := showDeploymentPlan(config, mode); err != nil {
		return err
	}

	// Check dry-run flag
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	if dryRun {
		color.Yellow("🔍 Dry run mode - no actual deployment performed")
		return nil
	}

	// Deploy based on mode
	switch mode {
	case "kubernetes":
		return deployToKubernetes(config)
	case "docker":
		return deployToDocker(config)
	default:
		return fmt.Errorf("unsupported mode: %s. Use 'kubernetes' or 'docker'", mode)
	}
}

// DeploymentConfig holds all deployment configuration
type DeploymentConfig struct {
	Name        string
	Image       string
	Port        int
	HealthPath  string
	Command     []string
	Args        []string
	Environment map[string]string
	SecretsFile string

	// Kubernetes specific
	Namespace     string
	Replicas      int
	CPURequest    string
	CPULimit      string
	MemoryRequest string
	MemoryLimit   string
	IngressHost   string
	IngressClass  string

	// Docker specific
	Network       string
	RestartPolicy string
	ExposePorts   bool
	PortMappings  []string

	// Common
	Labels map[string]string
}

func buildDeploymentConfig(cmd *cobra.Command, serverName, image, mode string) *DeploymentConfig {
	config := &DeploymentConfig{
		Name:  serverName,
		Image: image,
	}

	// Get defaults based on mode
	var defaults map[string]interface{}
	if mode == "kubernetes" {
		kubeDefaults := viper.GetStringMap("defaults.kubernetes")
		defaults = make(map[string]interface{})
		for k, v := range kubeDefaults {
			defaults[k] = v
		}
	} else {
		dockerDefaults := viper.GetStringMap("defaults.docker")
		defaults = make(map[string]interface{})
		for k, v := range dockerDefaults {
			defaults[k] = v
		}
	}

	// Apply defaults and overrides for common settings
	config.Port = getIntFlagOrDefault(cmd, "port", defaults, "port", 8000)
	config.HealthPath = getStringFlagOrDefault(cmd, "health-path", defaults, "health_path", "/health")
	config.Command = getStringSliceFlagOrDefault(cmd, "command", defaults, "command")
	config.Args = getStringSliceFlagOrDefault(cmd, "args", defaults, "args")
	config.Environment, _ = cmd.Flags().GetStringToString("env")
	config.SecretsFile, _ = cmd.Flags().GetString("secrets-file")
	config.Labels, _ = cmd.Flags().GetStringToString("labels")

	// Apply mode-specific defaults and overrides
	if mode == "kubernetes" {
		config.Namespace = getStringFlagOrConfig(cmd, "namespace", "kube_namespace", "default")
		config.Replicas = getIntFlagOrDefault(cmd, "replicas", defaults, "replicas", 1)
		config.CPURequest = getStringFlagOrDefault(cmd, "cpu-request", defaults, "cpu_request", "100m")
		config.CPULimit = getStringFlagOrDefault(cmd, "cpu-limit", defaults, "cpu_limit", "500m")
		config.MemoryRequest = getStringFlagOrDefault(cmd, "memory-request", defaults, "memory_request", "128Mi")
		config.MemoryLimit = getStringFlagOrDefault(cmd, "memory-limit", defaults, "memory_limit", "512Mi")
		config.IngressHost = getStringFlagOrDefault(cmd, "ingress-host", defaults, "ingress_host", "")
		config.IngressClass = getStringFlagOrDefault(cmd, "ingress-class", defaults, "ingress_class", "nginx")
	} else {
		config.Network = getStringFlagOrDefault(cmd, "network", defaults, "network", "mcp-network")
		config.RestartPolicy = getStringFlagOrDefault(cmd, "restart-policy", defaults, "restart_policy", "unless-stopped")
		config.ExposePorts, _ = cmd.Flags().GetBool("expose-ports")
		config.PortMappings, _ = cmd.Flags().GetStringSlice("ports")
	}

	return config
}

// Helper functions for getting flag values with defaults
func getStringFlagOrDefault(cmd *cobra.Command, flagName string, defaults map[string]interface{}, defaultKey, fallback string) string {
	if cmd.Flags().Changed(flagName) {
		value, _ := cmd.Flags().GetString(flagName)
		return value
	}
	if val, ok := defaults[defaultKey].(string); ok && val != "" {
		return val
	}
	return fallback
}

func getIntFlagOrDefault(cmd *cobra.Command, flagName string, defaults map[string]interface{}, defaultKey string, fallback int) int {
	if cmd.Flags().Changed(flagName) {
		value, _ := cmd.Flags().GetInt(flagName)
		return value
	}
	if val, ok := defaults[defaultKey].(int); ok && val != 0 {
		return val
	}
	return fallback
}

func getStringSliceFlagOrDefault(cmd *cobra.Command, flagName string, defaults map[string]interface{}, defaultKey string) []string {
	if cmd.Flags().Changed(flagName) {
		value, _ := cmd.Flags().GetStringSlice(flagName)
		return value
	}
	if val, ok := defaults[defaultKey].([]interface{}); ok {
		result := make([]string, len(val))
		for i, v := range val {
			result[i] = fmt.Sprintf("%v", v)
		}
		return result
	}
	return []string{}
}

func getStringFlagOrConfig(cmd *cobra.Command, flagName, configKey, fallback string) string {
	if cmd.Flags().Changed(flagName) {
		value, _ := cmd.Flags().GetString(flagName)
		return value
	}
	return viper.GetString(configKey)
}

func isValidServerName(name string) bool {
	if len(name) == 0 || len(name) > 63 {
		return false
	}
	for _, r := range name {
		if !((r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-') {
			return false
		}
	}
	return !strings.HasPrefix(name, "-") && !strings.HasSuffix(name, "-")
}

func showDeploymentPlan(config *DeploymentConfig, mode string) error {
	color.New(color.FgYellow, color.Bold).Println("📋 Deployment Plan:")
	fmt.Printf("  Name: %s\n", config.Name)
	fmt.Printf("  Image: %s\n", config.Image)
	fmt.Printf("  Port: %d\n", config.Port)
	fmt.Printf("  Health Path: %s\n", config.HealthPath)

	if len(config.Command) > 0 {
		fmt.Printf("  Command: %v\n", config.Command)
	}
	if len(config.Args) > 0 {
		fmt.Printf("  Args: %v\n", config.Args)
	}
	if len(config.Environment) > 0 {
		fmt.Printf("  Environment: %v\n", config.Environment)
	}

	if mode == "kubernetes" {
		fmt.Printf("  Namespace: %s\n", config.Namespace)
		fmt.Printf("  Replicas: %d\n", config.Replicas)
		fmt.Printf("  Resources: CPU(%s/%s) Memory(%s/%s)\n",
			config.CPURequest, config.CPULimit, config.MemoryRequest, config.MemoryLimit)
		if config.IngressHost != "" {
			fmt.Printf("  Ingress: %s (class: %s)\n", config.IngressHost, config.IngressClass)
		}
	} else {
		fmt.Printf("  Network: %s\n", config.Network)
		fmt.Printf("  Restart Policy: %s\n", config.RestartPolicy)
		fmt.Printf("  Expose Ports: %t\n", config.ExposePorts)
		if len(config.PortMappings) > 0 {
			fmt.Printf("  Port Mappings: %v\n", config.PortMappings)
		}
	}

	fmt.Println()
	return nil
}

func deployToKubernetes(config *DeploymentConfig) error {
	color.New(color.FgBlue).Println("🔄 Deploying to Kubernetes...")

	client, err := kubernetes.NewClient(GetConfig().KubeConfig)
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	// Convert to kubernetes.DeploymentConfig
	kubeConfig := &kubernetes.DeploymentConfig{
		Name:          config.Name,
		Image:         config.Image,
		Port:          config.Port,
		HealthPath:    config.HealthPath,
		Command:       config.Command,
		Args:          config.Args,
		Environment:   config.Environment,
		SecretsFile:   config.SecretsFile,
		Namespace:     config.Namespace,
		Replicas:      config.Replicas,
		CPURequest:    config.CPURequest,
		CPULimit:      config.CPULimit,
		MemoryRequest: config.MemoryRequest,
		MemoryLimit:   config.MemoryLimit,
		IngressHost:   config.IngressHost,
		IngressClass:  config.IngressClass,
		Labels:        config.Labels,
	}

	if err := client.DeployMCPServer(kubeConfig); err != nil {
		return fmt.Errorf("failed to deploy to Kubernetes: %w", err)
	}

	color.New(color.FgGreen).Printf("✅ Successfully deployed '%s' to Kubernetes\n", config.Name)
	return nil
}

func deployToDocker(config *DeploymentConfig) error {
	color.New(color.FgBlue).Println("🔄 Deploying to Docker...")

	client, err := docker.NewClient(GetConfig().DockerHost)
	if err != nil {
		return fmt.Errorf("failed to create Docker client: %w", err)
	}

	// Convert to docker.DeploymentConfig
	dockerConfig := &docker.DeploymentConfig{
		Name:          config.Name,
		Image:         config.Image,
		Port:          config.Port,
		HealthPath:    config.HealthPath,
		Command:       config.Command,
		Args:          config.Args,
		Environment:   config.Environment,
		Network:       config.Network,
		RestartPolicy: config.RestartPolicy,
		ExposePorts:   config.ExposePorts,
		PortMappings:  config.PortMappings,
		Labels:        config.Labels,
	}

	if err := client.DeployMCPServer(dockerConfig); err != nil {
		return fmt.Errorf("failed to deploy to Docker: %w", err)
	}

	color.New(color.FgGreen).Printf("✅ Successfully deployed '%s' to Docker\n", config.Name)
	return nil
}
