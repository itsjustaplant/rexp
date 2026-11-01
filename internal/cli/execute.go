package cli

import (
	"context"

	"github.com/spf13/cobra"

	"rexp/internal/config"
	"rexp/internal/programstatus"
)

// Execute owns the application lifecycle, including failures that Cobra's
// pre/post-run hooks cannot observe (argument errors and unsuccessful commands).
func Execute(root *cobra.Command, reporter *programstatus.Reporter) error {
	ctx := root.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	root.SetContext(programstatus.WithReporter(ctx, reporter))
	err := config.Initialize()
	command := root
	if err == nil {
		command, err = root.ExecuteC()
	}
	if ctx.Err() != nil {
		err = ctx.Err()
	}
	name := root.CommandPath()
	if command != nil {
		name = command.CommandPath()
	}
	reporter.Finish(err, name)
	return err
}
