package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"rexp/internal/config"
)

func TestStopCommandConfirmationAndConfigurationPreservation(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX fake Rex")
	}
	for _, test := range []struct {
		name, input, current                   string
		yes, missing, pending, kill, wantError bool
	}{
		{name: "confirmed", input: "yes\n", kill: true},
		{name: "short-confirmation", input: "y\n", kill: true},
		{name: "declined", input: "no\n"},
		{name: "default-no", input: "\n"},
		{name: "eof", wantError: true},
		{name: "yes", yes: true, kill: true},
		{name: "missing", missing: true},
		{name: "incomplete", input: "yes\n", pending: true, kill: true},
		{name: "current-protected", input: "yes\n", current: "session:test", wantError: true},
		{name: "current-explicit", current: "session:test", yes: true, kill: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			editorHome(t)
			path, err := config.CreateProject("demo", t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			original, _ := os.ReadFile(path)
			// Stop must not execute startup or close hooks itself.
			content := append(append([]byte{}, original...), []byte("hooks:\n  before_start: [[missing-start-program]]\n  after_close: [[missing-close-program]]\n")...)
			if err := os.WriteFile(path, content, 0600); err != nil {
				t.Fatal(err)
			}
			dir := t.TempDir()
			log := filepath.Join(dir, "calls.log")
			t.Setenv("PATH", dir)
			t.Setenv("REXP_STOP_LOG", log)
			t.Setenv("REX_SESSION", test.current)
			listing := `{"sessions":[{"session_id":"session:test","label":"rexp:demo"}]}`
			if test.pending {
				listing = strings.ReplaceAll(listing, "rexp:demo", "rexp:demo:starting")
			}
			if test.missing {
				listing = `{"sessions":[]}`
			}
			t.Setenv("REXP_STOP_LISTING", listing)
			script := `#!/bin/sh
printf '%s\n' "$@" >> "$REXP_STOP_LOG"
case "$1" in
 ls) printf '%s\n' "$REXP_STOP_LISTING" ;;
 kill) [ "$2" = 'session:test' ] || exit 98 ;;
 *) exit 99 ;;
esac
`
			if err := os.WriteFile(filepath.Join(dir, "rex"), []byte(script), 0700); err != nil {
				t.Fatal(err)
			}
			var out, stderr bytes.Buffer
			cmd := NewCommand("test")
			cmd.SetIn(strings.NewReader(test.input))
			cmd.SetOut(&out)
			cmd.SetErr(&stderr)
			args := []string{"stop", "demo"}
			if test.yes {
				args = append(args, "--yes")
			}
			cmd.SetArgs(args)
			err = cmd.Execute()
			if (err != nil) != test.wantError {
				t.Fatalf("error=%v output=%s stderr=%s", err, &out, &stderr)
			}
			calls, _ := os.ReadFile(log)
			if strings.Contains(string(calls), "kill\nsession:test\n") != test.kill {
				t.Fatalf("unexpected mutations: %s", calls)
			}
			after, err := os.ReadFile(path)
			if err != nil || !bytes.Equal(content, after) {
				t.Fatalf("modified config: %v", err)
			}
			if test.missing && !strings.Contains(out.String(), "not running") {
				t.Fatal("missing result")
			}
			if test.kill && !strings.Contains(out.String(), "Stopped project") {
				t.Fatal("missing success output")
			}
		})
	}
}

func TestStopFileTargetsAndArguments(t *testing.T) {
	editorHome(t)
	dir := t.TempDir()
	t.Chdir(dir)
	path, err := config.CreateLocalProject("demo", dir)
	if err != nil {
		t.Fatal(err)
	}
	// No Rex available: successful target loading reaches the expected PATH error.
	t.Setenv("PATH", t.TempDir())
	for _, args := range [][]string{{"stop", ".", "--yes"}, {"stop", "--file", path, "--yes"}} {
		cmd := NewCommand("test")
		cmd.SetArgs(args)
		if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "Rex CLI not found") {
			t.Fatalf("%v: %v", args, err)
		}
	}
	for _, args := range [][]string{{"stop"}, {"stop", "demo", "extra"}, {"stop", "demo", "--file", path}, {"stop", "--file", ""}, {"stop", "../demo"}} {
		cmd := NewCommand("test")
		cmd.SetArgs(args)
		if err := cmd.Execute(); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
}
