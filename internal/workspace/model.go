// Package workspace defines the configuration model shared by loaders and launchers.
package workspace

// Project is version 1 of the saved workspace configuration.
type Project struct {
	SchemaVersion int               `yaml:"schema_version"`
	Name          string            `yaml:"name"`
	Root          string            `yaml:"root"`
	Env           map[string]string `yaml:"env,omitempty"`
	Tabs          []Tab             `yaml:"tabs"`
}

type Tab struct {
	Name     string            `yaml:"name"`
	Selected bool              `yaml:"selected,omitempty"`
	Env      map[string]string `yaml:"env,omitempty"`
	Focus    string            `yaml:"focus,omitempty"`
	Layout   Layout            `yaml:"layout"`
}

// InitialTab returns the selected tab, falling back to the first. Call only on
// a validated project containing at least one tab.
func (project *Project) InitialTab() Tab {
	for _, tab := range project.Tabs {
		if tab.Selected {
			return tab
		}
	}
	return project.Tabs[0]
}

// Layout is either a terminal pane or a group of rows/columns.
type Layout struct {
	Pane     string            `yaml:"pane,omitempty"`
	Command  []string          `yaml:"command,omitempty"`
	Commands [][]string        `yaml:"commands,omitempty"`
	Env      map[string]string `yaml:"env,omitempty"`
	Cwd      string            `yaml:"cwd,omitempty"`
	Split    string            `yaml:"split,omitempty"`
	Sizes    []int             `yaml:"sizes,omitempty"`
	Children []Layout          `yaml:"children,omitempty"`
}

// Shares returns each split child's fraction of its parent's width or height.
// Call only on validated layouts. Omitted sizes divide the space equally.
func (layout Layout) Shares() []float64 {
	shares := make([]float64, len(layout.Children))
	for i := range shares {
		if layout.Sizes == nil {
			shares[i] = 1 / float64(len(shares))
		} else {
			shares[i] = float64(layout.Sizes[i]) / 100
		}
	}
	return shares
}
