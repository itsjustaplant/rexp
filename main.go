package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"rexp/internal/cli"
	"rexp/internal/programstatus"
)

// Set at build time with -ldflags "-X main.version=...".
var version = "0.3.0"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	command := cli.NewCommand(version)
	command.SetContext(ctx)
	if err := cli.Execute(command, programstatus.NewTerminal(os.Stderr)); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		if ctx.Err() != nil {
			os.Exit(130)
		}
		os.Exit(1)
	}
}
