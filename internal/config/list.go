package config

import (
	"fmt"
	"os"
	"slices"
	"strings"
)

// ListProjects returns saved project names in alphabetical order, without reading
// configuration contents. Nonregular files and invalid project names are ignored.
func ListProjects() ([]string, error) {
	projects, err := ProjectsDir()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(projects)
	if os.IsNotExist(err) {
		return []string{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("list project configurations in %q: %w", projects, err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.Type().IsRegular() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".yaml")
		if ValidateProjectName(name) == nil {
			names = append(names, name)
		}
	}
	slices.Sort(names)
	return names, nil
}
