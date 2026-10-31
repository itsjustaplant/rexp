package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestProjectNameValidation(t *testing.T) {
	for _, name := range []string{"app", "My-App_2", "7", strings.Repeat("a", 64)} {
		if err := ValidateProjectName(name); err != nil {
			t.Errorf("valid name %q: %v", name, err)
		}
	}
	for _, name := range []string{"", ".", "..", "../escape", `..\escape`, "/absolute", "a/b", `a\b`, "a:b", "a b", "a.yaml", "-app", "_app", "app\n", "é", strings.Repeat("a", 65), "CON", "con", "PRN", "AUX", "NUL", "COM1", "LPT9"} {
		if err := ValidateProjectName(name); err == nil {
			t.Errorf("accepted invalid name %q", name)
		}
	}
}

func TestCreateProjectCapturesRootAndPreservesExistingConfig(t *testing.T) {
	home := t.TempDir()
	setHome(t, home)
	root := filepath.Join(t.TempDir(), "project with spaces")
	if err := os.Mkdir(root, 0o700); err != nil {
		t.Fatal(err)
	}
	path, err := CreateProject("my-app", root)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(home, ".rexp", "projects", "my-app.yaml"); path != want {
		t.Fatalf("path = %q, want %q", path, want)
	}
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	for _, line := range strings.Split(string(content), "\n") {
		if strings.HasPrefix(line, "root: ") {
			var got string
			if err := json.Unmarshal([]byte(strings.TrimPrefix(line, "root: ")), &got); err != nil || got != root {
				t.Fatalf("root did not round-trip: %q, error %v", got, err)
			}
		}
	}
	if !strings.Contains(string(content), `name: "my-app"`) || !strings.Contains(string(content), "\nroot: ") || !strings.Contains(string(content), "command: []") {
		t.Fatalf("missing starter fields: %s", content)
	}
	if err := os.WriteFile(path, []byte("my edited configuration\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := CreateProject("my-app", root); err == nil || !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("expected duplicate error, got %v", err)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != "my edited configuration\n" {
		t.Fatalf("existing configuration overwritten: %q, error %v", got, err)
	}
}

func TestCreateProjectRejectsInvalidInputBeforeCreatingFiles(t *testing.T) {
	for _, input := range []struct{ name, root string }{
		{"../escape", t.TempDir()},
		{"app", "relative/path"},
		{"app", filepath.Join(t.TempDir(), "missing")},
	} {
		home := t.TempDir()
		setHome(t, home)
		if _, err := CreateProject(input.name, input.root); err == nil {
			t.Fatalf("accepted invalid input %+v", input)
		}
		if _, err := os.Stat(filepath.Join(home, ".rexp")); !os.IsNotExist(err) {
			t.Fatalf("invalid input created configuration directory: %v", err)
		}
	}
}
