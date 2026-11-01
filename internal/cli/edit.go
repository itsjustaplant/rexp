package cli

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"rexp/internal/config"
	"rexp/internal/editor"
	"rexp/internal/programstatus"
)

func editProjectCommand() *cobra.Command {
	var override string
	cmd := &cobra.Command{
		Use:     "edit <project>",
		Short:   "Edit a saved project configuration",
		Long:    "Open an existing project file in an external editor, wait for it to exit, and validate the saved YAML.\nInvalid configurations can be opened for repair. Changes do not modify a running Rex session.",
		Example: "  rexp edit my-app\n  rexp edit my-app --editor 'code --wait'",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errors.New("missing project name\nUsage: rexp edit <project>\nExample: rexp edit my-app")
			}
			return cobra.ExactArgs(1)(cmd, args)
		},
		RunE: func(cmd *cobra.Command, args []string) error { return editProject(cmd, args[0], override) },
	}
	cmd.Flags().StringVar(&override, "editor", "", "Editor command, including arguments (e.g. 'code --wait')")
	return cmd
}

func editProject(cmd *cobra.Command, name, override string) error {
	// Check only the path before editing: a user must be able to repair bad YAML.
	path, err := config.ProjectFile(name)
	if err != nil {
		return err
	}
	command, err := editor.Resolve(override, cmd.Flags().Changed("editor"))
	if err != nil {
		return err
	}
	programstatus.Editing(cmd.Context(), "Waiting for editor: "+name)
	if err := editor.Open(cmd.Context(), command, path, cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr()); err != nil {
		return fmt.Errorf("%w; configuration remains at %s", err, path)
	}
	programstatus.Working(cmd.Context(), "Validating edited project "+name)
	if _, err := config.LoadProject(name); err != nil {
		return fmt.Errorf("editor closed, but configuration validation failed: %w; your edits were not reverted", err)
	}
	_, err = fmt.Fprintf(cmd.OutOrStdout(), "Project %q is valid. Changes apply when creating a new workspace.\n", name)
	return err
}
