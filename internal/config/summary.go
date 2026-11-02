package config

import "path/filepath"

// ProjectSummary is configuration metadata, independent of live Rex state.
// Invalid entries retain their name/path and error, without misleading counts.
type ProjectSummary struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	Valid bool   `json:"valid"`
	Root  string `json:"root,omitempty"`
	Tabs  int    `json:"tabs,omitempty"`
	Panes int    `json:"panes,omitempty"`
	Error string `json:"error,omitempty"`
}

func SummarizeProjects(names []string) []ProjectSummary {
	summaries := make([]ProjectSummary, 0, len(names))
	for _, name := range names {
		summary := ProjectSummary{Name: name}
		dir, err := ProjectsDir()
		if err == nil {
			err = ValidateProjectName(name)
		}
		if err == nil {
			summary.Path = filepath.Join(dir, name+".yaml")
		}
		if err == nil {
			var project *Project
			project, err = LoadProject(name)
			if err == nil {
				summary.Valid = true
				summary.Root = project.Root
				summary.Tabs = len(project.Tabs)
				for _, tab := range project.Tabs {
					summary.Panes += paneCount(tab.Layout)
				}
			}
		}
		if err != nil {
			summary.Error = err.Error()
		}
		summaries = append(summaries, summary)
	}
	return summaries
}

func paneCount(layout Layout) int {
	if layout.Pane != "" {
		return 1
	}
	count := 0
	for _, child := range layout.Children {
		count += paneCount(child)
	}
	return count
}
