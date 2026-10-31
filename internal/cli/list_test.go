package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"rexp/internal/config"
)

func TestListCommand(t *testing.T) {
	for _, populated := range []bool{false, true} {
		home := t.TempDir()
		t.Setenv("HOME", home)
		t.Setenv("USERPROFILE", home)
		if populated {
			if err := config.Initialize(); err != nil {
				t.Fatal(err)
			}
			for _, name := range []string{"zebra", "alpha"} {
				if err := os.WriteFile(filepath.Join(home, ".rexp", "projects", name+".yaml"), nil, 0o600); err != nil {
					t.Fatal(err)
				}
			}
		}
		var stdout, stderr bytes.Buffer
		cmd := NewCommand("test")
		cmd.SetOut(&stdout)
		cmd.SetErr(&stderr)
		cmd.SetArgs([]string{"list"})
		if err := cmd.Execute(); err != nil {
			t.Fatal(err)
		}
		if populated && stdout.String() != "alpha\nzebra\n" {
			t.Fatalf("unexpected list output: %q", stdout.String())
		}
		if !populated && !strings.Contains(stdout.String(), "rexp new <project>") {
			t.Fatalf("missing empty-list guidance: %q", stdout.String())
		}
		if stderr.Len() != 0 {
			t.Fatalf("unexpected stderr: %q", stderr.String())
		}
	}
}

func TestListCommandRejectsArguments(t *testing.T) {
	cmd := NewCommand("test")
	cmd.SetArgs([]string{"list", "my-app"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("expected unexpected-argument error")
	}
}
