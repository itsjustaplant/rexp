package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/itsjustaplant/rexp/internal/config"
	"github.com/itsjustaplant/rexp/internal/programstatus"
)

func executeCompletion(t *testing.T, args ...string) (string, error) {
	t.Helper()
	var out, stderr, reports bytes.Buffer
	cmd := NewCommand("test")
	cmd.SetOut(&out)
	cmd.SetErr(&stderr)
	err := ExecuteArguments(cmd, programstatus.New(&reports), args)
	if reports.Len() != 0 || strings.Contains(out.String()+stderr.String(), "\x1b]7501;") {
		t.Fatalf("completion emitted terminal status: %s", &reports)
	}
	return out.String(), err
}

func TestCompletionGenerationHasNoInitializationSideEffects(t *testing.T) {
	for _, shell := range []string{"bash", "zsh", "fish", "powershell"} {
		t.Run(shell, func(t *testing.T) {
			home := filepath.Join(t.TempDir(), "not-created")
			t.Setenv("HOME", home)
			t.Setenv("USERPROFILE", home)
			out, err := executeCompletion(t, "completion", shell)
			if err != nil || !strings.Contains(out, "rexp") || len(out) < 100 {
				t.Fatalf("script generation: %v %s", err, out)
			}
			if _, err := os.Stat(home); !os.IsNotExist(err) {
				t.Fatalf("completion created home/config: %v", err)
			}
		})
	}
}

func TestCompletionProbesUseNamesOnly(t *testing.T) {
	editorHome(t)
	dir := t.TempDir()
	t.Chdir(dir)
	if err := config.Initialize(); err != nil {
		t.Fatal(err)
	}
	projects, _ := config.ProjectsDir()
	for _, name := range []string{"alpha.yaml", "alpine.yaml", "beta.yaml", "invalid name.yaml", "ignore.txt"} {
		if err := os.WriteFile(filepath.Join(projects, name), []byte("hooks: [invalid yaml"), 0600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(dir, config.LocalFilename), nil, 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", t.TempDir()) // Completion must not call Rex or an editor.
	for _, command := range []string{"start", "stop", "status", "edit", "validate", "delete", "search"} {
		out, err := executeCompletion(t, "__complete", command, "al")
		if err != nil || !strings.Contains(out, "alpha\n") || !strings.Contains(out, "alpine\n") || strings.Contains(out, "beta") || strings.Contains(out, "invalid name") {
			t.Fatalf("%s: %q %v", command, out, err)
		}
	}
	out, err := executeCompletion(t, "__complete", "start", ".")
	if err != nil || !strings.Contains(out, ".\n") {
		t.Fatalf("local candidate: %q %v", out, err)
	}
	out, err = executeCompletion(t, "__complete", "delete", ".")
	if err != nil || strings.Contains(out, ".\n") {
		t.Fatalf("delete offered local file: %q %v", out, err)
	}
	for _, args := range [][]string{{"__complete", "new", ""}, {"__complete", "save", ""}, {"__complete", "start", "alpha", ""}, {"__complete", "start", "--file", "file.yaml", ""}} {
		out, err := executeCompletion(t, args...)
		if err != nil || strings.Contains(out, "alpha") {
			t.Fatalf("unexpected candidate for %v: %q %v", args, out, err)
		}
	}
	out, err = executeCompletion(t, "__complete", "start", "--file", "")
	if err != nil || !strings.Contains(out, "yaml") {
		t.Fatalf("file completion %q %v", out, err)
	}
	out, err = executeCompletion(t, "__complete", "start", "--back")
	if err != nil || !strings.Contains(out, "--background") {
		t.Fatalf("flag completion %q %v", out, err)
	}
	out, err = executeCompletion(t, "__complete", "")
	if err != nil || !strings.Contains(out, "completion") || !strings.Contains(out, "search") {
		t.Fatalf("command completion %q %v", out, err)
	}
}

func TestCompletionWithMissingHomeAndInvalidShell(t *testing.T) {
	home := filepath.Join(t.TempDir(), "absent")
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	if _, err := executeCompletion(t, "__completeNoDesc", "start", ""); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"completion"}, {"completion", "unknown"}, {"completion", "bash", "zsh"}} {
		if _, err := executeCompletion(t, args...); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
	if _, err := os.Stat(home); !os.IsNotExist(err) {
		t.Fatalf("completion created files: %v", err)
	}
}
