package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEditorPreferenceValidation(t *testing.T) {
	setHome(t, t.TempDir())
	if got, err := EditorPreference(); err != nil || got != "" {
		t.Fatalf("missing preferences: %q %v", got, err)
	}
	projects, _ := ProjectsDir()
	if _, err := os.Stat(filepath.Dir(projects)); !os.IsNotExist(err) {
		t.Fatal("reading preferences created directories")
	}
	if err := Initialize(); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(filepath.Dir(projects), "config.yaml")
	for _, test := range []struct {
		data, want string
		valid      bool
	}{
		{"", "", true}, {"{}", "", true}, {"editor: code --wait\n", "code --wait", true},
		{"editor: 42\n", "", false}, {"editor: null\n", "", false}, {"editor: [code]\n", "", false},
		{"unknown: value\n", "", false}, {"editor: a\neditor: b\n", "", false}, {"[]", "", false},
		{"editor: a\n---\neditor: b\n", "", false}, {"editor: [broken", "", false},
	} {
		if err := os.WriteFile(path, []byte(test.data), 0o600); err != nil {
			t.Fatal(err)
		}
		got, err := EditorPreference()
		if (err == nil) != test.valid || err == nil && got != test.want {
			t.Errorf("%q: got %q %v", test.data, got, err)
		}
	}
}
