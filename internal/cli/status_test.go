package cli

import (
	"bytes"
	"context"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/itsjustaplant/rexp/internal/config"
	"github.com/itsjustaplant/rexp/internal/programstatus"
)

func reportStates(data string) []string {
	var states []string
	for _, match := range regexp.MustCompile("\x1b\\]7501;state=([a-z]+)").FindAllStringSubmatch(data, -1) {
		states = append(states, match[1])
	}
	return states
}

func TestExecuteReportsSuccessAndEarlyErrors(t *testing.T) {
	for _, test := range []struct {
		args      []string
		states    []string
		wantError bool
	}{
		{[]string{"new", "demo"}, []string{"working", "done"}, false},
		{[]string{"new"}, []string{"error"}, true},
		{[]string{"validate", "missing"}, []string{"working", "error"}, true},
		{[]string{"unknown-command"}, []string{"error"}, true},
		{[]string{"--bad-flag"}, []string{"error"}, true},
		{[]string{"--help"}, nil, false},
		{[]string{"--version"}, nil, false},
		{[]string{"help", "new"}, nil, false},
		{[]string{}, nil, false},
	} {
		t.Run(strings.Join(test.args, " "), func(t *testing.T) {
			editorHome(t)
			var reports, output, stderr bytes.Buffer
			cmd := NewCommand("test")
			cmd.SetArgs(test.args)
			cmd.SetOut(&output)
			cmd.SetErr(&stderr)
			err := Execute(cmd, programstatus.New(&reports))
			if (err != nil) != test.wantError {
				t.Fatalf("error: %v", err)
			}
			if got := reportStates(reports.String()); !reflect.DeepEqual(got, test.states) {
				t.Fatalf("states %v, want %v", got, test.states)
			}
			if strings.Contains(output.String()+stderr.String(), "\x1b]7501;") {
				t.Fatal("status polluted command output")
			}
		})
	}
}

func TestExecuteReportsInitializationFailure(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	if err := os.WriteFile(filepath.Join(home, ".rexp"), []byte("not a directory"), 0o600); err != nil {
		t.Fatal(err)
	}
	var reports bytes.Buffer
	cmd := NewCommand("test")
	cmd.SetArgs([]string{"list"})
	if err := Execute(cmd, programstatus.New(&reports)); err == nil {
		t.Fatal("expected initialization failure")
	}
	if got := reportStates(reports.String()); !reflect.DeepEqual(got, []string{"error"}) {
		t.Fatalf("states: %v", got)
	}
}

func TestCancelledBeforeRunDoesNotCreateProject(t *testing.T) {
	editorHome(t)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	var reports bytes.Buffer
	cmd := NewCommand("test")
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"new", "demo"})
	if err := Execute(cmd, programstatus.New(&reports)); !errors.Is(err, context.Canceled) {
		t.Fatalf("error: %v", err)
	}
	if got := reportStates(reports.String()); !reflect.DeepEqual(got, []string{"idle"}) {
		t.Fatalf("states: %v", got)
	}
	projects, _ := config.ProjectsDir()
	if _, err := os.Stat(filepath.Join(projects, "demo.yaml")); !os.IsNotExist(err) {
		t.Fatalf("cancelled command created a project: %v", err)
	}
	if _, err := readConfirmation(ctx, strings.NewReader("yes\n")); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancelled prompt accepted input: %v", err)
	}
}

func TestDeleteStatusTransitions(t *testing.T) {
	for _, test := range []struct {
		answer string
		states []string
	}{
		{"yes\n", []string{"working", "blocked", "working", "done"}},
		{"no\n", []string{"working", "blocked", "idle"}},
		{"", []string{"working", "blocked", "error"}},
	} {
		t.Run(test.answer, func(t *testing.T) {
			editorHome(t)
			if _, err := config.CreateProject("demo", t.TempDir()); err != nil {
				t.Fatal(err)
			}
			var reports, output bytes.Buffer
			cmd := NewCommand("test")
			cmd.SetArgs([]string{"delete", "demo"})
			cmd.SetIn(strings.NewReader(test.answer))
			cmd.SetOut(&output)
			cmd.SetErr(&output)
			_ = Execute(cmd, programstatus.New(&reports))
			if got := reportStates(reports.String()); !reflect.DeepEqual(got, test.states) {
				t.Fatalf("states %v, want %v", got, test.states)
			}
			if !strings.Contains(reports.String(), ":kind=permission:") {
				t.Fatal("confirmation did not report permission kind")
			}
		})
	}
}

func TestEditStatusTransitions(t *testing.T) {
	editorHome(t)
	if _, err := config.CreateProject("demo", t.TempDir()); err != nil {
		t.Fatal(err)
	}
	var reports, output bytes.Buffer
	cmd := NewCommand("test")
	cmd.SetArgs([]string{"edit", "demo", "--editor", fakeEditor(t, "keep")})
	cmd.SetOut(&output)
	if err := Execute(cmd, programstatus.New(&reports)); err != nil {
		t.Fatal(err)
	}
	if got := reportStates(reports.String()); !reflect.DeepEqual(got, []string{"working", "blocked", "working", "done"}) {
		t.Fatalf("states: %v", got)
	}
	if !strings.Contains(reports.String(), ":kind=question:") {
		t.Fatal("editor did not report waiting for user input")
	}
}

type promptReader struct {
	io.Reader
	started chan struct{}
}

func (reader promptReader) Read(data []byte) (int, error) {
	close(reader.started)
	return reader.Reader.Read(data)
}

func TestInterruptedConfirmationReturnsIdleAndKeepsProject(t *testing.T) {
	editorHome(t)
	path, err := config.CreateProject("demo", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	reader, writer := io.Pipe()
	defer reader.Close()
	defer writer.Close()
	started := make(chan struct{})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var reports, output bytes.Buffer
	cmd := NewCommand("test")
	cmd.SetContext(ctx)
	cmd.SetArgs([]string{"delete", "demo"})
	cmd.SetIn(promptReader{reader, started})
	cmd.SetOut(&output)
	cmd.SetErr(&output)
	finished := make(chan error, 1)
	go func() { finished <- Execute(cmd, programstatus.New(&reports)) }()
	select {
	case <-started:
	case <-time.After(5 * time.Second):
		t.Fatal("prompt did not start")
	}
	cancel()
	select {
	case err := <-finished:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("error: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("prompt ignored cancellation")
	}
	if got := reportStates(reports.String()); !reflect.DeepEqual(got, []string{"working", "blocked", "idle"}) {
		t.Fatalf("states: %v", got)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("interruption removed project: %v", err)
	}
}
