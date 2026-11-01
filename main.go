package main

import (
	"fmt"
	"os"

	"rexp/internal/cli"
	"rexp/internal/config"
)

// Set at build time with -ldflags "-X main.version=...".
var version = "0.3.0"

func main() {
	if err := config.Initialize(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
	if err := cli.NewCommand(version).Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
