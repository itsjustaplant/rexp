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
)

func TestSaveArgumentsAndOverwriteProtection(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("REX_SESSION", "")
	t.Setenv("PATH", t.TempDir())
	path, err := config.CreateProject("existing", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		args []string
		hint string
	}{
		{[]string{"save"}, "missing project name"},
		{[]string{"save", "a", "b"}, "accepts 1 arg"},
		{[]string{"save", "../escape"}, "project name"},
		{[]string{"save", "existing"}, "already exists"},
		{[]string{"save", "demo"}, "no current Rex session"},
		{[]string{"save", "demo", "--session", "session:test"}, "not found on PATH"},
	} {
		cmd := NewCommand("test")
		cmd.SetArgs(test.args)
		if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), test.hint) {
			t.Errorf("%v: got %v, want %q", test.args, err, test.hint)
		}
	}
	after, err := os.ReadFile(path)
	if err != nil || string(after) != string(before) {
		t.Fatalf("existing project changed: %v", err)
	}
	projects, _ := config.ProjectsDir()
	if _, err := os.Stat(filepath.Join(projects, "demo.yaml")); !os.IsNotExist(err) {
		t.Fatalf("failed save created a file: %v", err)
	}
}

func TestSaveCommandWithRexFixture(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("test Rex executable uses a POSIX shell")
	}
	home, bin, root := t.TempDir(), t.TempDir(), t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("REX_SESSION", "session:source")
	t.Setenv("PATH", bin)
	quoted, _ := json.Marshal(root)
	snapshot := `{"session_id":"session:source","active_window_id":"window:1","windows":[{"window_id":"window:1","label":"Development","focused_block_id":"block:1","layers":[{"kind":"tiled","layout":{"block_id":"block:1"},"blocks":[{"block_id":"block:1","label":"editor","creator_name":"com.superlogical.terminal"}]}]}]}`
	snapshotPath, processPath, logPath := filepath.Join(bin, "snapshot.json"), filepath.Join(bin, "process.json"), filepath.Join(bin, "calls.log")
	for path, contents := range map[string]string{
		snapshotPath: snapshot,
		processPath:  `{"foreground":{"cwd":` + string(quoted) + `,"argv0":"dangerous-program-with-unknown-arguments"}}`,
		filepath.Join(bin, "rex"): `#!/bin/sh
printf '%s\n' "$@" >> "$REXP_TEST_CALLS"
case "$1 $2" in
  'session inspect') exec /bin/cat "$REXP_TEST_SNAPSHOT" ;;
  'block call') exec /bin/cat "$REXP_TEST_PROCESS" ;;
  *) exit 99 ;;
esac
`,
	} {
		if err := os.WriteFile(path, []byte(contents), 0o700); err != nil {
			t.Fatal(err)
		}
	}
	t.Setenv("REXP_TEST_SNAPSHOT", snapshotPath)
	t.Setenv("REXP_TEST_PROCESS", processPath)
	t.Setenv("REXP_TEST_CALLS", logPath)
	var output, warnings bytes.Buffer
	cmd := NewCommand("test")
	cmd.SetOut(&output)
	cmd.SetErr(&warnings)
	cmd.SetArgs([]string{"save", "demo"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	project, err := config.LoadProject("demo")
	if err != nil {
		t.Fatal(err)
	}
	if project.Tabs[0].Layout.Cwd != root || project.Tabs[0].Layout.Pane != "editor" || len(project.Tabs[0].Layout.Command) != 0 {
		t.Fatalf("saved pane: %+v", project.Tabs[0].Layout)
	}
	if output.String() != "Saved "+filepath.Join(home, ".rexp", "projects", "demo.yaml")+"\n" || warnings.Len() != 0 {
		t.Fatalf("output=%q warnings=%q", output.String(), warnings.String())
	}
	// --session works independently of the shell environment, including labels
	// starting with a hyphen; they must remain positional to Rex.
	t.Setenv("REX_SESSION", "")
	cmd = NewCommand("test")
	cmd.SetOut(&output)
	cmd.SetErr(&warnings)
	cmd.SetArgs([]string{"save", "explicit", "--session=-label"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	calls, err := os.ReadFile(logPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(calls), "session\ninspect\n--json\n--\n-label\n") {
		t.Fatalf("session argument was not passed literally: %s", calls)
	}
}
