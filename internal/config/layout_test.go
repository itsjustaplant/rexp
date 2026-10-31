package config

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func projectYAML(root, layout string) []byte {
	quoted, _ := json.Marshal(root)
	return []byte(fmt.Sprintf("schema_version: 1\nname: demo\nroot: %s\ntabs:\n  - name: Development\n    layout:\n%s", quoted, layout))
}

func TestNestedLayoutResolvesFocusDirectoriesAndShares(t *testing.T) {
	root := t.TempDir()
	data := projectYAML(root, `      split: columns
      sizes: [50, 50]
      children:
        - pane: editor
          command: [nvim, .]
        - split: rows
          sizes: [34, 33, 33]
          children:
            - pane: server
              cwd: ./backend
              command: [npm, run, dev]
            - pane: logs
              command: [tail, -f, ./logs/app.log]
            - pane: shell
              command: []
`)
	project, err := DecodeProject(data)
	if err != nil {
		t.Fatal(err)
	}
	tab := project.Tabs[0]
	if tab.Focus != "editor" {
		t.Fatalf("default focus = %q", tab.Focus)
	}
	left := tab.Layout.Children[0]
	right := tab.Layout.Children[1]
	if left.Cwd != root || right.Children[0].Cwd != filepath.Join(root, "backend") {
		t.Fatalf("unexpected cwd resolution: %q / %q", left.Cwd, right.Children[0].Cwd)
	}
	if left.Command[0] != "nvim" || len(right.Children[2].Command) != 0 {
		t.Fatal("commands were not preserved")
	}
	if tab.Layout.Shares()[0] != 0.5 || right.Shares()[0] != 0.34 || right.Shares()[2] != 0.33 {
		t.Fatal("incorrect percentage shares")
	}
	right.Sizes = nil
	for _, share := range right.Shares() {
		if math.Abs(share-1.0/3) > 1e-12 {
			t.Fatalf("equal share = %g", share)
		}
	}
}

func TestLayoutValidationErrors(t *testing.T) {
	for _, tc := range []struct{ name, layout, hint string }{
		{"missing node type", "      command: []\n", "exactly one"},
		{"both node types", "      pane: shell\n      split: rows\n", "exactly one"},
		{"invalid direction", "      split: diagonal\n      children: [{pane: a}, {pane: b}]\n", ".split"},
		{"one child", "      split: rows\n      children: [{pane: a}]\n", ".children"},
		{"size count", "      split: rows\n      sizes: [100]\n      children: [{pane: a}, {pane: b}]\n", ".sizes"},
		{"size sum", "      split: rows\n      sizes: [30, 30]\n      children: [{pane: a}, {pane: b}]\n", "sum to 100"},
		{"zero size", "      split: rows\n      sizes: [0, 100]\n      children: [{pane: a}, {pane: b}]\n", ".sizes[0]"},
		{"fractional size", "      split: rows\n      sizes: [50.0, 50]\n      children: [{pane: a}, {pane: b}]\n", ".sizes[0]"},
		{"duplicate pane", "      split: rows\n      children: [{pane: a}, {pane: a}]\n", "duplicate pane"},
		{"pane with children", "      pane: a\n      children: []\n", "cannot define children"},
		{"split with command", "      split: rows\n      command: []\n      children: [{pane: a}, {pane: b}]\n", "cannot define command"},
		{"split with empty cwd", "      split: rows\n      cwd: ''\n      children: [{pane: a}, {pane: b}]\n", "cannot define command or cwd"},
		{"command string", "      pane: a\n      command: npm run dev\n", ".command"},
		{"numeric argument", "      pane: a\n      command: [echo, 42]\n", ".command[1]"},
		{"empty program", "      pane: a\n      command: ['', argument]\n", ".command[0]"},
		{"null command", "      pane: a\n      command: null\n", ".command"},
		{"numeric pane", "      pane: 42\n", ".pane"},
		{"unknown field", "      pane: a\n      widht: 50\n", "widht"},
		{"duplicate key", "      pane: a\n      pane: b\n", "already defined"},
		{"alias", "      split: rows\n      children: [&shell {pane: a}, *shell]\n", "aliases"},
		{"unknown tag", "      pane: !custom a\n", ".pane"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := DecodeProject(projectYAML(t.TempDir(), tc.layout))
			if err == nil || !strings.Contains(err.Error(), tc.hint) {
				t.Fatalf("expected error containing %q, got %v", tc.hint, err)
			}
		})
	}
}

func TestProjectValidation(t *testing.T) {
	valid := string(projectYAML(t.TempDir(), "      pane: shell\n"))
	for _, tc := range []struct{ name, data, hint string }{
		{"version", strings.Replace(valid, "schema_version: 1", "schema_version: 2", 1), "schema_version"},
		{"relative root", strings.Replace(valid, strings.Split(valid, "\n")[2], "root: ./relative", 1), "root must be an absolute"},
		{"empty tabs", strings.Split(valid, "tabs:")[0] + "tabs: []\n", "tabs must contain"},
		{"bad focus", strings.Replace(valid, "    layout:", "    focus: missing\n    layout:", 1), ".focus"},
		{"extra document", valid + "---\n{}\n", "exactly one"},
		{"syntax", "[broken YAML", "decode YAML"},
		{"old template", "name: demo\nroot: /tmp\ntabs: [{name: Terminal, panes: []}]\n", "panes"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := DecodeProject([]byte(tc.data))
			if err == nil || !strings.Contains(err.Error(), tc.hint) {
				t.Fatalf("expected %q, got %v", tc.hint, err)
			}
		})
	}
}

func TestNewProjectTemplateLoads(t *testing.T) {
	setHome(t, t.TempDir())
	root := t.TempDir()
	if _, err := CreateProject("demo", root); err != nil {
		t.Fatal(err)
	}
	project, err := LoadProject("demo")
	if err != nil {
		t.Fatal(err)
	}
	if project.Root != root || project.Tabs[0].Layout.Pane != "shell" {
		t.Fatalf("unexpected starter: %+v", project)
	}
	path, err := ProjectFile("demo")
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	data = []byte(strings.Replace(string(data), `name: "demo"`, `name: "other"`, 1))
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadProject("demo"); err == nil || !strings.Contains(err.Error(), "match filename") {
		t.Fatalf("expected filename mismatch error, got %v", err)
	}
}

func TestFocusAndPaneUniquenessAcrossTabs(t *testing.T) {
	data := string(projectYAML(t.TempDir(), "      pane: editor\n"))
	data = strings.Replace(data, "    layout:", "    focus: editor\n    layout:", 1)
	data += "  - name: Git\n    layout:\n      pane: git\n"
	project, err := DecodeProject([]byte(data))
	if err != nil {
		t.Fatal(err)
	}
	if project.Tabs[0].Focus != "editor" || project.Tabs[1].Focus != "git" {
		t.Fatal("explicit or default focus was lost")
	}
	for _, tc := range []struct{ data, hint string }{
		{strings.Replace(data, "pane: git", "pane: editor", 1), "duplicate pane"},
		{strings.Replace(data, "name: Git", "name: Development", 1), "duplicate tab"},
		{strings.Replace(data, "focus: editor", "focus: git", 1), "does not exist in this tab"},
	} {
		if _, err := DecodeProject([]byte(tc.data)); err == nil || !strings.Contains(err.Error(), tc.hint) {
			t.Fatalf("expected %q, got %v", tc.hint, err)
		}
	}
}
