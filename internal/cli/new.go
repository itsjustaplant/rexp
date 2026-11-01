package cli

import (
	"errors"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"rexp/internal/config"
)

func newProjectCommand() *cobra.Command {
	var edit bool
	var override string
	cmd := &cobra.Command{
		Use:     "new <project>",
		Short:   "Create a project configuration",
		Long:    "Create a starter YAML configuration in ~/.rexp/projects using the current directory as its root.\nExisting configurations are never overwritten. Use --edit to open the new file in an editor.",
		Example: "  rexp new my-app\n  rexp new my-app --edit",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errors.New("missing project name\nUsage: rexp new <project> [--edit]\nExample: rexp new my-app")
			}
			if err := cobra.ExactArgs(1)(cmd, args); err != nil {
				return err
			}
			if cmd.Flags().Changed("editor") && !edit {
				return errors.New("--editor requires --edit")
			}
			return config.ValidateProjectName(args[0])
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			root, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("find project directory: %w", err)
			}
			path, err := config.CreateProject(args[0], root)
			if err != nil {
				return err
			}
			cmd.Printf("Created %s\n", path)
			if edit {
				return editProject(cmd, args[0], override)
			}
			return nil
		},
	}
	cmd.Flags().BoolVarP(&edit, "edit", "e", false, "Open the new configuration in an editor and validate after it exits")
	cmd.Flags().StringVar(&override, "editor", "", "Editor command to use with --edit (e.g. 'code --wait')")
	return cmd
}
