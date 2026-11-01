package cli

import "github.com/spf13/cobra"

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
		listProjectsCommand(),
		validateProjectCommand(),
	)
	return root
}
