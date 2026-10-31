package cli

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"rexp/internal/config"
	"rexp/internal/rex"
)

func startProjectCommand() *cobra.Command {
	var options rex.Options
	cmd := &cobra.Command{
		Use:   "start <project>",
		Short: "Create or select a project's Rex workspace",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errors.New("missing project name\nUsage: rexp start <project>\nExample: rexp start my-app")
			}
			return cobra.ExactArgs(1)(cmd, args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			project, err := config.LoadProject(args[0])
			if err != nil {
				return err
			}
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
	return cmd
}
