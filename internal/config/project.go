package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

var projectNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$`)

// ValidateProjectName checks that a name is portable and safe as a filename.
func ValidateProjectName(name string) error {
	if !projectNamePattern.MatchString(name) {
		return fmt.Errorf("project name must be 1–64 ASCII letters, digits, hyphens, or underscores, starting with a letter or digit")
	}
	upper := strings.ToUpper(name)
	reserved := upper == "CON" || upper == "PRN" || upper == "AUX" || upper == "NUL"
	if len(upper) == 4 && (strings.HasPrefix(upper, "COM") || strings.HasPrefix(upper, "LPT")) && upper[3] >= '1' && upper[3] <= '9' {
		reserved = true
	}
	if reserved {
		return fmt.Errorf("project name %q is reserved on Windows; choose another name", name)
	}
	return nil
}

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
	if err := Initialize(); err != nil {
		return "", err
	}
	projects, err := ProjectsDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(projects, name+".yaml")
	// JSON strings are valid YAML quoted scalars, including Windows paths.
	quotedName, _ := json.Marshal(name)
	quotedRoot, err := json.Marshal(root)
	if err != nil {
		return "", fmt.Errorf("encode project root: %w", err)
	}
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
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if errors.Is(err, os.ErrExist) {
		return "", fmt.Errorf("project %q already exists at %s; existing configuration was preserved", name, path)
	}
	if err != nil {
		return "", fmt.Errorf("create project configuration: %w", err)
	}
	_, writeErr := file.WriteString(content)
	closeErr := file.Close()
	if err := errors.Join(writeErr, closeErr); err != nil {
		removeErr := os.Remove(path)
		return "", fmt.Errorf("write project configuration: %w", errors.Join(err, removeErr))
	}
	return path, nil
}
