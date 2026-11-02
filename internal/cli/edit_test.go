package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/itsjustaplant/rexp/internal/config"
)

// A real subprocess, portable across platforms, exercises argument boundaries
// and terminal streams without launching an interactive editor during tests.
func TestEditorHelperProcess(t *testing.T) {
	mode := os.Getenv("REXP_TEST_EDITOR_MODE")
	if mode == "" {
		return
	}
	args := os.Args
	for i, arg := range args {
		if arg == "--" {
			args = args[i+1:]
			break
		}
	}
	if len(args) == 0 {
		os.Exit(9)
	}
	path := args[len(args)-1]
	if log := os.Getenv("REXP_TEST_EDITOR_LOG"); log != "" {
		data, _ := json.Marshal(args)
		if err := os.WriteFile(log, data, 0o600); err != nil {
			os.Exit(8)
		}
	}
	if mode == "write" || mode == "fail" {
		if err := os.WriteFile(path, []byte(os.Getenv("REXP_TEST_EDITOR_CONTENT")), 0o600); err != nil {
			os.Exit(7)
		}
	}
	if mode == "streams" {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			os.Exit(6)
		}
		fmt.Fprintf(os.Stdout, "editor stdout: %s", data)
		fmt.Fprint(os.Stderr, "editor stderr")
	}
	if mode == "fail" {
		os.Exit(3)
	}
	os.Exit(0)
}

func fakeEditor(t *testing.T, mode string) string {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("REXP_TEST_EDITOR_MODE", mode)
	return strconv.Quote(exe) + " -test.run=^TestEditorHelperProcess$ --"
}

func editorHome(t *testing.T) {
	t.Helper()
	home := filepath.Join(t.TempDir(), "home with spaces")
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
}

func TestEditRepairsInvalidFileAndPreservesArguments(t *testing.T) {
	editorHome(t)
	path, err := config.CreateProject("demo", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	valid, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("broken: [yaml"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("REXP_TEST_EDITOR_CONTENT", string(valid))
	log := filepath.Join(t.TempDir(), "args.json")
	t.Setenv("REXP_TEST_EDITOR_LOG", log)
	var output bytes.Buffer
	cmd := NewCommand("test")
	cmd.SetOut(&output)
	cmd.SetArgs([]string{"edit", "demo", "--editor", fakeEditor(t, "write") + ` "an argument with spaces"`})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), `Project "demo" is valid`) {
		t.Fatalf("output: %q", output.String())
	}
	data, err := os.ReadFile(log)
	if err != nil {
		t.Fatal(err)
	}
	var args []string
	if err := json.Unmarshal(data, &args); err != nil {
		t.Fatal(err)
	}
	if len(args) != 2 || args[0] != "an argument with spaces" || args[1] != path {
		t.Fatalf("arguments: %v", args)
	}
	if _, err := config.LoadProject("demo"); err != nil {
		t.Fatal(err)
	}
}

func TestEditPreservesInvalidAndUnsuccessfullyEditedFiles(t *testing.T) {
	for _, mode := range []string{"write", "fail"} {
		t.Run(mode, func(t *testing.T) {
			editorHome(t)
			path, err := config.CreateProject("demo", t.TempDir())
			if err != nil {
				t.Fatal(err)
			}
			t.Setenv("REXP_TEST_EDITOR_CONTENT", "schema_version: 999\n")
			cmd := NewCommand("test")
			cmd.SetArgs([]string{"edit", "demo", "--editor", fakeEditor(t, mode)})
			err = cmd.Execute()
			if err == nil {
				t.Fatal("accepted invalid/failed edit")
			}
			if mode == "write" && !strings.Contains(err.Error(), "validation failed") {
				t.Fatalf("error: %v", err)
			}
			if mode == "fail" && !strings.Contains(err.Error(), "exit status 3") {
				t.Fatalf("error: %v", err)
			}
			data, readErr := os.ReadFile(path)
			if readErr != nil || string(data) != "schema_version: 999\n" {
				t.Fatalf("edits lost: %q %v", data, readErr)
			}
		})
	}
}

func TestNewEditFailurePreservesCreatedProject(t *testing.T) {
	editorHome(t)
	var output bytes.Buffer
	cmd := NewCommand("test")
	cmd.SetOut(&output)
	cmd.SetArgs([]string{"new", "demo", "--edit", "--editor", "rexp-editor-that-does-not-exist"})
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "not found on PATH") {
		t.Fatalf("error: %v", err)
	}
	if _, err := config.LoadProject("demo"); err != nil {
		t.Fatalf("new configuration not preserved: %v", err)
	}
	if !strings.Contains(output.String(), "Created ") {
		t.Fatalf("missing path: %q", output.String())
	}
}

func TestEditConnectsStreams(t *testing.T) {
	editorHome(t)
	if _, err := config.CreateProject("demo", t.TempDir()); err != nil {
		t.Fatal(err)
	}
	var output, stderr bytes.Buffer
	cmd := NewCommand("test")
	cmd.SetOut(&output)
	cmd.SetErr(&stderr)
	cmd.SetIn(strings.NewReader("typed text"))
	cmd.SetArgs([]string{"edit", "demo", "--editor", fakeEditor(t, "streams")})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), "editor stdout: typed text") || stderr.String() != "editor stderr" {
		t.Fatalf("stdout=%q stderr=%q", output.String(), stderr.String())
	}
}

func TestEditInvalidArgumentsAndMissingFiles(t *testing.T) {
	editorHome(t)
	t.Setenv("EDITOR", "rexp-editor-that-does-not-exist")
	for _, test := range []struct {
		args []string
		hint string
	}{
		{[]string{"edit"}, "missing project name"},
		{[]string{"edit", "one", "two"}, "accepts 1 arg"},
		{[]string{"edit", "../escape"}, "project name"},
		{[]string{"edit", "missing"}, "does not exist"},
		{[]string{"new", "demo", "--editor", "editor"}, "--editor requires --edit"},
		{[]string{"new", "demo", "--edit=false", "--editor", "editor"}, "--editor requires --edit"},
	} {
		cmd := NewCommand("test")
		cmd.SetArgs(test.args)
		if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), test.hint) {
			t.Errorf("%v: got %v, want %q", test.args, err, test.hint)
		}
	}
	projects, _ := config.ProjectsDir()
	if _, err := os.Stat(projects); !os.IsNotExist(err) {
		t.Fatalf("invalid command created files: %v", err)
	}
}
