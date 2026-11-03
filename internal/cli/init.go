package cli

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/itsjustaplant/rexp/internal/agentskill"
	"github.com/itsjustaplant/rexp/internal/demo"
	"github.com/itsjustaplant/rexp/internal/programstatus"
	"github.com/itsjustaplant/rexp/internal/rex"
	"github.com/itsjustaplant/rexp/internal/shellsetup"
	"github.com/spf13/cobra"
)

type initOptions struct {
	shell, client string
	noDemo        bool
}

func initCommand() *cobra.Command {
	var options initOptions
	cmd := &cobra.Command{
		Use: "init", Aliases: []string{"intro"}, Short: "Set up agent skills, shell completion, and a welcome workspace",
		Long: "Offer agent skill installation and shell completion, then open a two-pane clock and rexp demo.\nEvery installation is optional. Existing skills and shell settings are preserved.\nRun this command again to replay setup; a successful run suppresses automatic first-run setup.",
		Args: func(cmd *cobra.Command, args []string) error {
			if err := cobra.NoArgs(cmd, args); err != nil {
				return err
			}
			switch options.shell {
			case "", "zsh", "bash", "fish":
				return nil
			default:
				return errors.New("--shell must be zsh, bash, or fish")
			}
		},
		RunE: func(cmd *cobra.Command, _ []string) error { return runInit(cmd, options) },
	}
	cmd.Flags().StringVar(&options.shell, "shell", "", "Shell to configure: zsh, bash, or fish (defaults to SHELL)")
	cmd.Flags().StringVar(&options.client, "client", "", "Rex app client to show the demo in")
	cmd.Flags().BoolVar(&options.noDemo, "no-demo", false, "Set up skills and completion without offering the demo")
	_ = cmd.RegisterFlagCompletionFunc("shell", func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
		return []string{"zsh", "bash", "fish"}, cobra.ShellCompDirectiveNoFileComp
	})
	return cmd
}

func onboardingMarker(home string) string {
	return filepath.Join(home, ".rexp", "state", "onboarding-done")
}

func firstRun(cmd *cobra.Command) (bool, error) {
	in, inOK := cmd.InOrStdin().(*os.File)
	out, outOK := cmd.OutOrStdout().(*os.File)
	if !inOK || !outOK || !programstatus.IsTerminal(in) || !programstatus.IsTerminal(out) || os.Getenv("TERM") == "dumb" {
		return false, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return false, err
	}
	_, err = os.Lstat(onboardingMarker(home))
	if errors.Is(err, os.ErrNotExist) {
		return true, nil
	}
	return false, err
}

func runInit(cmd *cobra.Command, options initOptions) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	programstatus.Working(cmd.Context(), "Setting up rexp")
	cmd.Println("Welcome to rexp. Let's make this workspace yours.")
	cmd.Println("Skills and shell completion are optional. Your existing files are kept.")
	input := bufio.NewReader(cmd.InOrStdin())
	ask := func(question string, defaultYes bool) (bool, error) {
		for {
			programstatus.Approval(cmd.Context(), question)
			hint := "[y/N]"
			if defaultYes {
				hint = "[Y/n]"
			}
			cmd.Printf("\n%s %s ", question, hint)
			answer, err := readConfirmation(cmd.Context(), input)
			if err != nil && (!errors.Is(err, io.EOF) || len(answer) == 0) {
				return false, fmt.Errorf("read setup answer: %w; run rexp init to try again", err)
			}
			programstatus.Working(cmd.Context(), "Setting up rexp")
			switch strings.ToLower(strings.TrimSpace(answer)) {
			case "":
				return defaultYes, nil
			case "y", "yes":
				return true, nil
			case "n", "no":
				return false, nil
			default:
				cmd.Println("Please enter y or n.")
			}
		}
	}
	for _, agent := range []struct{ name, dir, title string }{
		{"codex", ".agents", "Codex"}, {"claude", ".claude", "Claude Code"},
	} {
		path := filepath.Join(home, agent.dir, "skills", "rexp", "SKILL.md")
		yes, err := ask(fmt.Sprintf("Install the %s skill at %s?", agent.title, tableText(path)), false)
		if err != nil {
			return err
		}
		if !yes {
			continue
		}
		_, err = agentskill.Install(home, agent.name, false)
		if errors.Is(err, agentskill.ErrConflict) {
			replace, askErr := ask("This skill differs from the bundled version. Replace it?", false)
			if askErr != nil {
				return askErr
			}
			if !replace {
				cmd.Println("Kept the existing skill.")
				continue
			}
			_, err = agentskill.Install(home, agent.name, true)
		}
		if err != nil {
			return err
		}
		cmd.Printf("Skill ready at %s\n", tableText(path))
	}
	shell := options.shell
	if shell == "" {
		shell = filepath.Base(os.Getenv("SHELL"))
	}
	files, err := shellsetup.Plan(home, shell)
	if err != nil {
		if shell == "zsh" || shell == "bash" || shell == "fish" {
			return err
		}
		cmd.Println("Choose --shell zsh, bash, or fish with rexp init to set up completion.")
	} else {
		cmd.Printf("\nCompletion for %s will use:\n", shell)
		for _, file := range files {
			cmd.Printf("  %s\n", tableText(file.Path))
		}
		yes, err := ask("Enable tab completion in these files?", false)
		if err != nil {
			return err
		}
		if yes {
			if err := shellsetup.Install(files); err != nil {
				return err
			}
			cmd.Println("Completion is ready. Open a new shell with rexp on PATH, then try rexp <Tab>.")
		}
	}
	if !options.noDemo {
		yes, err := ask("Open the clock and dinosaur demo in Rex?", true)
		if err != nil {
			return err
		}
		if yes {
			binary, err := os.Executable()
			if err != nil {
				return err
			}
			result, err := rex.NewClient().Start(cmd.Context(), demo.Project(binary, home), rex.Options{
				Client: options.client, Stdout: cmd.OutOrStdout(), Stderr: cmd.ErrOrStderr(),
			})
			if err != nil {
				return err
			}
			cmd.Printf("Demo ready in session %s. Close it when finished with: rex kill %s\n", tableText(result.SessionID), tableText(result.SessionID))
		}
	}
	if err := cmd.Context().Err(); err != nil {
		return err
	}
	marker := onboardingMarker(home)
	if err := os.MkdirAll(filepath.Dir(marker), 0700); err != nil {
		return err
	}
	file, err := os.OpenFile(marker, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0600)
	if err != nil && !errors.Is(err, os.ErrExist) {
		return err
	}
	if err == nil {
		_, writeErr := file.WriteString("1\n")
		if err := errors.Join(writeErr, file.Close()); err != nil {
			return err
		}
	}
	cmd.Println("\nYou're ready. Run rexp init any time to revisit setup.")
	return nil
}
