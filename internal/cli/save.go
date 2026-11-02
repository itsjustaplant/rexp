package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/itsjustaplant/rexp/internal/config"
	"github.com/itsjustaplant/rexp/internal/rex"
)

func saveProjectCommand() *cobra.Command {
	var session string
	cmd := &cobra.Command{
		Use:     "save <project>",
		Short:   "Save the current Rex session as a project configuration",
		Long:    "Save the session identified by REX_SESSION, or --session, into ~/.rexp/projects/<project>.yaml.\nCaptures terminal tabs, layout, labels, focus, and directories. Existing files are never overwritten.\nRex does not expose complete command arguments, so saved panes open shells until commands are added manually.",
		Example: "  rexp save my-app\n  rexp save my-app --session session:...",
		Args: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return errors.New("missing project name\nUsage: rexp save <project>\nExample: rexp save my-app")
			}
			if err := cobra.ExactArgs(1)(cmd, args); err != nil {
				return err
			}
			return config.ValidateProjectName(args[0])
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			projects, err := config.ProjectsDir()
			if err != nil {
				return err
			}
			path := filepath.Join(projects, args[0]+".yaml")
			if _, err := os.Lstat(path); err == nil {
				return fmt.Errorf("project %q already exists at %s; existing configuration was preserved", args[0], path)
			} else if !errors.Is(err, os.ErrNotExist) {
				return fmt.Errorf("inspect destination: %w", err)
			}
			if !cmd.Flags().Changed("session") {
				session = os.Getenv("REX_SESSION")
			}
			if session == "" {
				return errors.New("no current Rex session; run save inside Rex or pass --session <session>")
			}
			root, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("find project directory: %w", err)
			}
			result, err := rex.NewClient().Export(cmd.Context(), session, args[0], root)
			if err != nil {
				return fmt.Errorf("save Rex session: %w", err)
			}
			path, err = config.SaveProject(result.Project)
			if err != nil {
				return err
			}
			if _, err := fmt.Fprintf(cmd.OutOrStdout(), "Saved %s\n", path); err != nil {
				return err
			}
			return nil
		},
	}
	cmd.Flags().StringVar(&session, "session", "", "Rex session to save (defaults to REX_SESSION)")
	return cmd
}
