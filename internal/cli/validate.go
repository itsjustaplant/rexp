package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

func validateProjectCommand() *cobra.Command {
	var target projectTarget
	cmd := &cobra.Command{
		Use:   "validate <project>",
		Short: "Validate a saved project configuration",
		Args:  target.args,
		RunE: func(cmd *cobra.Command, args []string) error {
			project, err := target.load(cmd, args)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Project %q is valid (%d tabs).\n", project.Name, len(project.Tabs))
			return err
		},
	}
	target.bind(cmd)
	return cmd
}
