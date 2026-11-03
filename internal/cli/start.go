package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/itsjustaplant/rexp/internal/rex"
)

func startProjectCommand() *cobra.Command {
	return launchProjectCommand("start", true)
}

func useProjectCommand() *cobra.Command {
	return launchProjectCommand("use", false)
}

func launchProjectCommand(name string, allowBackground bool) *cobra.Command {
	var options rex.Options
	var target projectTarget
	cmd := &cobra.Command{
		Use:   name + " <project>",
		Short: "Create or select a project's Rex workspace",
		Args:  target.args,
		RunE: func(cmd *cobra.Command, args []string) error {
			project, err := target.load(cmd, args)
			if err != nil {
				return err
			}
			options.Stdout, options.Stderr = cmd.OutOrStdout(), cmd.ErrOrStderr()
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
	cmd.Flags().StringVar(&options.Client, "client", "", "Rex client to select the session in")
	if allowBackground {
		cmd.Flags().BoolVar(&options.Background, "background", false, "Create or reuse the session without switching the Rex app")
		cmd.MarkFlagsMutuallyExclusive("background", "client")
	} else {
		cmd.Short = "Start a workspace if needed and switch the Rex app to it"
		cmd.Long = "Switch to a saved project's workspace, creating it first if it is not running.\nExisting sessions are reused without rerunning commands or hooks.\nA new workspace is selected after startup completes, without waiting for its services to be healthy."
		cmd.Example = "  rexp use my-app\n  rexp use .\n  rexp use --file ./workspace.yaml --client macbook"
	}
	target.bind(cmd)
	return cmd
}
