package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"rexp/internal/rex"
)

func startProjectCommand() *cobra.Command {
	var options rex.Options
	var target projectTarget
	cmd := &cobra.Command{
		Use:   "start <project>",
		Short: "Create or select a project's Rex workspace",
		Args:  target.args,
		RunE: func(cmd *cobra.Command, args []string) error {
			project, err := target.load(cmd, args)
			if err != nil {
				return err
			}
			options.Stdout, options.Stderr = cmd.OutOrStdout(), cmd.ErrOrStderr()
			result, err := rex.NewClient().Start(cmd.Context(), project, options)
			if err != nil {
				return err
			}
			action := "Started"
			if result.Reused {
				action = "Reused"
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "%s project %q in Rex session %s.\n", action, project.Name, result.SessionID)
			return err
		},
	}
	cmd.Flags().BoolVar(&options.Background, "background", false, "Create or reuse the session without switching the Rex app")
	cmd.Flags().StringVar(&options.Client, "client", "", "Rex client to select the session in")
	cmd.MarkFlagsMutuallyExclusive("background", "client")
	target.bind(cmd)
	return cmd
}
