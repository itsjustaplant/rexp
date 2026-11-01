package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewCommandCreatesConfig(t *testing.T) {
	for _, flag := range []string{"", "--edit", "-e"} {
		t.Run("flag="+flag, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("USERPROFILE", home)
			t.Setenv("VISUAL", "")
			t.Setenv("EDITOR", fakeEditor(t, "keep"))
			var stdout, stderr bytes.Buffer
			cmd := NewCommand("test")
			cmd.SetOut(&stdout)
			cmd.SetErr(&stderr)
			args := []string{"new", "demo"}
			if flag != "" {
				args = append(args, flag)
			}
			cmd.SetArgs(args)
			if err := cmd.Execute(); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(home, ".rexp", "projects", "demo.yaml")
			content, err := os.ReadFile(path)
			if err != nil || !strings.Contains(string(content), `name: "demo"`) {
				t.Fatalf("configuration missing: %s, error %v", content, err)
			}
			if !strings.Contains(stdout.String(), path) {
				t.Fatalf("missing created path in output: %q", stdout.String())
			}
			if stderr.Len() != 0 {
				t.Fatalf("unexpected warning: %q", stderr.String())
			}
		})
	}
}

func TestNewCommandRejectsInvalidArguments(t *testing.T) {
	for _, args := range [][]string{{"new"}, {"new", "one", "two"}, {"new", "../escape"}, {"new", "demo", "--unknown"}, {"init", "demo"}} {
		home := t.TempDir()
		t.Setenv("HOME", home)
		t.Setenv("USERPROFILE", home)
		cmd := NewCommand("test")
		cmd.SetArgs(args)
		if err := cmd.Execute(); err == nil {
			t.Fatalf("accepted invalid arguments: %v", args)
		}
		if _, err := os.Stat(filepath.Join(home, ".rexp")); !os.IsNotExist(err) {
			t.Fatalf("invalid command created a project directory: %v", err)
		}
	}
}

func TestNewCommandExplainsMissingProjectName(t *testing.T) {
	cmd := NewCommand("test")
	cmd.SetArgs([]string{"new"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected missing project name error")
	}
	for _, hint := range []string{"missing project name", "rexp new <project>", "rexp new my-app"} {
		if !strings.Contains(err.Error(), hint) {
			t.Errorf("missing hint %q in error: %v", hint, err)
		}
	}
}
