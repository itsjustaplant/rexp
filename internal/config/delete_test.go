package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDeleteProjectPreservesOtherFilesAndSourceDirectory(t *testing.T) {
	home := t.TempDir()
	setHome(t, home)
	root := t.TempDir()
	source := filepath.Join(root, "source.txt")
	if err := os.WriteFile(source, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	path, err := CreateProject("demo", root)
	if err != nil {
		t.Fatal(err)
	}
	other, err := CreateProject("other", root)
	if err != nil {
		t.Fatal(err)
	}
	deleted, err := DeleteProject("demo")
	if err != nil || deleted != path {
		t.Fatalf("delete returned %q, %v", deleted, err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("configuration still exists: %v", err)
	}
	for _, path := range []string{source, other} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("unrelated file removed: %s: %v", path, err)
		}
	}
	if _, err := DeleteProject("demo"); err == nil {
		t.Fatal("expected error on repeated deletion")
	}
}

func TestDeleteRejectsInvalidNameAndMissingProject(t *testing.T) {
	home := t.TempDir()
	setHome(t, home)
	for _, name := range []string{"", "../escape", `..\escape`, "missing", "NUL"} {
		if _, err := DeleteProject(name); err == nil {
			t.Fatalf("accepted %q", name)
		}
	}
	if _, err := os.Stat(filepath.Join(home, ".rexp")); !os.IsNotExist(err) {
		t.Fatalf("lookup created directories: %v", err)
	}
}

func TestDeleteRejectsDirectory(t *testing.T) {
	home := t.TempDir()
	setHome(t, home)
	path := filepath.Join(home, ".rexp", "projects", "demo.yaml")
	if err := os.MkdirAll(path, 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := DeleteProject("demo"); err == nil {
		t.Fatal("accepted a directory as a configuration")
	}
	if info, err := os.Stat(path); err != nil || !info.IsDir() {
		t.Fatalf("directory removed: %v", err)
	}
}

func TestDeleteRejectsSymbolicLink(t *testing.T) {
	home := t.TempDir()
	setHome(t, home)
	if err := Initialize(); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), "target.yaml")
	if err := os.WriteFile(target, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(home, ".rexp", "projects", "demo.yaml")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := DeleteProject("demo"); err == nil {
		t.Fatal("accepted symbolic link")
	}
	if _, err := os.Lstat(link); err != nil {
		t.Fatalf("link removed: %v", err)
	}
	if got, err := os.ReadFile(target); err != nil || string(got) != "keep" {
		t.Fatalf("target changed: %q, %v", got, err)
	}
}
