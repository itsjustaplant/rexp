package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// ProjectFile returns an existing regular project configuration file.
// Directories and symbolic links are rejected without following them.
func ProjectFile(name string) (string, error) {
	if err := ValidateProjectName(name); err != nil {
		return "", err
	}
	projects, err := ProjectsDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(projects, name+".yaml")
	info, err := os.Lstat(path)
	if errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("project %q does not exist at %s", name, path)
	}
	if err != nil {
		return "", fmt.Errorf("inspect project configuration: %w", err)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("project configuration %q must be a regular file; directories and symbolic links are not supported", path)
	}
	return path, nil
}

// DeleteProject removes only the saved configuration, not the project directory
// or its running Rex session. The caller handles user confirmation.
func DeleteProject(name string) (string, error) {
	path, err := ProjectFile(name)
	if err != nil {
		return "", err
	}
	if err := os.Remove(path); err != nil {
		return "", fmt.Errorf("delete project configuration %q: %w", path, err)
	}
	return path, nil
}
