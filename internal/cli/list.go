package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"rexp/internal/config"
)

func listProjectsCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "list",
		Short:   "List saved project configurations",
		Long:    "List saved project names alphabetically, one per line.\nReads filenames in ~/.rexp/projects without parsing YAML or connecting to Rex.",
		Example: "  rexp list",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			names, err := config.ListProjects()
			if err != nil {
				return err
			}
			if len(names) == 0 {
				_, err := fmt.Fprintln(cmd.OutOrStdout(), "No projects found. Create one with 'rexp new <project>'.")
				return err
			}
			for _, name := range names {
				if _, err := fmt.Fprintln(cmd.OutOrStdout(), name); err != nil {
					return err
				}
			}
			return nil
		},
	}
}
