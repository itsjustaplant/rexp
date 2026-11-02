package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLocalConfigurationRootsAndIdentity(t *testing.T) {
	dir := t.TempDir()
	path, err := CreateLocalProject("demo", dir)
	if err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `root: "."`) {
		t.Fatalf("not shareable: %s", data)
	}
	for _, root := range []string{`root: "."`, `root: "sub directory"`, ``} {
		content := strings.Replace(string(data), `root: "."`, root, 1)
		if err := os.WriteFile(path, []byte(content), 0600); err != nil {
			t.Fatal(err)
		}
		project, err := LoadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		want := dir
		if root == `root: "sub directory"` {
			want = filepath.Join(dir, "sub directory")
		}
		want, _ = filepath.EvalSymlinks(dir)
		if root == `root: "sub directory"` {
			want = filepath.Join(want, "sub directory")
		}
		if project.Root != want || project.Tabs[0].Layout.Cwd != want || project.SourcePath == "" {
			t.Fatalf("unexpected resolution %+v", project)
		}
	}
	if _, err := CreateLocalProject("another", dir); err == nil {
		t.Fatal("overwrote local config")
	}
	if _, err := LocalFile(dir); err == nil {
		t.Fatal("accepted directory")
	}
	link := filepath.Join(dir, "link.yaml")
	if err := os.Symlink(path, link); err == nil {
		if _, err := LoadFile(link); err == nil {
			t.Fatal("accepted symlink")
		}
	}
	if _, err := DecodeProject([]byte(strings.Replace(string(data), `root: "."`, `root: relative`, 1))); err == nil {
		t.Fatal("named configuration accepted relative root")
	}
}

func TestHookConfigurationValidation(t *testing.T) {
	dir := t.TempDir()
	path, err := CreateLocalProject("demo", dir)
	if err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	for _, test := range []struct {
		yaml  string
		valid bool
	}{
		{"hooks:\n  before_start: [[echo, hello]]\n  after_start: [[echo, ready]]\n  after_close: [[echo, closed]]\n", true},
		{"hooks: {}\n", true},
		{"hooks:\n  before_close: []\n", true},
		{"hooks:\n  before_close: [[echo, goodbye]]\n", false},
		{"hooks:\n  after_start: [[]]\n", false},
		{"hooks:\n  after_start: [[echo, 123]]\n", false},
		{"hooks:\n  after_start: [echo]\n", false},
		{"hooks:\n  other: []\n", false},
		{"hooks: []\n", false},
		{"hooks:\n  before_start: null\n", false},
	} {
		t.Run(test.yaml, func(t *testing.T) {
			_, err := decodeProject(append(append([]byte{}, data...), []byte(test.yaml)...), dir)
			if (err == nil) != test.valid {
				t.Fatalf("valid=%v error=%v", test.valid, err)
			}
		})
	}
}
