package cmd

import (
	"fmt"
	"runtime"

	"github.com/fatih/color"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Show version information",
	Long:  `Display version, commit, and build information for the MCP Manager CLI.`,
	RunE:  runVersion,
}

func init() {
	rootCmd.AddCommand(versionCmd)
	
	versionCmd.Flags().Bool("short", false, "Show only the version number")
}

func runVersion(cmd *cobra.Command, args []string) error {
	short, _ := cmd.Flags().GetBool("short")
	
	if short {
		fmt.Println(version)
		return nil
	}

	// Get version info
	v, c, bt := GetVersionInfo()
	
	// Display formatted version information
	color.New(color.FgCyan, color.Bold).Println("MCP Manager CLI")
	fmt.Printf("Version:    %s\n", v)
	fmt.Printf("Commit:     %s\n", c)
	fmt.Printf("Built:      %s\n", bt)
	fmt.Printf("Go version: %s\n", runtime.Version())
	fmt.Printf("OS/Arch:    %s/%s\n", runtime.GOOS, runtime.GOARCH)
	
	return nil
}
