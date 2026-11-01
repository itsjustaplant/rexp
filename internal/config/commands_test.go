package config

import (
	"strings"
	"testing"
)

func TestCommandSequences(t *testing.T) {
	for _, test := range []struct{ name, fields, hint string }{
		{"sequence", "commands: [[npm, run, build], [npm, run, dev]]", ""},
		{"empty sequence", "commands: []", ""},
		{"both", "command: []\n      commands: []", "mutually exclusive"},
		{"empty command", "commands: [[]]", "must contain a program"},
		{"blank program", "commands: [[' ', arg]]", "must name a program"},
		{"flat", "commands: [echo, hello]", "array of strings"},
		{"string", "commands: echo hi", "invalid value type"},
		{"number", "commands: [[echo, 42]]", "invalid value type"},
		{"null", "commands: [null]", "array of strings"},
		{"split", "split: rows\n      children: [{pane: a}, {pane: b}]\n      commands: []", "split cannot define commands"},
	} {
		t.Run(test.name, func(t *testing.T) {
			layout := "      pane: shell\n      " + test.fields + "\n"
			if test.name == "split" {
				layout = "      " + test.fields + "\n"
			}
			project, err := DecodeProject(projectYAML(t.TempDir(), layout))
			if test.hint != "" {
				if err == nil || !strings.Contains(err.Error(), test.hint) {
					t.Fatalf("expected %q, got %v", test.hint, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if test.name == "sequence" && len(project.Tabs[0].Layout.Commands) != 2 {
				t.Fatal("sequence lost")
			}
		})
	}
}
