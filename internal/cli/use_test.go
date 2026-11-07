package cli

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/itsjustaplant/rexp/internal/config"
)

func TestUseStartsThenReusesAndSelectsAfterStartup(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX fake Rex and hooks")
	}
	editorHome(t)
	path, err := config.CreateProject("demo", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content = append(content, []byte(`hooks:
  before_start:
    - [/bin/sh, -c, 'printf "before_start\n" >> "$REXP_USE_CALLS"']
  after_start:
    - [/bin/sh, -c, 'printf "after_start\n" >> "$REXP_USE_CALLS"']
`)...)
	if err := os.WriteFile(path, content, 0600); err != nil {
		t.Fatal(err)
	}
	bin := t.TempDir()
	calls := filepath.Join(bin, "calls.log")
	t.Setenv("PATH", bin)
	t.Setenv("REXP_USE_CALLS", calls)
	t.Setenv("REXP_USE_READY", filepath.Join(bin, "ready"))
	script := "#!/bin/sh\n" + rexIdentityStub + `
printf '%s\n' "$*" >> "$REXP_USE_CALLS"
if [ "$1" = -C ]; then shift 2; fi
case "$1" in
  ls)
    if [ -f "$REXP_USE_READY" ]; then
      printf '%s\n' '{"sessions":[{"session_id":"session:test","label":"rexp:demo"}]}'
    else
      printf '%s\n' '{"sessions":[]}'
    fi
    ;;
  new)
    printf '%s\n' '{"session_id":"session:test","initial_windows":[{"block_ids":["block:1"]}]}'
    ;;
  block|focus) ;;
  session)
    [ "$2" = rename ] && [ "$4" = rexp:demo ] || exit 98
    printf ready > "$REXP_USE_READY"
    ;;
  do)
    [ -f "$REXP_USE_READY" ] && [ "$2" = session.select ] || exit 98
    ;;
  *) exit 98 ;;
esac
`
	if err := os.WriteFile(filepath.Join(bin, "rex"), []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	for _, action := range []string{"Started", "Reused"} {
		out, err := runDiscovery(t, "use", "demo", "--client", "macbook")
		if err != nil || !strings.Contains(out, action+` project "demo"`) {
			t.Fatalf("expected %s: %q %v", action, out, err)
		}
	}
	data, err := os.ReadFile(calls)
	if err != nil {
		t.Fatal(err)
	}
	log := string(data)
	if strings.Count(log, "\nnew ") != 1 || strings.Count(log, "\nbefore_start\n") != 1 || strings.Count(log, "\nafter_start\n") != 1 || strings.Count(log, "-C macbook do session.select session_id=session:test\n") != 2 {
		t.Fatalf("unexpected lifecycle calls:\n%s", log)
	}
	if strings.Index(log, "after_start\n") > strings.Index(log, "do session.select") {
		t.Fatalf("selected before startup hook completed:\n%s", log)
	}
}

func TestUseRejectsInvalidTargetsAndBackground(t *testing.T) {
	editorHome(t)
	t.Setenv("PATH", t.TempDir())
	for _, args := range [][]string{
		{"use"}, {"use", "demo", "extra"}, {"use", "demo", "--file", "workspace.yaml"},
		{"use", "--file", ""}, {"use", "missing"}, {"use", "demo", "--background"},
	} {
		if _, err := runDiscovery(t, args...); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
	for _, args := range [][]string{{"use", "--help"}, {"switch", "--help"}} {
		out, err := runDiscovery(t, args...)
		if err != nil || !strings.Contains(out, "--client") || strings.Contains(out, "--background") {
			t.Fatalf("help for %v: %q %v", args, out, err)
		}
	}
}

func TestUseLocalAndFileTargetsReachStartup(t *testing.T) {
	editorHome(t)
	dir := t.TempDir()
	t.Chdir(dir)
	path, err := config.CreateLocalProject("demo", dir)
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", t.TempDir())
	for _, args := range [][]string{{"use", "."}, {"use", "--file", path}} {
		if _, err := runDiscovery(t, args...); err == nil || !strings.Contains(err.Error(), "Rex CLI not found") {
			t.Fatalf("%v did not reach Rex: %v", args, err)
		}
	}
}
