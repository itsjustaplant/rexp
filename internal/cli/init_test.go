package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/itsjustaplant/rexp/internal/agentskill"
)

func initHome(t *testing.T) string {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("SHELL", "/bin/zsh")
	t.Setenv("ZDOTDIR", "")
	t.Setenv("XDG_CONFIG_HOME", "")
	return home
}

func runSetup(t *testing.T, answers string, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	cmd := NewCommand("test")
	cmd.SetIn(strings.NewReader(answers))
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

func TestInitDecliningSetupLeavesUserFilesAlone(t *testing.T) {
	home := initHome(t)
	t.Setenv("PATH", t.TempDir())
	for _, command := range []string{"init", "intro"} {
		out, err := runSetup(t, "n\nn\nn\nn\n", command)
		if err != nil || !strings.Contains(out, "You're ready") {
			t.Fatalf("declined setup: %q %v", out, err)
		}
	}
	for _, path := range []string{".agents", ".claude", ".zshrc", filepath.Join(".rexp", "projects")} {
		if _, err := os.Stat(filepath.Join(home, path)); !os.IsNotExist(err) {
			t.Fatalf("declined setup wrote %s: %v", path, err)
		}
	}
	if _, err := os.Stat(onboardingMarker(home)); err != nil {
		t.Fatal("completed setup did not record first-run state")
	}
}

func TestInitInstallsSelectedSkillsAndCompletionWithoutDuplicates(t *testing.T) {
	home := initHome(t)
	path := filepath.Join(home, ".zshrc")
	if err := os.WriteFile(path, []byte("# my settings\n"), 0600); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		out, err := runSetup(t, "y\nn\ny\n", "init", "--shell", "zsh", "--no-demo")
		if err != nil || !strings.Contains(out, "Completion is ready") {
			t.Fatalf("setup: %q %v", out, err)
		}
	}
	data, err := os.ReadFile(filepath.Join(home, ".agents", "skills", "rexp", "SKILL.md"))
	if err != nil || string(data) != agentskill.Content {
		t.Fatal("Codex skill missing")
	}
	if _, err := os.Stat(filepath.Join(home, ".claude")); !os.IsNotExist(err) {
		t.Fatal("installed declined Claude skill")
	}
	data, err = os.ReadFile(path)
	if err != nil || !strings.HasPrefix(string(data), "# my settings\n") || strings.Count(string(data), "# >>> rexp completion >>>") != 1 {
		t.Fatalf("damaged or duplicated shell setup: %q %v", data, err)
	}
}

func TestInitPreservesDifferingSkillUnlessReplacementApproved(t *testing.T) {
	home := initHome(t)
	path, err := agentskill.Install(home, "codex", false)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("custom skill"), 0600); err != nil {
		t.Fatal(err)
	}
	if _, err := runSetup(t, "y\nn\nn\nn\n", "init", "--no-demo"); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	if string(data) != "custom skill" {
		t.Fatal("overwrote skill without approval")
	}
	if _, err := runSetup(t, "y\ny\nn\nn\n", "init", "--no-demo"); err != nil {
		t.Fatal(err)
	}
	data, _ = os.ReadFile(path)
	if string(data) != agentskill.Content {
		t.Fatal("approved skill replacement did not run")
	}
}

func TestInitEOFInvalidArgsAndCancellationDoNotMarkCompleted(t *testing.T) {
	home := initHome(t)
	for _, args := range [][]string{{"init"}, {"init", "extra"}, {"init", "--shell", "invalid"}} {
		if _, err := runSetup(t, "", args...); err == nil {
			t.Fatalf("accepted %v with no answers", args)
		}
	}
	cmd := NewCommand("test")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"init"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("ignored cancellation")
	}
	if _, err := os.Stat(onboardingMarker(home)); !os.IsNotExist(err) {
		t.Fatal("marked interrupted setup complete")
	}
}

func TestBareCommandWithRedirectedStreamsDoesNotPrompt(t *testing.T) {
	home := initHome(t)
	out, err := runSetup(t, "", []string{}...)
	if err != nil || strings.Contains(out, "Install the Codex") || !strings.Contains(out, "Available Commands") {
		t.Fatalf("noninteractive root: %q %v", out, err)
	}
	if _, err := os.Stat(onboardingMarker(home)); !os.IsNotExist(err) {
		t.Fatal("noninteractive command marked onboarding complete")
	}
}

func TestInitLaunchesTwoPaneDemoWithoutSavingOrRerunningIt(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX fake Rex")
	}
	home := initHome(t)
	bin := t.TempDir()
	calls := filepath.Join(bin, "calls")
	t.Setenv("PATH", bin)
	t.Setenv("REXP_INIT_CALLS", calls)
	t.Setenv("REXP_INIT_READY", filepath.Join(bin, "ready"))
	script := "#!/bin/sh\n" + rexIdentityStub + `
printf '%s\n' "$*" >> "$REXP_INIT_CALLS"
if [ "$1" = -C ]; then shift 2; fi
case "$1" in
  ls)
    if [ -f "$REXP_INIT_READY" ]; then
      printf '%s\n' '{"sessions":[{"session_id":"session:demo","label":"rexp:rexp-intro"}]}'
    else
      printf '%s\n' '{"sessions":[]}'
    fi ;;
  new) printf '%s\n' '{"session_id":"session:demo","initial_windows":[{"block_ids":["block:clock"]}]}' ;;
  split) printf '%s\n' '{"block_ids":["block:logo"]}' ;;
  block|focus) ;;
  session) printf ready > "$REXP_INIT_READY" ;;
  do) [ -f "$REXP_INIT_READY" ] && [ "$2" = session.select ] || exit 98 ;;
  *) exit 98 ;;
esac
`
	if err := os.WriteFile(filepath.Join(bin, "rex"), []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	for range 2 {
		out, err := runSetup(t, "n\nn\nn\ny\n", "init", "--client", "macbook")
		if err != nil || !strings.Contains(out, "Demo ready in session session:demo") {
			t.Fatalf("demo: %q %v", out, err)
		}
	}
	data, _ := os.ReadFile(calls)
	log := string(data)
	if strings.Count(log, "\nnew ") != 1 || strings.Count(log, "\nsplit ") != 1 || !strings.Contains(log, "__demo clock") || !strings.Contains(log, "__demo logo") || strings.Count(log, "-C macbook do session.select session_id=session:demo") != 2 {
		t.Fatalf("wrong demo lifecycle:\n%s", log)
	}
	if _, err := os.Stat(filepath.Join(home, ".rexp", "projects")); !os.IsNotExist(err) {
		t.Fatal("demo changed saved project configurations")
	}
}
