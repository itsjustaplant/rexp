package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"rexp/internal/config"
)

// A positional name always means a global config, except explicit '.' which
// chooses .rexp.yaml. File selection never silently overrides a global name.
type projectTarget struct{ file string }

func (target *projectTarget) bind(cmd *cobra.Command) {
	cmd.Flags().StringVar(&target.file, "file", "", "Read a configuration file (relative roots resolve beside it)")
}

func (target *projectTarget) args(cmd *cobra.Command, args []string) error {
	if cmd.Flags().Changed("file") {
		if len(args) != 0 {
			return fmt.Errorf("choose either a project name or --file, not both")
		}
		if target.file == "" {
			return fmt.Errorf("--file requires a configuration path")
		}
		return nil
	}
	if len(args) == 0 {
		return fmt.Errorf("missing project name\nUsage: rexp %s <project> or rexp %s --file <path>\nExample: rexp %s my-app", cmd.Name(), cmd.Name(), cmd.Name())
	}
	return cobra.ExactArgs(1)(cmd, args)
}

func (target *projectTarget) path(cmd *cobra.Command, args []string) (string, bool, error) {
	if cmd.Flags().Changed("file") {
		path, err := config.LocalFile(target.file)
		return path, true, err
	}
	if args[0] == "." {
		path, err := config.LocalFile(config.LocalFilename)
		return path, true, err
	}
	path, err := config.ProjectFile(args[0])
	return path, false, err
}

func (target *projectTarget) load(cmd *cobra.Command, args []string) (*config.Project, error) {
	path, local, err := target.path(cmd, args)
	if err != nil {
		return nil, err
	}
	if local {
		return config.LoadFile(path)
	}
	return config.LoadProject(args[0])
}
