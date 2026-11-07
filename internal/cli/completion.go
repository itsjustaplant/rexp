package cli

import (
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/itsjustaplant/rexp/internal/config"
)

func completionCommand() *cobra.Command {
	return &cobra.Command{
		Use:       "completion <shell>",
		Short:     "Generate a shell completion script",
		Long:      "Generate a completion script for bash, zsh, fish, or powershell.\nLoad the script in your shell to complete commands, flags, and saved project names.\nThis command does not install files or change your shell settings.",
		Example:   "  source <(rexp completion zsh)\n  source <(rexp completion bash)\n  rexp completion fish | source\n  rexp completion powershell | Out-String | Invoke-Expression",
		Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
		RunE: func(cmd *cobra.Command, args []string) error {
			root := cmd.Root()
			out := cmd.OutOrStdout()
			switch args[0] {
			case "bash":
				return root.GenBashCompletionV2(out, true)
			case "zsh":
				return root.GenZshCompletion(out)
			case "fish":
				return root.GenFishCompletion(out, true)
			default:
				return root.GenPowerShellCompletionWithDesc(out)
			}
		},
	}
}

func isCompletionInvocation(args []string) bool {
	if len(args) == 0 {
		return false
	}
	return completionName(args[0])
}

func completionName(name string) bool {
	return name == "completion" || name == "__complete" || name == "__completeNoDesc"
}

// Completion reads names only, including names with currently invalid YAML.
// Local '.' is offered only when its regular file exists; no YAML is loaded.
func completeProjects(local bool) func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
	return func(cmd *cobra.Command, args []string, prefix string) ([]string, cobra.ShellCompDirective) {
		if len(args) > 0 || cmd.Flags().Changed("file") {
			return nil, cobra.ShellCompDirectiveNoFileComp
		}
		names, err := config.ListProjects()
		if err != nil {
			return nil, cobra.ShellCompDirectiveError | cobra.ShellCompDirectiveNoFileComp
		}
		result := []string{}
		for _, name := range names {
			if strings.HasPrefix(name, prefix) {
				result = append(result, name)
			}
		}
		if local && strings.HasPrefix(".", prefix) {
			if info, err := os.Lstat(config.LocalFilename); err == nil && info.Mode().IsRegular() {
				result = append(result, ".")
			}
		}
		return result, cobra.ShellCompDirectiveNoFileComp
	}
}

func configureCompletions(root *cobra.Command) {
	root.ValidArgsFunction = cobra.NoFileCompletions
	for _, cmd := range root.Commands() {
		switch cmd.Name() {
		case "start", "bind", "switch", "use", "stop", "status", "edit", "validate":
			cmd.ValidArgsFunction = completeProjects(true)
		case "delete", "search":
			cmd.ValidArgsFunction = completeProjects(false)
		case "completion": // Its static shell list belongs to Cobra.
		default:
			cmd.ValidArgsFunction = cobra.NoFileCompletions
		}
		for _, flag := range []string{"editor", "client", "session"} {
			if cmd.Flags().Lookup(flag) != nil {
				_ = cmd.RegisterFlagCompletionFunc(flag, cobra.NoFileCompletions)
			}
		}
	}
}
