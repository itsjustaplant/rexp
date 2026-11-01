package rex

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"rexp/internal/panerunner"
	"rexp/internal/workspace"
)

type fakeRex struct {
	calls   [][]string
	listing string
	fail    string
	splits  int
}

func (fake *fakeRex) run(_ context.Context, args ...string) ([]byte, error) {
	fake.calls = append(fake.calls, append([]string(nil), args...))
	if args[0] == fake.fail {
		return nil, errors.New("injected failure")
	}
	switch args[0] {
	case "ls":
		if fake.listing != "" {
			return []byte(fake.listing), nil
		}
		return []byte(`{"sessions":[]}`), nil
	case "new":
		return []byte(`{"session_id":"session:test","initial_windows":[{"block_ids":["block:1"]}]}`), nil
	case "split":
		fake.splits++
		return []byte(fmt.Sprintf(`{"block_ids":["block:%d"]}`, fake.splits+1)), nil
	case "window":
		return []byte(`{"window_id":"window:tab2","block_ids":["block:tab2"]}`), nil
	}
	return nil, nil
}

func projectFixture(t *testing.T) *workspace.Project {
	t.Helper()
	dir := t.TempDir()
	return &workspace.Project{Name: "demo", Root: dir, Tabs: []workspace.Tab{{Name: "Development", Focus: "editor", Layout: workspace.Layout{Pane: "editor", Cwd: dir}}}}
}

func argument(args []string, flag string) string {
	for i := range args {
		if args[i] == flag && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}

func TestStartNestedLayout(t *testing.T) {
	project := projectFixture(t)
	leaf := func(name string) workspace.Layout { return workspace.Layout{Pane: name, Cwd: project.Root} }
	editor := leaf("editor")
	editor.Command = []string{"editor", "a file", "$(touch unsafe)"}
	project.Tabs[0].Layout = workspace.Layout{Split: "columns", Sizes: []int{50, 50}, Children: []workspace.Layout{editor, {Split: "rows", Sizes: []int{34, 33, 33}, Children: []workspace.Layout{leaf("server"), leaf("logs"), leaf("shell")}}}}
	project.Tabs = append(project.Tabs, workspace.Tab{Name: "--help", Focus: "other", Layout: leaf("other")})
	fake := &fakeRex{}
	result, err := (&Client{run: fake.run}).Start(context.Background(), project, Options{Client: "client:test"})
	if err != nil {
		t.Fatal(err)
	}
	if result.SessionID != "session:test" || result.Reused {
		t.Fatalf("unexpected result: %+v", result)
	}
	var splits [][]string
	for _, call := range fake.calls {
		if call[0] == "split" {
			splits = append(splits, call)
		}
	}
	if len(splits) != 3 {
		t.Fatalf("splits: %v", splits)
	}
	for i, expected := range []struct{ anchor, direction, ratio, label string }{{"block:1", "right", "50", "server"}, {"block:2", "below", "66", "logs"}, {"block:3", "below", "50", "shell"}} {
		call := splits[i]
		if argument(call, "-b") != expected.anchor || argument(call, "--split") != expected.direction || argument(call, "--ratio") != expected.ratio || argument(call, "--label") != expected.label {
			t.Fatalf("split %d: %v", i, call)
		}
	}
	newCall := fake.calls[1]
	if newCall[1] != "rexp:demo:starting" || !reflect.DeepEqual(newCall[len(newCall)-4:], []string{"--", "editor", "a file", "$(touch unsafe)"}) {
		t.Fatalf("command arguments changed: %v", newCall)
	}
	last := fake.calls[len(fake.calls)-3:]
	want := [][]string{{"focus", "-s", "session:test", "block:1"}, {"session", "rename", "session:test", "rexp:demo"}, {"-C", "client:test", "do", "session.select", "session_id=session:test"}}
	if !reflect.DeepEqual(last, want) {
		t.Fatalf("final focus/selection: %v", last)
	}
	var renamed bool
	for _, call := range fake.calls {
		if reflect.DeepEqual(call, []string{"window", "rename", "-s", "session:test", "--", "window:tab2", "--help"}) {
			renamed = true
		}
		if call[0] == "window" && call[1] == "new" {
			for _, arg := range call {
				if arg == "--help" {
					t.Fatal("tab label passed as a flag")
				}
			}
		}
	}
	if !renamed {
		t.Fatal("tab name was not set literally")
	}
}

func TestStartReusesSession(t *testing.T) {
	for _, background := range []bool{false, true} {
		t.Run(fmt.Sprint(background), func(t *testing.T) {
			fake := &fakeRex{listing: `{"sessions":[{"session_id":"session:existing","label":"rexp:demo"}]}`}
			result, err := (&Client{run: fake.run}).Start(context.Background(), projectFixture(t), Options{Background: background})
			if err != nil || !result.Reused {
				t.Fatalf("result=%+v err=%v", result, err)
			}
			want := [][]string{{"ls", "--json"}}
			if !background {
				want = append(want, []string{"do", "session.select", "session_id=session:existing"})
			}
			if !reflect.DeepEqual(fake.calls, want) {
				t.Fatalf("reuse launched commands: %v", fake.calls)
			}
		})
	}
}

func TestStartSelectsConfiguredTabWithoutChangingOrder(t *testing.T) {
	project := projectFixture(t)
	project.Tabs = append(project.Tabs, workspace.Tab{Name: "Server", Selected: true, Focus: "server", Layout: workspace.Layout{Pane: "server", Cwd: project.Root}})
	fake := &fakeRex{}
	if _, err := (&Client{run: fake.run}).Start(context.Background(), project, Options{Background: true}); err != nil {
		t.Fatal(err)
	}
	if argument(fake.calls[1], "--window") != "Development" {
		t.Fatal("tab order changed")
	}
	last := fake.calls[len(fake.calls)-2]
	if !reflect.DeepEqual(last, []string{"focus", "-s", "session:test", "block:tab2"}) {
		t.Fatalf("final focus: %v", last)
	}
}

func TestStartWrapsSequencesForInitialTabNewTabsAndSplits(t *testing.T) {
	project := projectFixture(t)
	leaf := func(name string) workspace.Layout {
		return workspace.Layout{Pane: name, Cwd: project.Root, Commands: [][]string{{"echo", "first"}, {"echo", "second"}}}
	}
	project.Tabs[0].Layout = workspace.Layout{Split: "columns", Children: []workspace.Layout{leaf("editor"), leaf("right")}}
	project.Tabs = append(project.Tabs, workspace.Tab{Name: "Other", Focus: "other", Layout: leaf("other")})
	fake := &fakeRex{}
	if _, err := (&Client{run: fake.run}).Start(context.Background(), project, Options{Background: true}); err != nil {
		t.Fatal(err)
	}
	wrapped := 0
	for _, call := range fake.calls {
		if call[0] != "new" && call[0] != "split" && !(call[0] == "window" && call[1] == "new") {
			continue
		}
		for i, arg := range call {
			if arg == "--" {
				if len(call[i+1:]) != 3 || call[i+2] != panerunner.Mode {
					t.Fatalf("not a pane runner: %v", call)
				}
				payload, err := panerunner.Decode(call[i+3])
				if err != nil || len(payload.Commands) != 2 {
					t.Fatalf("payload: %+v %v", payload, err)
				}
				wrapped++
			}
		}
	}
	if wrapped != 3 {
		t.Fatalf("wrapped %d panes", wrapped)
	}
}

func TestStartSplitsSiblingRegionsBeforeTheirChildren(t *testing.T) {
	project := projectFixture(t)
	leaf := func(name string) workspace.Layout { return workspace.Layout{Pane: name, Cwd: project.Root} }
	project.Tabs[0].Layout = workspace.Layout{Split: "columns", Children: []workspace.Layout{
		{Split: "rows", Children: []workspace.Layout{leaf("editor"), leaf("bottom")}},
		leaf("right"),
	}}
	fake := &fakeRex{}
	if _, err := (&Client{run: fake.run}).Start(context.Background(), project, Options{Background: true}); err != nil {
		t.Fatal(err)
	}
	var splits [][]string
	for _, call := range fake.calls {
		if call[0] == "split" {
			splits = append(splits, call)
		}
	}
	if len(splits) != 2 || argument(splits[0], "--split") != "right" || argument(splits[1], "--split") != "below" || argument(splits[1], "-b") != "block:1" {
		t.Fatalf("subdivided the wrong region: %v", splits)
	}
}

func TestStartRejectsIncompleteCreationResponses(t *testing.T) {
	for _, response := range []string{`{}`, `{"session_id":"session:test","initial_windows":[]}`, `{"session_id":"session:test","initial_windows":[{"block_ids":[""]}]}`, `not json`} {
		calls := 0
		client := &Client{run: func(_ context.Context, args ...string) ([]byte, error) {
			calls++
			if args[0] == "ls" {
				return []byte(`{"sessions":[]}`), nil
			}
			return []byte(response), nil
		}}
		if _, err := client.Start(context.Background(), projectFixture(t), Options{}); err == nil {
			t.Fatalf("accepted %q", response)
		}
		if calls != 2 {
			t.Fatalf("retried or continued after incomplete creation: %d calls", calls)
		}
	}
}

func TestStartRejectsAmbiguousSessions(t *testing.T) {
	for _, listing := range []string{
		`{"sessions":[{"session_id":"s1","label":"rexp:demo:starting"}]}`,
		`{"sessions":[{"session_id":"s1","label":"rexp:DEMO"}]}`,
		`{"sessions":[{"session_id":"s1","label":"rexp:demo"},{"session_id":"s2","label":"rexp:demo"}]}`,
		`{"sessions":[{"label":"rexp:demo"}]}`,
		`not json`,
	} {
		fake := &fakeRex{listing: listing}
		if _, err := (&Client{run: fake.run}).Start(context.Background(), projectFixture(t), Options{}); err == nil {
			t.Fatalf("accepted %s", listing)
		}
		if len(fake.calls) != 1 {
			t.Fatalf("mutated ambiguous session: %v", fake.calls)
		}
	}
}

func TestStartPreflightBeforeMutation(t *testing.T) {
	project := projectFixture(t)
	project.Tabs[0].Layout.Cwd = filepath.Join(project.Root, "missing")
	fake := &fakeRex{}
	if _, err := (&Client{run: fake.run}).Start(context.Background(), project, Options{}); err == nil {
		t.Fatal("accepted missing cwd")
	}
	if len(fake.calls) != 0 {
		t.Fatalf("called Rex before preflight: %v", fake.calls)
	}
}

func TestStartLeavesFailedWorkspaceInspectible(t *testing.T) {
	project := projectFixture(t)
	project.Tabs[0].Layout = workspace.Layout{Split: "columns", Children: []workspace.Layout{project.Tabs[0].Layout, {Pane: "other", Cwd: project.Root}}}
	fake := &fakeRex{fail: "split"}
	result, err := (&Client{run: fake.run}).Start(context.Background(), project, Options{})
	if err == nil || !strings.Contains(err.Error(), "existing panes were left running") || result.SessionID != "session:test" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	if len(fake.calls) != 4 || fake.calls[3][0] != "split" {
		t.Fatalf("retried or mutated after failure: %v", fake.calls)
	}
}

func TestStartSelectionFailureKeepsReadySession(t *testing.T) {
	fake := &fakeRex{fail: "do"}
	result, err := (&Client{run: fake.run}).Start(context.Background(), projectFixture(t), Options{})
	if result.SessionID != "session:test" || err == nil || !strings.Contains(err.Error(), "is ready") {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	if !reflect.DeepEqual(fake.calls[len(fake.calls)-2], []string{"session", "rename", "session:test", "rexp:demo"}) {
		t.Fatalf("did not mark ready: %v", fake.calls)
	}
}

func TestExecuteMissingRex(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	if _, err := execute(context.Background(), "ls", "--json"); err == nil || !strings.Contains(err.Error(), "not found on PATH") {
		t.Fatalf("unexpected error: %v", err)
	}
}
