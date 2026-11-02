package cli

import (
	"github.com/itsjustaplant/rexp/internal/config"
	"github.com/spf13/cobra"
)

func listProjectsCommand() *cobra.Command {
	var output projectOutput
	cmd := &cobra.Command{
		Use:     "list",
		Short:   "List saved project configurations",
		Long:    "List saved project names alphabetically, one per line.\nUse --long for metadata or --json for machine-readable details.\nReads global configuration files without contacting Rex or running commands/hooks.",
		Example: "  rexp list\n  rexp list --long\n  rexp list --json",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			names, err := config.ListProjects()
			if err != nil {
				return err
			}
			return output.write(cmd, names, "No projects found. Create one with 'rexp new <project>'.")
		},
	}
	output.bind(cmd)
	return cmd
}
