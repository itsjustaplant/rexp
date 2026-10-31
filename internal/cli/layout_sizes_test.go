package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"testing"

	"rexp/internal/config"
)

func TestValidateAndStartRejectInvalidSizes(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	// A validation error must be returned before start tries to execute Rex.
	t.Setenv("PATH", t.TempDir())
	root := t.TempDir()
	quotedRoot, err := json.Marshal(root)
	if err != nil {
		t.Fatal(err)
	}
	path, err := config.CreateProject("demo", root)
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name, sizes, children, message string
	}{
		{"sum 101", "[51, 50]", "[{pane: a}, {pane: b}]", "sizes must sum to 100, got 101"},
		{"sum 99", "[33, 33, 33]", "[{pane: a}, {pane: b}, {pane: c}]", "sizes must sum to 100, got 99"},
		{"three sizes for two children", "[34, 33, 33]", "[{pane: a}, {pane: b}]", "sizes must have one percentage per child"},
	} {
		t.Run(test.name, func(t *testing.T) {
			// Put the invalid split after a valid pane, so start must finish
			// validating the whole tree before launching any part of it.
			data := fmt.Sprintf(`schema_version: 1
name: demo
root: %s
tabs:
  - name: Development
    layout:
      split: columns
      children:
        - pane: editor
        - split: rows
          sizes: %s
          children: %s
`, quotedRoot, test.sizes, test.children)
			if err := os.WriteFile(path, []byte(data), 0o600); err != nil {
				t.Fatal(err)
			}
			for _, command := range []string{"validate", "start"} {
				t.Run(command, func(t *testing.T) {
					var output bytes.Buffer
					cmd := NewCommand("test")
					cmd.SetOut(&output)
					cmd.SetArgs([]string{command, "demo"})
					err := cmd.Execute()
					if err == nil || !strings.Contains(err.Error(), "tabs[0].layout.children[1]."+test.message) {
						t.Fatalf("expected nested size validation error %q, got %v", test.message, err)
					}
					if output.Len() != 0 {
						t.Fatalf("unexpected success output: %q", output.String())
					}
				})
			}
		})
	}
}
