package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"rexp/internal/rex"
)

func statusProjectCommand() *cobra.Command {
	var target projectTarget
	var jsonOutput bool
	cmd := &cobra.Command{
		Use:     "status <project>",
		Short:   "Show a project's Rex workspace status",
		Long:    "Inspect whether a project's session is running, missing, incomplete, or ambiguous.\nReads Rex session labels without changing sessions or running hooks.\nThis reports workspace existence, not application health. All reported states exit successfully.",
		Example: "  rexp status my-app\n  rexp status .\n  rexp status --file ./workspace.yaml --json",
		Args:    target.args,
		RunE: func(cmd *cobra.Command, args []string) error {
			project, err := target.load(cmd, args)
			if err != nil {
				return err
			}
			result, err := rex.NewClient().Status(cmd.Context(), project)
			if err != nil {
				return err
			}
			if jsonOutput {
				return json.NewEncoder(cmd.OutOrStdout()).Encode(result)
			}
			out := cmd.OutOrStdout()
			switch result.Status {
			case rex.WorkspaceRunning:
				_, err = fmt.Fprintf(out, "Project %q is running (session %s).\n", project.Name, result.Sessions[0].ID)
			case rex.WorkspaceMissing:
				_, err = fmt.Fprintf(out, "Project %q is not running.\n", project.Name)
			case rex.WorkspaceIncomplete:
				_, err = fmt.Fprintf(out, "Project %q has an incomplete startup (session %s).\n", project.Name, result.Sessions[0].ID)
			case rex.WorkspaceAmbiguous:
				if _, err = fmt.Fprintf(out, "Project %q has an ambiguous workspace: %s.\n", project.Name, result.Reason); err != nil {
					return err
				}
				for _, session := range result.Sessions {
					if _, err = fmt.Fprintf(out, "  %s (%s)\n", tableText(session.ID), tableText(session.Label)); err != nil {
						return err
					}
				}
			}
			return err
		},
	}
	target.bind(cmd)
	cmd.Flags().BoolVar(&jsonOutput, "json", false, "Output workspace status and matching session IDs as JSON")
	return cmd
}
