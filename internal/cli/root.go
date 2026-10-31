package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

// NewCommand constructs an independent command tree for one invocation.
func NewCommand(version string) *cobra.Command {
	root := &cobra.Command{
		Use:           "rexp",
		Short:         "Project workspaces for Rex",
		Long:          "rexp manages named project workspaces for Rex.\n\nCreate a starter configuration with new. Editing and session operations are not implemented yet.",
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

	edit := pendingCommand("edit <project>", "Edit a project configuration (not implemented)", cobra.ExactArgs(1))
	edit.Flags().String("editor", "", "Editor command to open the configuration")
	root.AddCommand(
		newProjectCommand(),
		deleteProjectCommand(),
		edit,
		pendingCommand("start <project>", "Open a project workspace in Rex (not implemented)", cobra.ExactArgs(1)),
		listProjectsCommand(),
	)
	return root
}

func pendingCommand(use, short string, args cobra.PositionalArgs) *cobra.Command {
	return &cobra.Command{
		Use:   use,
		Short: short,
		Args:  args,
		RunE: func(cmd *cobra.Command, _ []string) error {
			return fmt.Errorf("%s is not implemented yet", cmd.CommandPath())
		},
	}
}
