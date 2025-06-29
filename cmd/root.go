package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	cfgFile string
	config  *Config

	// Version information
	version   = "dev"
	commit    = "unknown"
	buildTime = "unknown"
)

// Config represents the application configuration
type Config struct {
	DefaultMode   string         `mapstructure:"default_mode"`   // "kubernetes" or "docker"
	KubeConfig    string         `mapstructure:"kube_config"`    // Path to kubeconfig file
	KubeNamespace string         `mapstructure:"kube_namespace"` // Default Kubernetes namespace
	DockerHost    string         `mapstructure:"docker_host"`    // Docker daemon host
	Defaults      DefaultsConfig `mapstructure:"defaults"`       // Default configurations
}

// DefaultsConfig holds default values for deployments
type DefaultsConfig struct {
	Kubernetes KubernetesDefaults `mapstructure:"kubernetes"`
	Docker     DockerDefaults     `mapstructure:"docker"`
}

// KubernetesDefaults holds default values for Kubernetes deployments
type KubernetesDefaults struct {
	Image         string            `mapstructure:"image"`
	Port          int               `mapstructure:"port"`
	HealthPath    string            `mapstructure:"health_path"`
	Command       []string          `mapstructure:"command"`
	Args          []string          `mapstructure:"args"`
	Replicas      int               `mapstructure:"replicas"`
	CPURequest    string            `mapstructure:"cpu_request"`
	CPULimit      string            `mapstructure:"cpu_limit"`
	MemoryRequest string            `mapstructure:"memory_request"`
	MemoryLimit   string            `mapstructure:"memory_limit"`
	IngressHost   string            `mapstructure:"ingress_host"`
	IngressClass  string            `mapstructure:"ingress_class"`
	Labels        map[string]string `mapstructure:"labels"`
	Annotations   map[string]string `mapstructure:"annotations"`
}

// DockerDefaults holds default values for Docker deployments
type DockerDefaults struct {
	Image         string            `mapstructure:"image"`
	Port          int               `mapstructure:"port"`
	HealthPath    string            `mapstructure:"health_path"`
	Command       []string          `mapstructure:"command"`
	Args          []string          `mapstructure:"args"`
	Network       string            `mapstructure:"network"`
	RestartPolicy string            `mapstructure:"restart_policy"`
	Labels        map[string]string `mapstructure:"labels"`
	ExposePorts   bool              `mapstructure:"expose_ports"`
}

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "mcp-manager",
	Short: "MCP Server Management Tool",
	Long: `A command-line tool for managing Model Context Protocol (MCP) servers.
Supports both Kubernetes and Docker deployment modes.

Examples:
  # List all MCP servers in Kubernetes mode
  mcp-manager list --mode kubernetes

  # Deploy an MCP server to Docker
  mcp-manager deploy --mode docker --name my-server --image my-mcp-server:latest

  # Get server health status
  mcp-manager health my-server

  # View server logs
  mcp-manager logs my-server --follow`,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	cobra.OnInitialize(initConfig)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.mcp-manager.yaml)")
	rootCmd.PersistentFlags().String("mode", "", "deployment mode: kubernetes or docker (overrides config)")
	rootCmd.PersistentFlags().String("kubeconfig", "", "path to kubeconfig file (overrides config)")
	rootCmd.PersistentFlags().String("namespace", "default", "kubernetes namespace (overrides config)")

	// Bind flags to viper
	viper.BindPFlag("mode", rootCmd.PersistentFlags().Lookup("mode"))
	viper.BindPFlag("kube_config", rootCmd.PersistentFlags().Lookup("kubeconfig"))
	viper.BindPFlag("kube_namespace", rootCmd.PersistentFlags().Lookup("namespace"))
}

// initConfig reads in config file and ENV variables.
func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		// Find home directory.
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)

		// Search config in home directory with name ".mcp-manager" (without extension).
		viper.AddConfigPath(home)
		viper.AddConfigPath(".")
		viper.SetConfigType("yaml")
		viper.SetConfigName(".mcp-manager")
	}

	// Set defaults
	viper.SetDefault("default_mode", "kubernetes")
	viper.SetDefault("kube_namespace", "default")
	viper.SetDefault("docker_host", "unix:///var/run/docker.sock")

	// Kubernetes defaults
	viper.SetDefault("defaults.kubernetes.image", "")
	viper.SetDefault("defaults.kubernetes.port", 8000)
	viper.SetDefault("defaults.kubernetes.health_path", "")
	viper.SetDefault("defaults.kubernetes.replicas", 1)
	viper.SetDefault("defaults.kubernetes.cpu_request", "100m")
	viper.SetDefault("defaults.kubernetes.cpu_limit", "500m")
	viper.SetDefault("defaults.kubernetes.memory_request", "128Mi")
	viper.SetDefault("defaults.kubernetes.memory_limit", "512Mi")
	viper.SetDefault("defaults.kubernetes.ingress_class", "nginx")

	// Docker defaults
	viper.SetDefault("defaults.docker.image", "")
	viper.SetDefault("defaults.docker.port", 8000)
	viper.SetDefault("defaults.docker.health_path", "")
	viper.SetDefault("defaults.docker.network", "mcp-network")
	viper.SetDefault("defaults.docker.restart_policy", "unless-stopped")
	viper.SetDefault("defaults.docker.expose_ports", true)

	viper.AutomaticEnv() // read in environment variables that match

	// If a config file is found, read it in.
	if err := viper.ReadInConfig(); err == nil {
		fmt.Fprintln(os.Stderr, "Using config file:", viper.ConfigFileUsed())
	}

	// Unmarshal config
	config = &Config{}
	if err := viper.Unmarshal(config); err != nil {
		fmt.Fprintf(os.Stderr, "Error unmarshaling config: %v\n", err)
		os.Exit(1)
	}

	// Set kubeconfig default if not specified
	if config.KubeConfig == "" {
		if home, err := os.UserHomeDir(); err == nil {
			config.KubeConfig = filepath.Join(home, ".kube", "config")
		}
	}
}

// GetConfig returns the current configuration
func GetConfig() *Config {
	return config
}

// GetMode returns the current deployment mode (kubernetes or docker)
func GetMode() string {
	if mode := viper.GetString("mode"); mode != "" {
		return mode
	}
	return config.DefaultMode
}

// SetVersionInfo sets the version information
func SetVersionInfo(v, c, bt string) {
	version = v
	commit = c
	buildTime = bt
}

// GetVersionInfo returns the version information
func GetVersionInfo() (string, string, string) {
	return version, commit, buildTime
}
