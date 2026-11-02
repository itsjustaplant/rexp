package cli

import (
	"errors"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"rexp/internal/programstatus"
	"rexp/internal/rex"
)

func stopProjectCommand() *cobra.Command {
	var target projectTarget
	var yes bool
	cmd := &cobra.Command{
		Use:     "stop <project>",
		Short:   "Close a project's running Rex workspace",
		Long:    "Close all tabs, panes, and running programs in the matching Rex session after confirmation.\nThe configuration and project files are kept. Incomplete startup sessions can also be stopped.",
		Example: "  rexp stop my-app\n  rexp stop .\n  rexp stop --file ./workspace.yaml --yes",
		Args:    target.args,
		RunE: func(cmd *cobra.Command, args []string) error {
			project, err := target.load(cmd, args)
			if err != nil {
				return err
			}
			client := rex.NewClient()
			session, err := client.FindStopTarget(cmd.Context(), project)
			if err != nil {
				return err
			}
			if session.ID == "" {
				cmd.Printf("Project %q is not running.\n", project.Name)
				return nil
			}
			current := session.ID == os.Getenv("REX_SESSION")
			if current && !yes {
				return errors.New("this is your current Rex session; run stop from another session or use --yes to close this terminal too")
			}
			if !yes {
				programstatus.Approval(cmd.Context(), "Approve stopping project "+project.Name)
				cmd.PrintErrf("Stop project %q (session %s), closing all its panes and programs? [y/N] ", project.Name, session.ID)
				answer, err := readConfirmation(cmd.Context(), cmd.InOrStdin())
				if err != nil && (!errors.Is(err, io.EOF) || len(answer) == 0) {
					return fmt.Errorf("read stop confirmation: %w; use --yes to stop without prompting", err)
				}
				answer = strings.ToLower(strings.TrimSpace(answer))
				if answer != "y" && answer != "yes" {
					programstatus.Idle(cmd.Context(), "Stop cancelled; workspace kept running")
					cmd.Println("Cancelled; workspace was kept running.")
					return nil
				}
			}
			programstatus.Working(cmd.Context(), "Stopping project "+project.Name)
			if current {
				cmd.Printf("Stopping project %q; this terminal will close.\n", project.Name)
			}
			stopped, err := client.Stop(cmd.Context(), project, session)
			if err != nil {
				return err
			}
			if stopped {
				cmd.Printf("Stopped project %q.\n", project.Name)
			} else {
				cmd.Printf("Project %q is not running.\n", project.Name)
			}
			return nil
		},
	}
	target.bind(cmd)
	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Stop without prompting (required to close your current session)")
	return cmd
}
