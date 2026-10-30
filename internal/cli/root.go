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
		Long:          "rexp launches named project workspaces in Rex.\n\nDevelopment scaffold: project configuration and session operations are not implemented yet.",
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

	init := pendingCommand("init <project>", "Create a project configuration (not implemented)", cobra.ExactArgs(1))
	init.Flags().String("editor", "", "Editor command to open the configuration, for example 'code --wait'")
	edit := pendingCommand("edit <project>", "Edit a project configuration (not implemented)", cobra.ExactArgs(1))
	edit.Flags().String("editor", "", "Editor command to open the configuration")
	root.AddCommand(
		init,
		edit,
		pendingCommand("start <project>", "Open a project workspace in Rex (not implemented)", cobra.ExactArgs(1)),
		pendingCommand("list", "List saved project configurations (not implemented)", cobra.NoArgs),
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
