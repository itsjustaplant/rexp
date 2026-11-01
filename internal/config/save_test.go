package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestSaveProjectRoundTripAndNoOverwrite(t *testing.T) {
	setHome(t, t.TempDir())
	root := filepath.Join(t.TempDir(), "project with spaces")
	project, err := DecodeProject(projectYAML(root, `      split: columns
      sizes: [51, 49]
      children:
        - pane: editor
          command: [editor, 'a file']
        - pane: shell
          cwd: ./backend
`))
	if err != nil {
		t.Fatal(err)
	}
	path, err := SaveProject(project)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadProject(project.Name)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Root != root || loaded.Tabs[0].Layout.Sizes[0] != 51 || loaded.Tabs[0].Layout.Children[1].Cwd != filepath.Join(root, "backend") || loaded.Tabs[0].Layout.Children[0].Command[1] != "a file" {
		t.Fatalf("workspace changed during save: %+v", loaded)
	}
	if runtime.GOOS != "windows" {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if info.Mode().Perm() != 0o600 {
			t.Fatalf("permissions: %o", info.Mode().Perm())
		}
	}
	before, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	project.Tabs[0].Name = "Different"
	if _, err := SaveProject(project); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("overwrite: %v", err)
	}
	after, err := os.ReadFile(path)
	if err != nil || string(after) != string(before) {
		t.Fatalf("existing file changed: %v", err)
	}
}

func TestSaveProjectRejectsInvalidWorkspace(t *testing.T) {
	setHome(t, t.TempDir())
	if _, err := SaveProject(nil); err == nil {
		t.Fatal("accepted nil project")
	}
	if _, err := SaveProject(&Project{Name: "../escape"}); err == nil {
		t.Fatal("accepted invalid project")
	}
	projects, err := ProjectsDir()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(projects); !os.IsNotExist(err) {
		t.Fatalf("created files before validation: %v", err)
	}
}
