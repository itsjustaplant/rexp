package cli

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/itsjustaplant/rexp/internal/config"
)

func TestDeleteCommandConfirmation(t *testing.T) {
	for _, tc := range []struct {
		name, input, flag      string
		wantDeleted, wantError bool
	}{
		{name: "confirm", input: "y\n", wantDeleted: true},
		{name: "confirm long", input: "YES\n", wantDeleted: true},
		{name: "decline", input: "n\n"},
		{name: "empty answer", input: "\n"},
		{name: "invalid answer", input: "maybe\n"},
		{name: "no input", wantError: true},
		{name: "yes flag", flag: "--yes", wantDeleted: true},
		{name: "short yes flag", flag: "-y", wantDeleted: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("HOME", home)
			t.Setenv("USERPROFILE", home)
			path, err := config.CreateProject("demo", t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			var stdout, stderr bytes.Buffer
			cmd := NewCommand("test")
			cmd.SetIn(strings.NewReader(tc.input))
			cmd.SetOut(&stdout)
			cmd.SetErr(&stderr)
			args := []string{"delete", "demo"}
			if tc.flag != "" {
				args = append(args, tc.flag)
			}
			cmd.SetArgs(args)
			err = cmd.Execute()
			if (err != nil) != tc.wantError {
				t.Fatalf("unexpected error: %v", err)
			}
			_, statErr := os.Stat(path)
			if tc.wantDeleted {
				if !os.IsNotExist(statErr) || !strings.Contains(stdout.String(), "Deleted "+path) {
					t.Fatalf("expected deletion, stat=%v, output=%q", statErr, stdout.String())
				}
			} else if statErr != nil {
				t.Fatalf("configuration was not preserved: %v", statErr)
			}
			if tc.flag != "" && stderr.Len() != 0 {
				t.Fatalf("prompted despite --yes: %q", stderr.String())
			}
		})
	}
}

func TestDeleteCommandRejectsInvalidArguments(t *testing.T) {
	for _, args := range [][]string{{"delete"}, {"delete", "one", "two"}, {"delete", "../escape", "--yes"}, {"delete", "demo", "--unknown"}, {"delete", "missing", "--yes"}} {
		cmd := NewCommand("test")
		home := t.TempDir()
		t.Setenv("HOME", home)
		t.Setenv("USERPROFILE", home)
		cmd.SetArgs(args)
		cmd.SetIn(strings.NewReader(""))
		err := cmd.Execute()
		if err == nil {
			t.Fatalf("accepted invalid arguments: %v", args)
		}
		if len(args) == 1 && (!strings.Contains(err.Error(), "missing project name") || !strings.Contains(err.Error(), "Example: rexp delete my-app")) {
			t.Fatalf("missing helpful usage: %v", err)
		}
	}
}
