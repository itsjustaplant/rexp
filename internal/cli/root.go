package cli

import (
	"github.com/spf13/cobra"
	"rexp/internal/programstatus"
)

// NewCommand constructs an independent command tree for one invocation.
func NewCommand(version string) *cobra.Command {
	root := &cobra.Command{
		Use:           "rexp",
		Short:         "Project workspaces for Rex",
		Long:          "rexp manages named project workspaces for Rex.\n\nCreate configurations with new, update them with edit, check them with validate, and launch layouts with start.",
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
		Args:          cobra.NoArgs,
		PersistentPreRunE: func(cmd *cobra.Command, _ []string) error {
			if err := cmd.Context().Err(); err != nil {
				return err
			}
			if cmd.Parent() != nil && cmd.RunE != nil {
				programstatus.Working(cmd.Context(), "Running "+cmd.CommandPath())
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, _ []string) error {
			return cmd.Help()
		},
		CompletionOptions: cobra.CompletionOptions{DisableDefaultCmd: true},
	}
	root.SetVersionTemplate("rexp {{.Version}}\n")

	root.AddCommand(
		newProjectCommand(),
		saveProjectCommand(),
		deleteProjectCommand(),
		editProjectCommand(),
		startProjectCommand(),
		stopProjectCommand(),
		listProjectsCommand(),
		validateProjectCommand(),
	)
	return root
}
