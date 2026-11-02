package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"go.yaml.in/yaml/v3"

	"rexp/internal/workspace"
)

// Aliases preserve the configuration API while keeping the model parser-independent.
type Project = workspace.Project
type Tab = workspace.Tab
type Layout = workspace.Layout

// LoadProject reads and validates a named file. It never launches a command.
func LoadProject(name string) (*Project, error) {
	path, err := ProjectFile(name)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read project configuration %q: %w", path, err)
	}
	project, err := DecodeProject(data)
	if err != nil {
		return nil, fmt.Errorf("project configuration %q: %w", path, err)
	}
	if project.Name != name {
		return nil, fmt.Errorf("project configuration %q: name must match filename %q", path, name)
	}
	return project, nil
}

// DecodeProject validates YAML and resolves pane directories and default focus.
// Root and cwd paths are not required to exist until the workspace is launched.
func DecodeProject(data []byte) (*Project, error) {
	return decodeProject(data, "")
}

func decodeProject(data []byte, directory string) (*Project, error) {
	var document yaml.Node
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	if err := decoder.Decode(&document); err != nil {
		return nil, fmt.Errorf("decode YAML: %w", err)
	}
	var extra yaml.Node
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err != nil {
			return nil, fmt.Errorf("decode YAML: %w", err)
		}
		return nil, errors.New("configuration must contain exactly one YAML document")
	}
	if err := checkYAMLTypes(&document, "configuration", 0); err != nil {
		return nil, err
	}
	var project Project
	strict := yaml.NewDecoder(bytes.NewReader(data))
	strict.KnownFields(true)
	if err := strict.Decode(&project); err != nil {
		return nil, fmt.Errorf("decode configuration: %w", err)
	}
	if project.SchemaVersion != 1 {
		return nil, errors.New("schema_version must be 1; configurations using the old tabs[].panes template must be updated to tabs[].layout")
	}
	if err := ValidateProjectName(project.Name); err != nil {
		return nil, fmt.Errorf("name: %w", err)
	}
	if directory != "" && !filepath.IsAbs(project.Root) {
		project.Root = filepath.Join(directory, project.Root)
	}
	if !filepath.IsAbs(project.Root) {
		return nil, errors.New("root must be an absolute directory path")
	}
	if strings.ContainsRune(project.Root, 0) {
		return nil, errors.New("root must not contain NUL")
	}
	project.Root = filepath.Clean(project.Root)
	if err := workspace.ValidateEnvironment(project.Env, "env"); err != nil {
		return nil, err
	}
	if len(project.Hooks.BeforeClose) > 0 {
		return nil, errors.New("hooks.before_close is not implemented: Rex exposes only an after-close event")
	}
	for phase, commands := range map[string][][]string{"before_start": project.Hooks.BeforeStart, "after_start": project.Hooks.AfterStart, "after_close": project.Hooks.AfterClose} {
		for i, command := range commands {
			if err := workspace.ValidateCommand(command, fmt.Sprintf("hooks.%s[%d]", phase, i)); err != nil {
				return nil, err
			}
		}
	}
	if len(project.Tabs) == 0 {
		return nil, errors.New("tabs must contain at least one tab")
	}
	panes := map[string]bool{}
	tabs := map[string]bool{}
	selected := false
	for i := range project.Tabs {
		tab := &project.Tabs[i]
		path := fmt.Sprintf("tabs[%d]", i)
		if err := workspace.ValidateEnvironment(tab.Env, path+".env"); err != nil {
			return nil, err
		}
		if tab.Selected {
			if selected {
				return nil, fmt.Errorf("%s.selected: at most one tab may be selected", path)
			}
			selected = true
		}
		if strings.TrimSpace(tab.Name) == "" {
			return nil, fmt.Errorf("%s.name must not be empty", path)
		}
		if tabs[tab.Name] {
			return nil, fmt.Errorf("%s.name: duplicate tab name %q", path, tab.Name)
		}
		tabs[tab.Name] = true
		local := map[string]bool{}
		first, err := validateLayout(&tab.Layout, path+".layout", project.Root, panes, local, 0)
		if err != nil {
			return nil, err
		}
		if tab.Focus == "" {
			tab.Focus = first
		} else if !local[tab.Focus] {
			return nil, fmt.Errorf("%s.focus: pane %q does not exist in this tab", path, tab.Focus)
		}
	}
	return &project, nil
}

func validateLayout(layout *Layout, path, root string, all, local map[string]bool, depth int) (string, error) {
	if depth > 32 {
		return "", fmt.Errorf("%s: layout nesting exceeds 32 levels", path)
	}
	if (layout.Pane == "") == (layout.Split == "") {
		return "", fmt.Errorf("%s must define exactly one of pane or split", path)
	}
	if layout.Pane != "" {
		if err := workspace.ValidateEnvironment(layout.Env, path+".env"); err != nil {
			return "", err
		}
		if layout.Children != nil || layout.Sizes != nil {
			return "", fmt.Errorf("%s: a pane cannot define children or sizes", path)
		}
		if err := ValidateProjectName(layout.Pane); err != nil {
			return "", fmt.Errorf("%s.pane: %w", path, err)
		}
		if all[layout.Pane] {
			return "", fmt.Errorf("%s.pane: duplicate pane identifier %q", path, layout.Pane)
		}
		all[layout.Pane], local[layout.Pane] = true, true
		if layout.Command != nil && layout.Commands != nil {
			return "", fmt.Errorf("%s: command and commands are mutually exclusive", path)
		}
		if len(layout.Command) > 0 {
			if err := workspace.ValidateCommand(layout.Command, path+".command"); err != nil {
				return "", err
			}
		}
		for i, command := range layout.Commands {
			if err := workspace.ValidateCommand(command, fmt.Sprintf("%s.commands[%d]", path, i)); err != nil {
				return "", err
			}
		}
		if strings.ContainsRune(layout.Cwd, 0) {
			return "", fmt.Errorf("%s.cwd must not contain NUL", path)
		}
		if layout.Cwd == "" {
			layout.Cwd = root
		} else if !filepath.IsAbs(layout.Cwd) {
			layout.Cwd = filepath.Join(root, layout.Cwd)
		} else {
			layout.Cwd = filepath.Clean(layout.Cwd)
		}
		return layout.Pane, nil
	}
	if layout.Split != "columns" && layout.Split != "rows" {
		return "", fmt.Errorf("%s.split must be columns or rows", path)
	}
	if layout.Command != nil || layout.Cwd != "" {
		return "", fmt.Errorf("%s: a split cannot define command or cwd", path)
	}
	if layout.Commands != nil {
		return "", fmt.Errorf("%s: a split cannot define commands", path)
	}
	if layout.Env != nil {
		return "", fmt.Errorf("%s: env belongs to a project, tab, or pane, not a split", path)
	}
	if len(layout.Children) < 2 {
		return "", fmt.Errorf("%s.children must contain at least two layouts", path)
	}
	if layout.Sizes != nil {
		if len(layout.Sizes) != len(layout.Children) {
			return "", fmt.Errorf("%s.sizes must have one percentage per child", path)
		}
		total := 0
		for i, size := range layout.Sizes {
			if size <= 0 || size >= 100 {
				return "", fmt.Errorf("%s.sizes[%d] must be an integer between 1 and 99", path, i)
			}
			total += size
		}
		if total != 100 {
			return "", fmt.Errorf("%s.sizes must sum to 100, got %d", path, total)
		}
	}
	first := ""
	for i := range layout.Children {
		pane, err := validateLayout(&layout.Children[i], fmt.Sprintf("%s.children[%d]", path, i), root, all, local, depth+1)
		if err != nil {
			return "", err
		}
		if i == 0 {
			first = pane
		}
	}
	return first, nil
}

// Reject YAML's implicit string/number coercions before decoding into structs.
func checkYAMLTypes(node *yaml.Node, path string, depth int) error {
	if depth > 128 {
		return fmt.Errorf("%s: YAML nesting is too deep", path)
	}
	if node.Kind == yaml.AliasNode || node.Tag == "!!null" {
		return fmt.Errorf("%s: YAML aliases and null values are not supported; omit optional fields instead", path)
	}
	if node.Kind == yaml.MappingNode {
		fields := map[string]bool{}
		for i := 0; i < len(node.Content); i += 2 {
			key, value := node.Content[i], node.Content[i+1]
			fields[key.Value] = true
			field := path + "." + key.Value
			if key.Tag != "!!str" {
				return fmt.Errorf("%s: field names must be strings", path)
			}
			// Environment keys are user-defined names, not workspace field
			// names (for example, env may legitimately contain a key 'command').
			if key.Value == "env" {
				if value.Kind != yaml.MappingNode {
					return fmt.Errorf("%s must be a mapping of string names to string values", field)
				}
				for j := 0; j < len(value.Content); j += 2 {
					name, val := value.Content[j], value.Content[j+1]
					if name.Kind != yaml.ScalarNode || name.Tag != "!!str" || val.Kind != yaml.ScalarNode || val.Tag != "!!str" {
						return fmt.Errorf("%s: environment names and values must be strings", field)
					}
				}
				continue
			}
			tag, kind := "", yaml.Kind(0)
			switch key.Value {
			case "selected":
				tag, kind = "!!bool", yaml.ScalarNode
			case "schema_version":
				tag, kind = "!!int", yaml.ScalarNode
			case "name", "root", "focus", "pane", "cwd", "split":
				tag, kind = "!!str", yaml.ScalarNode
			case "tabs", "children", "command", "commands", "sizes", "before_start", "after_start", "after_close", "before_close":
				kind = yaml.SequenceNode
			case "layout", "hooks":
				kind = yaml.MappingNode
			}
			if kind != 0 && (value.Kind != kind || (tag != "" && value.Tag != tag)) {
				return fmt.Errorf("%s: invalid value type", field)
			}
			if key.Value == "sizes" || key.Value == "command" {
				want := "!!str"
				if key.Value == "sizes" {
					want = "!!int"
				}
				for j, item := range value.Content {
					if item.Kind != yaml.ScalarNode || item.Tag != want {
						return fmt.Errorf("%s[%d]: invalid value type", field, j)
					}
				}
			}
			if key.Value == "commands" || key.Value == "before_start" || key.Value == "after_start" || key.Value == "after_close" || key.Value == "before_close" {
				for j, command := range value.Content {
					if command.Kind != yaml.SequenceNode {
						return fmt.Errorf("%s[%d] must be an array of strings", field, j)
					}
					for k, arg := range command.Content {
						if arg.Kind != yaml.ScalarNode || arg.Tag != "!!str" {
							return fmt.Errorf("%s[%d][%d]: invalid value type", field, j, k)
						}
					}
				}
			}
			if err := checkYAMLTypes(value, field, depth+1); err != nil {
				return err
			}
		}
		if fields["pane"] && fields["split"] {
			return fmt.Errorf("%s must define exactly one of pane or split", path)
		}
		if fields["command"] && fields["commands"] {
			return fmt.Errorf("%s: command and commands are mutually exclusive", path)
		}
		if fields["split"] && fields["commands"] {
			return fmt.Errorf("%s: a split cannot define commands", path)
		}
		if fields["split"] && fields["env"] {
			return fmt.Errorf("%s: env belongs to a project, tab, or pane, not a split", path)
		}
		if fields["split"] && (fields["cwd"] || fields["command"]) {
			return fmt.Errorf("%s: a split cannot define command or cwd", path)
		}
		return nil
	}
	for i, child := range node.Content {
		if err := checkYAMLTypes(child, fmt.Sprintf("%s[%d]", path, i), depth+1); err != nil {
			return err
		}
	}
	return nil
}
