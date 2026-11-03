package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/itsjustaplant/rexp/internal/agentskill"
	"github.com/spf13/cobra"
)

func skillCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use: "skill", Short: "Show or install the shared Codex and Claude skill",
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error { return cmd.Help() },
	}
	show := &cobra.Command{
		Use: "show", Short: "Print the bundled SKILL.md", Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, _ []string) error {
			_, err := io.WriteString(cmd.OutOrStdout(), agentskill.Content)
			return err
		},
	}
	var force bool
	install := &cobra.Command{
		Use: "install <codex|claude>", Short: "Install the skill for the current user",
		Long:      "Install the bundled skill to ~/.agents/skills/rexp/SKILL.md for Codex\nor ~/.claude/skills/rexp/SKILL.md for Claude Code. Identical installs are unchanged.\nExisting different files are preserved unless --force is supplied.",
		Example:   "  rexp skill install codex\n  rexp skill install claude\n  rexp skill install codex --force",
		Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		ValidArgs: []string{"codex", "claude"},
		RunE: func(cmd *cobra.Command, args []string) error {
			home, err := os.UserHomeDir()
			if err != nil {
				return fmt.Errorf("find user home directory: %w", err)
			}
			path, err := agentskill.Install(home, args[0], force)
			if err != nil {
				return err
			}
			_, err = fmt.Fprintf(cmd.OutOrStdout(), "Skill ready at %s\n", path)
			return err
		},
	}
	install.Flags().BoolVar(&force, "force", false, "Replace an existing skill file")
	cmd.AddCommand(show, install)
	return cmd
}
