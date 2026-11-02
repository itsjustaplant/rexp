// Package hooks runs local lifecycle commands and observes Rex session closure.
package hooks

import (
	"context"
	"fmt"
	"io"
	"rexp/internal/panerunner"
	"rexp/internal/workspace"
)

func Run(ctx context.Context, project *workspace.Project, session, phase string, commands [][]string, out, stderr io.Writer) error {
	if len(commands) == 0 {
		return nil
	}
	env := workspace.MergeEnvironment(project.Env, map[string]string{"REXP_PROJECT": project.Name, "REXP_SESSION_ID": session})
	_, err := panerunner.Execute(ctx, panerunner.Payload{Version: 1, Commands: commands, Env: env, Cwd: project.Root}, nil, out, stderr)
	if err != nil {
		return fmt.Errorf("hooks.%s: %w", phase, err)
	}
	return nil
}
