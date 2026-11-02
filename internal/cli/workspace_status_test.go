package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"rexp/internal/config"
	"rexp/internal/rex"
)

func TestStatusCLIStatesAreReadOnly(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX fake Rex")
	}
	editorHome(t)
	root := t.TempDir()
	path, err := config.CreateProject("demo", root)
	if err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content = append(content, []byte("env:\n  SECRET: secret-not-for-output\nhooks:\n  before_start: [[program-must-not-run]]\n  after_start: [[program-must-not-run]]\n  after_close: [[program-must-not-run]]\n")...)
	if err := os.WriteFile(path, content, 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(root); err != nil {
		t.Fatal(err)
	} // Root existence is irrelevant to status.
	bin := t.TempDir()
	calls := filepath.Join(bin, "calls.log")
	t.Setenv("PATH", bin)
	t.Setenv("REXP_STATUS_CALLS", calls)
	script := `#!/bin/sh
printf '%s\n' "$*" >> "$REXP_STATUS_CALLS"
[ "$1" = ls ] && [ "$2" = --json ] || exit 98
printf '%s\n' "$REXP_STATUS_LISTING"
`
	if err := os.WriteFile(filepath.Join(bin, "rex"), []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		state         rex.WorkspaceState
		listing, hint string
	}{
		{rex.WorkspaceMissing, `{"sessions":[]}`, "not running"},
		{rex.WorkspaceRunning, `{"sessions":[{"session_id":"session:1","label":"rexp:demo"}]}`, "is running"},
		{rex.WorkspaceIncomplete, `{"sessions":[{"session_id":"session:1","label":"rexp:demo:starting"}]}`, "incomplete startup"},
		{rex.WorkspaceAmbiguous, `{"sessions":[{"session_id":"session:2","label":"rexp:demo"},{"session_id":"session:1","label":"rexp:demo"}]}`, "ambiguous workspace"},
	} {
		for _, jsonOutput := range []bool{false, true} {
			t.Setenv("REXP_STATUS_LISTING", test.listing)
			var out, stderr bytes.Buffer
			cmd := NewCommand("test")
			cmd.SetOut(&out)
			cmd.SetErr(&stderr)
			args := []string{"status", "demo"}
			if jsonOutput {
				args = append(args, "--json")
			}
			cmd.SetArgs(args)
			if err := cmd.Execute(); err != nil {
				t.Fatalf("%s: %v", test.state, err)
			}
			if strings.Contains(out.String(), "secret-not-for-output") || strings.Contains(out.String(), "program-must-not-run") {
				t.Fatal("leaked project configuration")
			}
			if jsonOutput {
				var result rex.WorkspaceStatus
				if err := json.Unmarshal(out.Bytes(), &result); err != nil || result.Status != test.state {
					t.Fatalf("JSON %s error=%v", &out, err)
				}
				if result.Sessions == nil {
					t.Fatal("sessions must be an array")
				}
			} else if !strings.Contains(out.String(), test.hint) {
				t.Fatalf("output %q missing %q", out.String(), test.hint)
			}
			if stderr.Len() != 0 {
				t.Fatalf("stderr %s", &stderr)
			}
		}
	}
	after, err := os.ReadFile(path)
	if err != nil || !bytes.Equal(content, after) {
		t.Fatal("status modified configuration")
	}
	data, err := os.ReadFile(calls)
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if line != "ls --json" {
			t.Fatalf("mutating Rex request %q", line)
		}
	}
}

func TestStatusFileTargetsAndErrors(t *testing.T) {
	editorHome(t)
	dir := t.TempDir()
	t.Chdir(dir)
	path, err := config.CreateLocalProject("demo", dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", t.TempDir())
	for _, args := range [][]string{{"status", "."}, {"status", "--file", path, "--json"}} {
		cmd := NewCommand("test")
		cmd.SetArgs(args)
		if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "Rex CLI not found") {
			t.Fatalf("%v: %v", args, err)
		}
	}
	for _, args := range [][]string{{"status"}, {"status", "demo", "extra"}, {"status", "demo", "--file", path}, {"status", "--file", ""}, {"status", "missing"}} {
		cmd := NewCommand("test")
		cmd.SetArgs(args)
		if err := cmd.Execute(); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
}
