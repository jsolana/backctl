package main

import (
	"fmt"
	"os"

	"github.com/jsolana/backctl/internal/cli"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

func main() {
	cmd := cli.NewRootCommand(version, commit, date)
	if err := cmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}
}
