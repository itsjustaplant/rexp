package config

import (
	"strings"
	"testing"
)

func TestEnvironmentTypesAndScopes(t *testing.T) {
	for _, test := range []struct {
		name, env string
		valid     bool
	}{
		{"strings", "{PORT: '3000', EMPTY: '', command: literal}", true},
		{"empty", "{}", true},
		{"number", "{PORT: 3000}", false},
		{"bool", "{DEBUG: true}", false},
		{"null", "{EMPTY: null}", false},
		{"list", "[PORT, value]", false},
		{"invalid name", "{'BAD-NAME': value}", false},
		{"leading digit", "{'1PORT': value}", false},
		{"NUL", `{VALUE: "a\0b"}`, false},
		{"duplicate", "{PORT: '1', PORT: '2'}", false},
	} {
		for _, scope := range []string{"project", "tab", "pane"} {
			t.Run(test.name+"/"+scope, func(t *testing.T) {
				data := string(projectYAML(t.TempDir(), "      pane: shell\n"))
				switch scope {
				case "project":
					data = strings.Replace(data, "tabs:\n", "env: "+test.env+"\ntabs:\n", 1)
				case "tab":
					data = strings.Replace(data, "    layout:", "    env: "+test.env+"\n    layout:", 1)
				case "pane":
					data += "      env: " + test.env + "\n"
				}
				if _, err := DecodeProject([]byte(data)); (err == nil) != test.valid {
					t.Fatalf("valid=%v error=%v", test.valid, err)
				}
			})
		}
	}
	data := projectYAML(t.TempDir(), "      split: columns\n      env: {}\n      children: [{pane: a}, {pane: b}]\n")
	if _, err := DecodeProject(data); err == nil || !strings.Contains(err.Error(), "not a split") {
		t.Fatalf("accepted split env: %v", err)
	}
}

func TestEnvironmentAndCommandsSaveRoundTrip(t *testing.T) {
	setHome(t, t.TempDir())
	project, err := DecodeProject(projectYAML(t.TempDir(), "      pane: shell\n      env: {PORT: '3000'}\n      commands: [[echo, first], [echo, second]]\n"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := SaveProject(project); err != nil {
		t.Fatal(err)
	}
	loaded, err := LoadProject(project.Name)
	if err != nil || loaded.Tabs[0].Layout.Env["PORT"] != "3000" || len(loaded.Tabs[0].Layout.Commands) != 2 {
		t.Fatalf("round trip: %+v %v", loaded, err)
	}
}
