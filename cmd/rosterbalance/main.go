// Package main provides the rosterbalance CLI entrypoint.
package main

import (
	"fmt"
	"os"

	"github.com/iilei/roster-balance-cli/internal/cli"
)

// These variables are replaced by -ldflags during the build.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// GetVersion returns the build metadata injected by GoReleaser.
func GetVersion() cli.Version {
	return cli.Version{
		Version: version,
		Commit:  commit,
		Date:    date,
	}
}

func main() {
	if err := cli.NewRootCommand(GetVersion()).Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
