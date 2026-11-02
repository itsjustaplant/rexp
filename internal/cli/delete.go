package cli

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"strings"

	"github.com/spf13/cobra"

	"github.com/itsjustaplant/rexp/internal/config"
	"github.com/itsjustaplant/rexp/internal/programstatus"
)

func deleteProjectCommand() *cobra.Command {
	var yes bool
	cmd := &cobra.Command{
		Use:     "delete <project>",
		Short:   "Delete a saved project configuration",
		Long:    "Delete a project's YAML file from ~/.rexp/projects after confirmation.\nThe project's source directory and any running Rex session are left alone.",
		Example: "  rexp delete my-app\n  rexp delete my-app --yes",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errors.New("missing project name\nUsage: rexp delete <project> [--yes]\nExample: rexp delete my-app")
			}
			if err := cobra.ExactArgs(1)(cmd, args); err != nil {
				return err
			}
			return config.ValidateProjectName(args[0])
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			path, err := config.ProjectFile(args[0])
			if err != nil {
				return err
			}
			if !yes {
				programstatus.Approval(cmd.Context(), "Approve deletion of project "+args[0])
				cmd.PrintErrf("Delete project %q configuration at %s? [y/N] ", args[0], path)
				answer, err := readConfirmation(cmd.Context(), cmd.InOrStdin())
				if err != nil && (!errors.Is(err, io.EOF) || len(answer) == 0) {
					return fmt.Errorf("read deletion confirmation: %w; use --yes to delete without prompting", err)
				}
				answer = strings.ToLower(strings.TrimSpace(answer))
				if answer != "y" && answer != "yes" {
					programstatus.Idle(cmd.Context(), "Deletion cancelled; project configuration kept")
					cmd.Println("Cancelled; project configuration was kept.")
					return nil
				}
			}
			programstatus.Working(cmd.Context(), "Deleting project "+args[0])
			path, err = config.DeleteProject(args[0])
			if err != nil {
				return err
			}
			cmd.Printf("Deleted %s\n", path)
			return nil
		},
	}
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Delete without asking for confirmation")
	return cmd
}

// An interrupted prompt must return even while its terminal is awaiting input.
func readConfirmation(ctx context.Context, input io.Reader) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	type result struct {
		answer string
		err    error
	}
	ready := make(chan result, 1)
	go func() { answer, err := bufio.NewReader(input).ReadString('\n'); ready <- result{answer, err} }()
	select {
	case <-ctx.Done():
		return "", ctx.Err()
	case read := <-ready:
		if err := ctx.Err(); err != nil {
			return "", err
		}
		return read.answer, read.err
	}
}
