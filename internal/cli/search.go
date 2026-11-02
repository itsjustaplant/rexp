package cli

import (
	"errors"
	"strings"

	"github.com/spf13/cobra"
	"rexp/internal/config"
)

func searchProjectsCommand() *cobra.Command {
	var output projectOutput
	cmd := &cobra.Command{
		Use:     "search <query>",
		Short:   "Search saved project names",
		Long:    "Match a case-insensitive substring of saved global project names.\nResults are alphabetical. YAML contents are not searched. No matches is a successful result.\nUse --long or --json for the same metadata formats as list.",
		Example: "  rexp search api\n  rexp search api --long\n  rexp search api --json",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errors.New("missing search query\nUsage: rexp search <query>\nExample: rexp search api")
			}
			if err := cobra.ExactArgs(1)(cmd, args); err != nil {
				return err
			}
			if strings.TrimSpace(args[0]) == "" {
				return errors.New("search query must not be empty")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			names, err := config.ListProjects()
			if err != nil {
				return err
			}
			matches := []string{}
			query := strings.ToLower(args[0])
			for _, name := range names {
				if strings.Contains(strings.ToLower(name), query) {
					matches = append(matches, name)
				}
			}
			return output.write(cmd, matches, "No projects matched.")
		},
	}
	output.bind(cmd)
	return cmd
}
