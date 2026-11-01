package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"rexp/internal/cli"
	"rexp/internal/panerunner"
	"rexp/internal/programstatus"
)

// Set at build time with -ldflags "-X main.version=...".
var version = "0.5.0"

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt)
	defer stop()
	if len(os.Args) > 1 && os.Args[1] == panerunner.Mode {
		code, err := panerunner.Run(ctx, os.Args[2:], os.Stdin, os.Stdout, os.Stderr)
		if err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
		}
		os.Exit(code)
	}
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
