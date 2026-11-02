package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/itsjustaplant/rexp/internal/config"
	"github.com/itsjustaplant/rexp/internal/editor"
	"github.com/itsjustaplant/rexp/internal/programstatus"
)

func editProjectCommand() *cobra.Command {
	var override string
	var target projectTarget
	cmd := &cobra.Command{
		Use:     "edit <project>",
		Short:   "Edit a saved project configuration",
		Long:    "Open an existing project file in an external editor, wait for it to exit, and validate the saved YAML.\nInvalid configurations can be opened for repair. Changes do not modify a running Rex session.",
		Example: "  rexp edit my-app\n  rexp edit my-app --editor 'code --wait'",
		Args:    target.args,
		RunE: func(cmd *cobra.Command, args []string) error {
			path, local, err := target.path(cmd, args)
			if err != nil {
				return err
			}
			name := path
			if !local {
				name = args[0]
			}
			return editProjectFile(cmd, path, name, override, local)
		},
	}
	cmd.Flags().StringVar(&override, "editor", "", "Editor command, including arguments (e.g. 'code --wait')")
	target.bind(cmd)
	return cmd
}

func editProject(cmd *cobra.Command, name, override string) error {
	// Check only the path before editing: a user must be able to repair bad YAML.
	path, err := config.ProjectFile(name)
	if err != nil {
		return err
	}
	return editProjectFile(cmd, path, name, override, false)
}

func editProjectFile(cmd *cobra.Command, path, name, override string, local bool) error {
	command, err := editor.Resolve(override, cmd.Flags().Changed("editor"))
	if err != nil {
		return err
	}
	programstatus.Editing(cmd.Context(), "Waiting for editor: "+name)
	if err := editor.Open(cmd.Context(), command, path, cmd.InOrStdin(), cmd.OutOrStdout(), cmd.ErrOrStderr()); err != nil {
		return fmt.Errorf("%w; configuration remains at %s", err, path)
	}
	programstatus.Working(cmd.Context(), "Validating edited project "+name)
	if local {
		_, err = config.LoadFile(path)
	} else {
		_, err = config.LoadProject(name)
	}
	if err != nil {
		return fmt.Errorf("editor closed, but configuration validation failed: %w; your edits were not reverted", err)
	}
	_, err = fmt.Fprintf(cmd.OutOrStdout(), "Project %q is valid. Changes apply when creating a new workspace.\n", name)
	return err
}
