package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestListProjectsFiltersAndSortsWithoutParsingYAML(t *testing.T) {
	home := t.TempDir()
	setHome(t, home)
	if err := Initialize(); err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(home, ".rexp", "projects")
	for _, name := range []string{"zebra.yaml", "alpha.yaml", "alpha-beta.yaml", "middle.yaml", "notes.txt", "other.yml", "bad name.yaml", ".yaml", "CON.yaml"} {
		if name == "CON.yaml" && os.PathSeparator == '\\' {
			continue // Windows does not allow creating this reserved filename.
		}
		if err := os.WriteFile(filepath.Join(dir, name), []byte("[invalid YAML on purpose"), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.Mkdir(filepath.Join(dir, "directory.yaml"), 0o700); err != nil {
		t.Fatal(err)
	}
	got, err := ListProjects()
	if err != nil {
		t.Fatal(err)
	}
	if want := []string{"alpha", "alpha-beta", "middle", "zebra"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("names = %v, want %v", got, want)
	}
}

func TestListProjectsEmptyAndMissingDirectory(t *testing.T) {
	home := t.TempDir()
	setHome(t, home)
	for _, initialized := range []bool{false, true} {
		if initialized {
			if err := Initialize(); err != nil {
				t.Fatal(err)
			}
		}
		names, err := ListProjects()
		if err != nil || len(names) != 0 {
			t.Fatalf("expected empty list, got %v, %v", names, err)
		}
		if !initialized {
			if _, err := os.Stat(filepath.Join(home, ".rexp")); !os.IsNotExist(err) {
				t.Fatalf("listing unexpectedly created directories: %v", err)
			}
		}
	}
}

func TestListProjectsRejectsUnreadableDirectoryPath(t *testing.T) {
	home := t.TempDir()
	setHome(t, home)
	if err := os.Mkdir(filepath.Join(home, ".rexp"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(home, ".rexp", "projects"), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := ListProjects(); err == nil || !strings.Contains(err.Error(), "list project configurations") {
		t.Fatalf("expected directory error, got %v", err)
	}
}

func TestListProjectsIgnoresSymbolicLinks(t *testing.T) {
	home := t.TempDir()
	setHome(t, home)
	if err := Initialize(); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "target.yaml")
	if err := os.WriteFile(target, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(home, ".rexp", "projects", "linked.yaml")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if names, err := ListProjects(); err != nil || len(names) != 0 {
		t.Fatalf("expected symlink to be ignored, got %v, %v", names, err)
	}
}
