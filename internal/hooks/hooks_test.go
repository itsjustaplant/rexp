package hooks

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"rexp/internal/workspace"
)

func TestRunUsesProjectDirectoryEnvironmentAndStopsOnFailure(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX fixture")
	}
	root := t.TempDir()
	project := &workspace.Project{Name: "demo", Root: root, Env: map[string]string{"HOOK_VALUE": "hello world", "REXP_PROJECT": "wrong"}}
	var out bytes.Buffer
	err := Run(context.Background(), project, "session:test", "before_start", [][]string{{"sh", "-c", `printf '%s|%s|%s|%s' "$PWD" "$HOOK_VALUE" "$REXP_PROJECT" "$REXP_SESSION_ID"`}}, &out, &out)
	if err != nil {
		t.Fatal(err)
	}
	canonical, _ := filepath.EvalSymlinks(root)
	if out.String() != canonical+"|hello world|demo|session:test" {
		t.Fatalf("hook context %q", out.String())
	}
	out.Reset()
	err = Run(context.Background(), project, "", "after_start", [][]string{{"sh", "-c", "exit 7"}, {"sh", "-c", "echo should-not-run"}}, &out, &out)
	if err == nil || !strings.Contains(err.Error(), "hooks.after_start") || out.Len() != 0 {
		t.Fatalf("failure err=%v output=%s", err, &out)
	}
}

func TestWorkerRequiresSuccessfulCloseEvent(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX fake Rex")
	}
	for _, test := range []struct {
		name, output, exit string
		run                bool
	}{
		{"close", "rexp-hook-ready\\nrexp-hook-closed\\n", "0", true},
		{"disconnect", "rexp-hook-ready\\n", "1", false},
		{"eof", "rexp-hook-ready\\n", "0", false},
		{"failed-close", "rexp-hook-ready\\nrexp-hook-closed\\n", "1", false},
		{"not-ready", "rexp-hook-closed\\n", "0", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			dir := t.TempDir()
			marker := filepath.Join(dir, "marker")
			script := "#!/bin/sh\nprintf '" + test.output + "'\nprintf 'underlying Rex diagnostic\\n' >&2\nexit " + test.exit + "\n"
			if err := os.WriteFile(filepath.Join(dir, "rex"), []byte(script), 0700); err != nil {
				t.Fatal(err)
			}
			t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
			data := snapshot{Version: 1, Session: "session:test", Project: workspace.Project{Name: "demo", Root: dir, Hooks: workspace.Hooks{AfterClose: [][]string{{"sh", "-c", `printf done > "$1"`, "sh", marker}}}}}
			encoded, _ := json.Marshal(data)
			path := filepath.Join(dir, "snapshot.json")
			os.WriteFile(path, encoded, 0600)
			var out bytes.Buffer
			err := Worker(context.Background(), []string{path}, &out, &out)
			if !strings.Contains(out.String(), "underlying Rex diagnostic") {
				t.Fatal("lost Rex stderr diagnostic")
			}
			_, statErr := os.Stat(marker)
			if (err == nil) != test.run || (statErr == nil) != test.run {
				t.Fatalf("run=%v err=%v marker=%v output=%s", test.run, err, statErr, &out)
			}
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Fatalf("snapshot not cleaned: %v", err)
			}
		})
	}
}

func TestMain(m *testing.M) {
	if len(os.Args) > 1 && os.Args[1] == Mode {
		if err := Worker(context.Background(), os.Args[2:], os.Stdout, os.Stderr); err != nil {
			os.Stderr.WriteString(err.Error())
			os.Exit(1)
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func TestWatchDetachedHandshakeAndClose(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("POSIX fake Rex")
	}
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	trigger := filepath.Join(dir, "close")
	marker := filepath.Join(dir, "done")
	t.Setenv("CLOSE_TRIGGER", trigger)
	script := `#!/bin/sh
printf 'rexp-hook-ready\n'
while [ ! -f "$CLOSE_TRIGGER" ]; do sleep 0.05; done
printf 'rexp-hook-closed\n'
`
	if err := os.WriteFile(filepath.Join(dir, "rex"), []byte(script), 0700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	project := &workspace.Project{Name: "demo", Root: dir, Hooks: workspace.Hooks{AfterClose: [][]string{{"sh", "-c", `printf done > "$1"`, "sh", marker}}}}
	if err := Watch(context.Background(), project, "session:test"); err != nil {
		t.Fatal(err)
	}
	// Always release the worker even if an assertion below fails.
	defer os.WriteFile(trigger, nil, 0600)
	if _, err := os.Stat(marker); !os.IsNotExist(err) {
		t.Fatal("cleanup ran before closure")
	}
	if err := Watch(context.Background(), project, "session:test"); err == nil {
		t.Fatal("duplicate observer allowed")
	}
	os.WriteFile(trigger, nil, 0600)
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if content, err := os.ReadFile(marker); err == nil && string(content) == "done" {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatal("detached cleanup did not run")
}
