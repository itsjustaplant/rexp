package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"

	"github.com/itsjustaplant/rexp/internal/cli"
	"github.com/itsjustaplant/rexp/internal/hooks"
	"github.com/itsjustaplant/rexp/internal/panerunner"
	"github.com/itsjustaplant/rexp/internal/programstatus"
)

// Set at build time with -ldflags "-X main.version=...".
var version = "1.0.0"

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
	if len(os.Args) > 1 && os.Args[1] == hooks.Mode {
		if err := hooks.Worker(ctx, os.Args[2:], os.Stdout, os.Stderr); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
		return
	}
	command := cli.NewCommand(version)
	command.SetContext(ctx)
	if err := cli.ExecuteArguments(command, programstatus.NewTerminal(os.Stderr), os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		if ctx.Err() != nil {
			os.Exit(130)
		}
		os.Exit(1)
	}
}
