package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"go.yaml.in/yaml/v3"

	"rexp/internal/workspace"
)

// ValidateProjectName checks that a name is portable and safe as a filename.
func ValidateProjectName(name string) error { return workspace.ValidateName(name) }

// CreateProject creates a starter YAML configuration and never overwrites a file.
func CreateProject(name, root string) (string, error) {
	if err := ValidateProjectName(name); err != nil {
		return "", err
	}
	if !filepath.IsAbs(root) {
		return "", fmt.Errorf("project root must be an absolute directory")
	}
	info, err := os.Stat(root)
	if err != nil {
		return "", fmt.Errorf("inspect project root: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("project root %q is not a directory", root)
	}
	return createProjectFile(name, starterProject(name, root))
}

func starterProject(name, root string) []byte {
	// JSON strings are valid YAML quoted scalars, including Windows paths.
	quotedName, _ := json.Marshal(name)
	quotedRoot, _ := json.Marshal(root)
	content := fmt.Sprintf(`# rexp project configuration
# Pane directories will be relative to root. An empty command opens a shell.
schema_version: 1
name: %s
root: %s
tabs:
  - name: Terminal
    layout:
      pane: shell
      command: []
`, quotedName, quotedRoot)
	return []byte(content)
}

// SaveProject validates and serializes an exported workspace. Existing files,
// including symbolic links, are never overwritten.
func SaveProject(project *Project) (string, error) {
	if project == nil {
		return "", errors.New("cannot save an empty project")
	}
	content, err := yaml.Marshal(project)
	if err != nil {
		return "", fmt.Errorf("encode project configuration: %w", err)
	}
	if _, err := DecodeProject(content); err != nil {
		return "", fmt.Errorf("validate saved configuration: %w", err)
	}
	header := "# Saved Rex workspace.\n# Empty or omitted commands open shells. rexp save cannot reconstruct full command arguments.\n# Add pane commands manually to recreate your programs.\n"
	return createProjectFile(project.Name, append([]byte(header), content...))
}

func createProjectFile(name string, content []byte) (string, error) {
	if err := ValidateProjectName(name); err != nil {
		return "", err
	}
	if err := Initialize(); err != nil {
		return "", err
	}
	projects, err := ProjectsDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(projects, name+".yaml")
	return writeProjectFile(name, path, content)
}

func writeProjectFile(name, path string, content []byte) (string, error) {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if errors.Is(err, os.ErrExist) {
		return "", fmt.Errorf("project %q already exists at %s; existing configuration was preserved", name, path)
	}
	if err != nil {
		return "", fmt.Errorf("create project configuration: %w", err)
	}
	_, writeErr := file.Write(content)
	closeErr := file.Close()
	if err := errors.Join(writeErr, closeErr); err != nil {
		removeErr := os.Remove(path)
		return "", fmt.Errorf("write project configuration: %w", errors.Join(err, removeErr))
	}
	return path, nil
}
