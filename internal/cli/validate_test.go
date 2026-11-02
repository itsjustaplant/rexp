package cli

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/itsjustaplant/rexp/internal/config"
)

func TestValidateCommand(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	path, err := config.CreateProject("demo", t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	var output bytes.Buffer
	cmd := NewCommand("test")
	cmd.SetOut(&output)
	cmd.SetArgs([]string{"validate", "demo"})
	if err := cmd.Execute(); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output.String(), `Project "demo" is valid`) {
		t.Fatalf("unexpected output: %q", output.String())
	}
	if err := os.WriteFile(path, []byte("schema_version: 999\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cmd = NewCommand("test")
	cmd.SetArgs([]string{"validate", "demo"})
	if err := cmd.Execute(); err == nil || !strings.Contains(err.Error(), "schema_version") {
		t.Fatalf("expected schema error, got %v", err)
	}
	for _, args := range [][]string{{"validate"}, {"validate", "one", "two"}, {"validate", "../escape"}, {"validate", "missing"}} {
		cmd = NewCommand("test")
		cmd.SetArgs(args)
		if err := cmd.Execute(); err == nil {
			t.Fatalf("accepted arguments: %v", args)
		}
	}
}
