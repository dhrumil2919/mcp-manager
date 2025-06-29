package main

import (
	"fmt"
	"os"

	"github.com/protocol-server-manager/cmd"
)

// Version information (set by build flags)
var (
	Version   = "dev"
	Commit    = "unknown"
	BuildTime = "unknown"
)

func main() {
	// Set version information for the CLI
	cmd.SetVersionInfo(Version, Commit, BuildTime)

	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
