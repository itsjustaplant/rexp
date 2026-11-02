package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/itsjustaplant/rexp/internal/config"
)

func TestLocalNewValidateAndEdit(t *testing.T) {
	editorHome(t)
	dir := t.TempDir()
	t.Chdir(dir)
	editor := fakeEditor(t, "keep")
	cmd := NewCommand("test")
	var output bytes.Buffer
	cmd.SetOut(&output)
	cmd.SetArgs([]string{"new", "demo", "--local", "--edit", "--editor", editor})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, config.LocalFilename)
	if _, err := os.Stat(path); err != nil {
		t.Fatal(err)
	}
	global, _ := config.ProjectsDir()
	if _, err := os.Stat(filepath.Join(global, "demo.yaml")); !os.IsNotExist(err) {
		t.Fatal("registered local config globally")
	}
	for _, args := range [][]string{{"validate", "."}, {"validate", "--file", path}, {"edit", ".", "--editor", editor}, {"edit", "--file", path, "--editor", editor}} {
		cmd := NewCommand("test")
		cmd.SetOut(&output)
		cmd.SetArgs(args)
		if err := cmd.Execute(); err != nil {
			t.Fatalf("%v: %v", args, err)
		}
	}
	t.Chdir(t.TempDir())
	cmd = NewCommand("test")
	cmd.SetOut(&output)
	cmd.SetArgs([]string{"validate", "--file", path})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	for _, command := range []string{"validate", "edit", "start"} {
		cmd := NewCommand("test")
		cmd.SetArgs([]string{command, "demo", "--file", path})
		if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "not both") {
			t.Fatalf("%s accepted ambiguous target: %v", command, err)
		}
	}
	// A bare name always means global, even with a local file in this directory.
	t.Chdir(dir)
	cmd = NewCommand("test")
	cmd.SetArgs([]string{"validate", "demo"})
	if err := cmd.Execute(); err == nil {
		t.Fatal("silently loaded local config for a global name")
	}
}
