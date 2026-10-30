package main

import (
	"fmt"
	"os"

	"rexp/internal/cli"
)

// Set at build time with -ldflags "-X main.version=...".
var version = "0.0.1"

func main() {
	if err := cli.NewCommand(version).Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}
