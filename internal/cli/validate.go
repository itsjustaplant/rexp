package cli

import (
	"errors"
	"fmt"

	"github.com/spf13/cobra"

	"rexp/internal/config"
)

func validateProjectCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "validate <project>",
		Short: "Validate a saved project configuration",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errors.New("missing project name\nUsage: rexp validate <project>\nExample: rexp validate my-app")
			}
			return cobra.ExactArgs(1)(cmd, args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			project, err := config.LoadProject(args[0])
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Project %q is valid (%d tabs).\n", project.Name, len(project.Tabs))
			return err
		},
	}
}
