package rex

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/itsjustaplant/rexp/internal/config"
)

const exportFixture = `{
 "session_id":"session:source", "active_window_id":"window:2",
 "windows":[
  {"window_id":"window:1","label":"Development","focused_block_id":"block:3",
   "layers":[{"kind":"tiled","layout":{"split":{"direction":"horizontal","ratio":0.5,
    "before":{"block_id":"block:1"},"after":{"split":{"direction":"vertical","ratio":0.6666666666666666,
     "before":{"block_id":"block:2"},"after":{"block_id":"block:3"}}}}},
    "blocks":[
     {"block_id":"block:1","label":"shell","creator_name":"com.superlogical.terminal","host_id":"host:1"},
     {"block_id":"block:2","label":"shell","creator_name":"com.superlogical.terminal","host_id":"host:1"},
     {"block_id":"block:3","label":"Not a valid identifier","creator_name":"com.superlogical.terminal","host_id":"host:1"}]}]},
  {"window_id":"window:2","label":"Development","focused_block_id":"block:4",
   "layers":[{"kind":"tiled","layout":{"block_id":"block:4"},"blocks":[
    {"block_id":"block:4","label":"CON","creator_name":"com.superlogical.terminal","host_id":"host:1"}]}]}
 ]}`

func exportFake(t *testing.T, fixture string) (*Client, *[][]string) {
	t.Helper()
	calls := [][]string{}
	dir := t.TempDir()
	process, err := json.Marshal(map[string]any{"foreground": map[string]string{"cwd": dir, "argv0": "npm"}, "last_exit": nil})
	if err != nil {
		t.Fatal(err)
	}
	return &Client{run: func(_ context.Context, args ...string) ([]byte, error) {
		calls = append(calls, append([]string(nil), args...))
		if len(args) > 1 && args[0] == "session" && args[1] == "inspect" {
			return []byte(fixture), nil
		}
		if len(args) > 1 && args[0] == "block" && args[1] == "call" && args[len(args)-1] == "process" {
			return process, nil
		}
		return nil, errors.New("unexpected Rex call")
	}}, &calls
}

func TestExportRoundTripsTabsLayoutNamesAndFocus(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	client, calls := exportFake(t, exportFixture)
	result, err := client.Export(context.Background(), "session:source", "demo", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if len(*calls) != 6 || !reflect.DeepEqual((*calls)[0], []string{"session", "inspect", "--json", "--", "session:source"}) {
		t.Fatalf("calls: %v", *calls)
	}
	tabs := result.Project.Tabs
	if tabs[0].Selected || !tabs[1].Selected {
		t.Fatal("active tab was not preserved")
	}
	if len(tabs) != 2 || tabs[1].Name != "Development (2)" || tabs[0].Focus != "pane-3" || tabs[1].Focus != "pane-4" {
		t.Fatalf("tabs: %+v", tabs)
	}
	left, right := tabs[0].Layout.Children[0], tabs[0].Layout.Children[1]
	if tabs[0].Layout.Split != "columns" || right.Split != "rows" || right.Sizes[0] != 67 || right.Sizes[1] != 33 || left.Pane != "shell" || right.Children[0].Pane != "shell-2" {
		t.Fatalf("layout: %+v", tabs[0].Layout)
	}
	if len(left.Command) != 0 || !filepath.IsAbs(left.Cwd) {
		t.Fatalf("guessed a command or omitted cwd: %+v", left)
	}
	if len(result.Warnings) < 5 {
		t.Fatalf("expected warnings for renaming, active tab, and command omission: %v", result.Warnings)
	}
	if _, err := config.SaveProject(result.Project); err != nil {
		t.Fatal(err)
	}
	if loaded, err := config.LoadProject("demo"); err != nil {
		t.Fatalf("export is not loadable: %v", err)
	} else if loaded.InitialTab().Name != tabs[1].Name {
		t.Fatal("selected tab did not round trip")
	}
}

func TestExportRejectsUnsupportedOrInconsistentSnapshots(t *testing.T) {
	for _, test := range []struct{ name, old, new, hint string }{
		{"floating", "\"kind\":\"tiled\"", "\"kind\":\"floating\"", "unsupported layers"},
		{"missing ratio", "\"ratio\":0.5,", "", "invalid split ratio"},
		{"zero ratio", "\"ratio\":0.5", "\"ratio\":0", "invalid split ratio"},
		{"narrow split", "\"ratio\":0.5", "\"ratio\":0.0001", "too narrow"},
		{"direction", "\"direction\":\"horizontal\"", "\"direction\":\"diagonal\"", "unsupported split direction"},
		{"nonterminal", "\"creator_name\":\"com.superlogical.terminal\"", "\"creator_name\":\"other\"", "unsupported creator"},
		{"missing node", "\"before\":{\"block_id\":\"block:1\"}", "\"before\":null", "missing layout node"},
		{"duplicate block", "\"after\":{\"block_id\":\"block:3\"}", "\"after\":{\"block_id\":\"block:2\"}", "missing or occurs more than once"},
		{"missing focus", "\"focused_block_id\":\"block:3\"", "\"focused_block_id\":\"block:missing\"", "focuses a pane outside"},
		{"missing active tab", "\"active_window_id\":\"window:2\"", "\"active_window_id\":\"window:missing\"", "active tab is missing"},
	} {
		t.Run(test.name, func(t *testing.T) {
			fixture := strings.Replace(exportFixture, test.old, test.new, 1)
			if fixture == exportFixture {
				t.Fatal("fixture not changed")
			}
			client, _ := exportFake(t, fixture)
			if _, err := client.Export(context.Background(), "session:source", "demo", t.TempDir()); err == nil || !strings.Contains(err.Error(), test.hint) {
				t.Fatalf("expected %q, got %v", test.hint, err)
			}
		})
	}
	client, _ := exportFake(t, strings.Replace(exportFixture, `"host_id":"host:1"`, `"host_id":"host:other"`, 1))
	if _, err := client.Export(context.Background(), "session:source", "demo", t.TempDir()); err == nil || !strings.Contains(err.Error(), "different hosts") {
		t.Fatalf("mixed hosts: %v", err)
	}
}

func TestExportDetectsChangesDuringRead(t *testing.T) {
	client, calls := exportFake(t, exportFixture)
	run := client.run
	client.run = func(ctx context.Context, args ...string) ([]byte, error) {
		if len(*calls) > 1 && args[0] == "session" {
			return []byte(strings.Replace(exportFixture, `"ratio":0.5`, `"ratio":0.6`, 1)), nil
		}
		return run(ctx, args...)
	}
	if _, err := client.Export(context.Background(), "session:source", "demo", t.TempDir()); err == nil || !strings.Contains(err.Error(), "changed while saving") {
		t.Fatalf("accepted changed layout: %v", err)
	}
}

func TestExportDirectorySources(t *testing.T) {
	dir := t.TempDir()
	quoted, _ := json.Marshal(dir)
	for _, test := range []struct {
		name, process, pwd, hint string
		reported                 bool
	}{
		{"foreground", `{"foreground":{"cwd":` + string(quoted) + `},"child":{"cwd":"bad"}}`, "", "", false},
		{"ancestor", `{"foreground":{},"ancestors":[{"cwd":` + string(quoted) + `}]}`, "", "", false},
		{"child", `{"child":{"cwd":` + string(quoted) + `}}`, "", "", false},
		{"reported", `{}`, `{"pwd":` + string(quoted) + `}`, "", true},
		{"unavailable", `{}`, `{}`, "no absolute pane directory", false},
		{"relative", `{"foreground":{"cwd":"relative/path"}}`, "", "not an absolute native path", false},
	} {
		t.Run(test.name, func(t *testing.T) {
			client := &Client{run: func(_ context.Context, args ...string) ([]byte, error) {
				if args[len(args)-1] == "process" {
					return []byte(test.process), nil
				}
				return []byte(test.pwd), nil
			}}
			got, reported, err := client.exportDirectory(context.Background(), "session:source", "block:1")
			if test.hint != "" {
				if err == nil || !strings.Contains(err.Error(), test.hint) {
					t.Fatalf("got %v", err)
				}
				return
			}
			if err != nil || got != dir || reported != test.reported {
				t.Fatalf("cwd=%q reported=%v err=%v", got, reported, err)
			}
		})
	}
}

func TestExportDirectoryReportedURL(t *testing.T) {
	if os.PathSeparator != '/' {
		t.Skip("POSIX file URL")
	}
	client := &Client{run: func(_ context.Context, args ...string) ([]byte, error) {
		if args[len(args)-1] == "process" {
			return []byte(`{}`), nil
		}
		return []byte(`{"pwd":"file://localhost/tmp/project%20with%20spaces"}`), nil
	}}
	got, reported, err := client.exportDirectory(context.Background(), "s", "b")
	if err != nil || !reported || got != "/tmp/project with spaces" {
		t.Fatalf("cwd=%q reported=%v err=%v", got, reported, err)
	}
}
