// Package main provides command-line interface entrypoint for rosterbalance
package main

import "fmt"

// These variables are replaced by -ldflags during the build.
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// Version holds the build metadata injected by GoReleaser.
type Version struct {
	Version string
	Commit  string
	Date    string
}

// GetVersion returns a populated Version struct.
func GetVersion() Version {
	return Version{
		Version: version,
		Commit:  commit,
		Date:    date,
	}
}

func main() {
	v := GetVersion()
	fmt.Printf("app version %s (commit: %s, built at: %s)\n", v.Version, v.Commit, v.Date)
}
