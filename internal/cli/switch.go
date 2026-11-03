package cli

import (
	"errors"
	"fmt"
	"strings"

	"github.com/itsjustaplant/rexp/internal/rex"
	"github.com/spf13/cobra"
)

func switchSessionCommand() *cobra.Command {
	var target projectTarget
	var appClient string
	cmd := &cobra.Command{
		Use:     "switch <session>",
		Short:   "Switch the Rex app to an existing session",
		Long:    "Select an existing Rex session by full ID, exact label, or global project name.\nA project name also matches rexp:<name>; ambiguous matches require a full session ID.\nUse . or --file for a repository-local workspace. Never creates sessions or runs hooks.",
		Example: "  rexp switch my-app\n  rexp switch rexp:my-app\n  rexp switch session:...\n  rexp switch .\n  rexp switch --file ./workspace.yaml --client macbook",
		Args: func(cmd *cobra.Command, args []string) error {
			if err := target.args(cmd, args); err != nil {
				if len(args) == 0 && !cmd.Flags().Changed("file") {
					return errors.New("missing session name or ID\nUsage: rexp switch <session> or rexp switch --file <path>\nExample: rexp switch my-app")
				}
				return err
			}
			if len(args) == 1 && strings.TrimSpace(args[0]) == "" {
				return errors.New("a session name or ID is required")
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			var name string
			if cmd.Flags().Changed("file") || args[0] == "." {
				project, err := target.load(cmd, args)
				if err != nil {
					return err
				}
				name = rex.SessionLabel(project)
			} else {
				name = args[0]
			}
			session, err := rex.NewClient().Switch(cmd.Context(), name, appClient)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Switched to Rex session %s (%s).\n", tableText(session.Label), tableText(session.ID))
			return err
		},
	}
	target.bind(cmd)
	cmd.Flags().StringVar(&appClient, "client", "", "Rex client to select the session in (otherwise uses Rex's default)")
	return cmd
}
