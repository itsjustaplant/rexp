package cli

import (
	"github.com/itsjustaplant/rexp/internal/rex"
	"github.com/spf13/cobra"
)

func bindProjectCommand() *cobra.Command {
	var target projectTarget
	var session string
	var replace, incomplete, forget bool
	cmd := &cobra.Command{
		Use: "bind <project>", Short: "Associate a workspace with an existing Rex session ID",
		Long:    "Record the full session ID for a saved workspace without renaming, selecting, or changing its programs.\nUse this to adopt an already renamed session or select one of several legacy duplicates.\n--forget resets the association without closing any session; inspect Rex before starting a replacement.",
		Example: "  rexp bind my-app --session session:...\n  rexp bind . --session session:...\n  rexp bind --file ./workspace.yaml --session session:...\n  rexp bind my-app --forget",
		Args:    target.args,
		RunE: func(cmd *cobra.Command, args []string) error {
			project, err := target.load(cmd, args)
			if err != nil {
				return err
			}
			client := rex.NewClient()
			if forget {
				if err := client.Forget(cmd.Context(), project); err != nil {
					return err
				}
				cmd.Printf("Forgot the association for project %q; no Rex session was closed.\n", project.Name)
				return nil
			}
			selected, err := client.Bind(cmd.Context(), project, session, replace, incomplete)
			if err != nil {
				return err
			}
			cmd.Printf("Bound project %q to Rex session %s (%s).\n", project.Name, tableText(selected.ID), tableText(selected.Label))
			return nil
		},
	}
	target.bind(cmd)
	cmd.Flags().StringVar(&session, "session", "", "Full ID of the existing Rex session")
	cmd.Flags().BoolVar(&replace, "replace", false, "Replace an existing association without closing the previous session")
	cmd.Flags().BoolVar(&incomplete, "incomplete", false, "Record the selected session as an incomplete startup")
	cmd.Flags().BoolVar(&forget, "forget", false, "Forget the association without closing its session")
	cmd.MarkFlagsOneRequired("session", "forget")
	cmd.MarkFlagsMutuallyExclusive("session", "forget")
	cmd.MarkFlagsMutuallyExclusive("replace", "forget")
	cmd.MarkFlagsMutuallyExclusive("incomplete", "forget")
	return cmd
}
