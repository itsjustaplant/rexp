package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/itsjustaplant/rexp/internal/config"
	"github.com/itsjustaplant/rexp/internal/rex"
)

func switchFixture(t *testing.T) string {
	t.Helper()
	if runtime.GOOS == "windows" {
		t.Skip("POSIX fake Rex")
	}
	editorHome(t)
	bin := t.TempDir()
	calls := filepath.Join(bin, "calls.log")
	t.Setenv("PATH", bin)
	t.Setenv("REXP_SWITCH_CALLS", calls)
	script := `#!/bin/sh
printf '%s\n' "$*" >> "$REXP_SWITCH_CALLS"
if [ "$1" = ls ] && [ "$2" = --json ]; then
  printf '%s\n' "$REXP_SWITCH_LISTING"
  exit 0
fi
if [ "$1" = -C ]; then shift 2; fi
[ "$1" = do ] && [ "$2" = session.select ] || exit 98
`
	if err := os.WriteFile(filepath.Join(bin, "rex"), []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	return calls
}

func TestSwitchCLINamedSessionWithoutConfig(t *testing.T) {
	calls := switchFixture(t)
	t.Setenv("REXP_SWITCH_LISTING", `{"sessions":[{"session_id":"session:1","label":"rexp:demo"}]}`)
	out, err := runDiscovery(t, "switch", "demo", "--client", "macbook")
	if err != nil || !strings.Contains(out, "Switched to Rex session rexp:demo (session:1)") {
		t.Fatalf("output=%q err=%v", out, err)
	}
	data, err := os.ReadFile(calls)
	if err != nil || string(data) != "ls --json\n-C macbook do session.select session_id=session:1\n" {
		t.Fatalf("unexpected Rex calls %q: %v", data, err)
	}
}

func TestSwitchCLILocalTargetsDoNotRunHooksOrRequireDirectories(t *testing.T) {
	switchFixture(t)
	dir := t.TempDir()
	t.Chdir(dir)
	path, err := config.CreateLocalProject("demo", dir)
	if err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content = []byte(strings.Replace(string(content), `root: "."`, `root: "./missing"`, 1))
	content = append(content, []byte("hooks:\n  before_start: [[must-not-run]]\n  after_start: [[must-not-run]]\n  after_close: [[must-not-run]]\n")...)
	if err := os.WriteFile(path, content, 0600); err != nil {
		t.Fatal(err)
	}
	project, err := config.LoadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	label := rex.SessionLabel(project)
	t.Setenv("REXP_SWITCH_LISTING", `{"sessions":[{"session_id":"session:local","label":"`+label+`"},{"session_id":"session:global","label":"rexp:demo"}]}`)
	for _, args := range [][]string{{"switch", "."}, {"switch", "--file", path}} {
		out, err := runDiscovery(t, args...)
		if err != nil || !strings.Contains(out, "session:local") || strings.Contains(out, "session:global") {
			t.Fatalf("%v: %q %v", args, out, err)
		}
	}
	after, err := os.ReadFile(path)
	if err != nil || string(after) != string(content) {
		t.Fatal("switch changed the configuration")
	}
}

func TestSwitchCLIRejectsMissingAndInvalidTargets(t *testing.T) {
	calls := switchFixture(t)
	for _, args := range [][]string{{"switch"}, {"switch", ""}, {"switch", " "}, {"switch", "one", "two"}, {"switch", "demo", "--file", "config.yaml"}, {"switch", "--file", ""}} {
		if _, err := runDiscovery(t, args...); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
	if _, err := os.Stat(calls); !os.IsNotExist(err) {
		t.Fatalf("invalid arguments contacted Rex: %v", err)
	}
	t.Setenv("REXP_SWITCH_LISTING", `{"sessions":[]}`)
	out, err := runDiscovery(t, "switch", "missing")
	if err == nil || !strings.Contains(err.Error(), "no running Rex session") || strings.Contains(out, "Switched") {
		t.Fatalf("missing target: %q %v", out, err)
	}
	data, err := os.ReadFile(calls)
	if err != nil || string(data) != "ls --json\n" {
		t.Fatalf("missing target mutated Rex: %q %v", data, err)
	}
}
