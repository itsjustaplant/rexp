package cli

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"rexp/internal/config"
)

func TestStartRejectsInvalidInputBeforeRex(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("PATH", t.TempDir())
	path, err := config.CreateProject("invalid", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("schema_version: 999\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		args    []string
		message string
	}{
		{[]string{"start"}, "missing project name"},
		{[]string{"start", "a", "b"}, "accepts 1 arg"},
		{[]string{"start", "../escape"}, "project name"},
		{[]string{"start", "missing"}, "does not exist"},
		{[]string{"start", "invalid"}, "schema_version"},
		{[]string{"start", "invalid", "--background", "--client", "some-client"}, "[background client] were all set"},
	} {
		cmd := NewCommand("test")
		cmd.SetArgs(test.args)
		if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), test.message) {
			t.Errorf("%v: got %v, want %q", test.args, err, test.message)
		}
	}
}

func TestStartHelp(t *testing.T) {
	var output bytes.Buffer
	cmd := NewCommand("test")
	cmd.SetOut(&output)
	cmd.SetArgs([]string{"start", "--help"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{"start <project>", "--background", "--client"} {
		if !strings.Contains(output.String(), expected) {
			t.Errorf("help missing %q: %s", expected, output.String())
		}
	}
}
