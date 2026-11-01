package config

import (
	"strings"
	"testing"
)

func TestInitialTabSelection(t *testing.T) {
	base := string(projectYAML(t.TempDir(), "      pane: editor\n")) + "  - name: Server\n    layout:\n      pane: server\n"
	for _, test := range []struct{ label, data, want, hint string }{
		{"default", base, "Development", ""},
		{"second", strings.Replace(base, "  - name: Server", "  - name: Server\n    selected: true", 1), "Server", ""},
		{"false", strings.Replace(base, "  - name: Server", "  - name: Server\n    selected: false", 1), "Development", ""},
		{"multiple", strings.ReplaceAll(base, "    layout:", "    selected: true\n    layout:"), "", "at most one"},
		{"string", strings.Replace(base, "  - name: Server", "  - name: Server\n    selected: 'true'", 1), "", "selected"},
		{"number", strings.Replace(base, "  - name: Server", "  - name: Server\n    selected: 1", 1), "", "selected"},
		{"null", strings.Replace(base, "  - name: Server", "  - name: Server\n    selected: null", 1), "", "selected"},
	} {
		t.Run(test.label, func(t *testing.T) {
			project, err := DecodeProject([]byte(test.data))
			if test.hint != "" {
				if err == nil || !strings.Contains(err.Error(), test.hint) {
					t.Fatalf("expected %q, got %v", test.hint, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if project.InitialTab().Name != test.want {
				t.Fatalf("initial tab: %s", project.InitialTab().Name)
			}
		})
	}
}
