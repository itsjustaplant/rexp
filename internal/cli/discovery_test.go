package cli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"rexp/internal/config"
	"rexp/internal/workspace"
)

func discoveryFixture(t *testing.T) {
	t.Helper()
	editorHome(t)
	project := &workspace.Project{SchemaVersion: 1, Name: "Alpha-api", Root: t.TempDir(), Env: map[string]string{"SECRET": "do-not-print-this"}, Hooks: workspace.Hooks{BeforeStart: [][]string{{"must-not-execute"}}}, Tabs: []workspace.Tab{{Name: "Dev", Layout: workspace.Layout{Split: "columns", Children: []workspace.Layout{{Pane: "left"}, {Split: "rows", Children: []workspace.Layout{{Pane: "right-top"}, {Pane: "right-bottom"}}}}}}, {Name: "Other", Layout: workspace.Layout{Pane: "other"}}}}
	if _, err := config.SaveProject(project); err != nil {
		t.Fatal(err)
	}
	if _, err := config.CreateProject("beta-api", t.TempDir()); err != nil {
		t.Fatal(err)
	}
	projects, _ := config.ProjectsDir()
	if err := os.WriteFile(filepath.Join(projects, "broken.yaml"), []byte("invalid: [yaml"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(projects, "ignored.txt"), nil, 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", t.TempDir())
}

func runDiscovery(t *testing.T, args ...string) (string, error) {
	t.Helper()
	var out bytes.Buffer
	cmd := NewCommand("test")
	cmd.SetOut(&out)
	cmd.SetErr(&out)
	cmd.SetArgs(args)
	err := cmd.Execute()
	return out.String(), err
}

func TestSearchNamesAndEmptyResults(t *testing.T) {
	discoveryFixture(t)
	for _, test := range []struct{ query, want string }{
		{"API", "Alpha-api\nbeta-api\n"}, {"PHA", "Alpha-api\n"}, {"broken", "broken\n"}, {"no-match", "No projects matched.\n"}, {"must-not-execute", "No projects matched.\n"},
	} {
		out, err := runDiscovery(t, "search", test.query)
		if err != nil || out != test.want {
			t.Fatalf("query=%q output=%q err=%v", test.query, out, err)
		}
	}
	for _, args := range [][]string{{"search"}, {"search", ""}, {"search", " "}, {"search", "one", "two"}, {"search", "api", "--long", "--json"}, {"list", "--long", "--json"}} {
		if _, err := runDiscovery(t, args...); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
}

func TestListAndSearchJSONMetadata(t *testing.T) {
	discoveryFixture(t)
	out, err := runDiscovery(t, "list", "--json")
	if err != nil {
		t.Fatal(err)
	}
	var summaries []config.ProjectSummary
	if err := json.Unmarshal([]byte(out), &summaries); err != nil {
		t.Fatalf("invalid JSON: %v %q", err, out)
	}
	if len(summaries) != 3 {
		t.Fatalf("entries: %+v", summaries)
	}
	first := summaries[0]
	if first.Name != "Alpha-api" || !first.Valid || first.Tabs != 2 || first.Panes != 4 || !filepath.IsAbs(first.Root) || !filepath.IsAbs(first.Path) {
		t.Fatalf("metadata %+v", first)
	}
	broken := summaries[2]
	if broken.Name != "broken" || broken.Valid || broken.Error == "" || broken.Tabs != 0 || broken.Root != "" {
		t.Fatalf("invalid entry %+v", broken)
	}
	if strings.Contains(out, "do-not-print-this") || strings.Contains(out, "must-not-execute") {
		t.Fatal("leaked env/commands into metadata")
	}
	for _, args := range [][]string{{"search", "API", "--json"}, {"search", "BROKEN", "--json"}} {
		out, err := runDiscovery(t, args...)
		if err != nil {
			t.Fatal(err)
		}
		if err := json.Unmarshal([]byte(out), &summaries); err != nil {
			t.Fatal(err)
		}
		if strings.Contains(out, "ignored") {
			t.Fatal("included unrelated file")
		}
	}
	out, err = runDiscovery(t, "search", "absent", "--json")
	if err != nil || strings.TrimSpace(out) != "[]" {
		t.Fatalf("empty results: %q %v", out, err)
	}
}

func TestDetailedOutputRetainsInvalidEntries(t *testing.T) {
	discoveryFixture(t)
	out, err := runDiscovery(t, "list", "--long")
	if err != nil {
		t.Fatal(err)
	}
	for _, word := range []string{"NAME", "ROOT", "TABS", "PANES", "STATUS", "PATH", "ERROR", "Alpha-api", "beta-api", "broken", "invalid"} {
		if !strings.Contains(out, word) {
			t.Fatalf("missing %q from %s", word, out)
		}
	}
	out, err = runDiscovery(t, "search", "broken", "--long")
	if err != nil || !strings.Contains(out, "invalid") || strings.Contains(out, "beta-api") {
		t.Fatalf("filtered details %q %v", out, err)
	}
	out, err = runDiscovery(t, "list")
	if err != nil || out != "Alpha-api\nbeta-api\nbroken\n" {
		t.Fatalf("changed plain list: %q %v", out, err)
	}
}

func TestEmptyJSONAndEscapedTableFields(t *testing.T) {
	editorHome(t)
	out, err := runDiscovery(t, "list", "--json")
	if err != nil || strings.TrimSpace(out) != "[]" {
		t.Fatalf("empty JSON %q %v", out, err)
	}
	input := "/path\twith\ncontrols\x1b[31m"
	if got := tableText(input); strings.ContainsAny(got, "\n\t\x1b") {
		t.Fatalf("unescaped table value %q", got)
	}
}
