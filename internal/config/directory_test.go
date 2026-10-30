package config

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func setHome(t *testing.T, home string) {
	t.Helper()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
}

func TestInitializeCreatesHomeDirectories(t *testing.T) {
	home := t.TempDir()
	setHome(t, home)
	if err := Initialize(); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{filepath.Join(home, ".rexp"), filepath.Join(home, ".rexp", "projects")} {
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if !info.IsDir() {
			t.Fatalf("%s is not a directory", path)
		}
		if runtime.GOOS != "windows" && info.Mode().Perm()&0o077 != 0 {
			t.Fatalf("%s permissions allow group or other access: %v", path, info.Mode())
		}
	}
}

func TestInitializePreservesExistingFiles(t *testing.T) {
	home := t.TempDir()
	setHome(t, home)
	if err := Initialize(); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(home, ".rexp", "projects", "existing.yaml")
	want := "name: existing\n"
	if err := os.WriteFile(path, []byte(want), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := Initialize(); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil || string(got) != want {
		t.Fatalf("existing file changed: %q, error: %v", got, err)
	}
}

func TestInitializeRepairsMissingProjectsDirectory(t *testing.T) {
	home := t.TempDir()
	setHome(t, home)
	if err := os.Mkdir(filepath.Join(home, ".rexp"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := Initialize(); err != nil {
		t.Fatal(err)
	}
	if info, err := os.Stat(filepath.Join(home, ".rexp", "projects")); err != nil || !info.IsDir() {
		t.Fatalf("projects directory missing: %v", err)
	}
}

func TestInitializeReportsFileConflicts(t *testing.T) {
	for _, conflict := range []string{".rexp", filepath.Join(".rexp", "projects")} {
		t.Run(conflict, func(t *testing.T) {
			home := t.TempDir()
			setHome(t, home)
			path := filepath.Join(home, conflict)
			if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
				t.Fatal(err)
			}
			if err := os.WriteFile(path, []byte("preserve me"), 0o600); err != nil {
				t.Fatal(err)
			}
			if err := Initialize(); err == nil || !strings.Contains(err.Error(), "initialize rexp directory") {
				t.Fatalf("expected initialization error, got %v", err)
			}
			got, err := os.ReadFile(path)
			if err != nil || string(got) != "preserve me" {
				t.Fatalf("conflicting file changed: %q, error: %v", got, err)
			}
		})
	}
}

func TestInitializeReportsMissingHome(t *testing.T) {
	setHome(t, "")
	if err := Initialize(); err == nil || !strings.Contains(err.Error(), "find user home directory") {
		t.Fatalf("expected home lookup error, got %v", err)
	}
}
