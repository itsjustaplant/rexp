package editor

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

func TestParseEditorCommand(t *testing.T) {
	for _, test := range []struct {
		command string
		args    []string
	}{
		{`code --wait`, []string{"code", "--wait"}},
		{`"/Applications/An Editor/bin/editor" --wait`, []string{"/Applications/An Editor/bin/editor", "--wait"}},
		{`'C:\Program Files\Editor\editor.exe' --wait`, []string{`C:\Program Files\Editor\editor.exe`, "--wait"}},
		{`/path\ with\ spaces/editor "a b" ''`, []string{"/path with spaces/editor", "a b", ""}},
		{`editor "a\"b"`, []string{"editor", `a"b`}},
		{`editor '$(touch marker)' ; echo`, []string{"editor", "$(touch marker)", ";", "echo"}},
		{`editor "$HOME"`, []string{"editor", "$HOME"}},
	} {
		got, err := parse(test.command)
		if err != nil || !reflect.DeepEqual(got, test.args) {
			t.Errorf("%q: %v %v", test.command, got, err)
		}
	}
	for _, command := range []string{"", "   ", `''`, `editor "unclosed`, "editor \\", "editor\x00argument"} {
		if _, err := parse(command); err == nil {
			t.Errorf("accepted %q", command)
		}
	}
}

func TestEditorSelectionPrecedence(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("VISUAL", "")
	t.Setenv("EDITOR", "")
	fallback := "vi"
	if runtime.GOOS == "windows" {
		fallback = "notepad.exe"
	}
	assert := func(override string, explicit bool, want string) {
		t.Helper()
		got, err := Resolve(override, explicit)
		if err != nil || got[0] != want {
			t.Fatalf("got %v %v, want %q", got, err, want)
		}
	}
	assert("", false, fallback)
	t.Setenv("EDITOR", "editor --wait")
	assert("", false, "editor")
	t.Setenv("VISUAL", "visual --wait")
	assert("", false, "visual")
	if err := os.MkdirAll(filepath.Join(home, ".rexp"), 0o700); err != nil {
		t.Fatal(err)
	}
	settings := filepath.Join(home, ".rexp", "config.yaml")
	if err := os.WriteFile(settings, []byte("editor: configured --wait\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	assert("", false, "configured")
	assert("explicit --wait", true, "explicit")
	if _, err := Resolve("", true); err == nil {
		t.Fatal("empty explicit override fell back")
	}
	if err := os.WriteFile(settings, []byte("editor: [invalid]\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Resolve("", false); err == nil {
		t.Fatal("invalid preference silently ignored")
	}
	assert("explicit", true, "explicit")
}
