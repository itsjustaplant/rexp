package config

import (
	"fmt"
	"os"
	"path/filepath"
)

const LocalFilename = ".rexp.yaml"

// LocalFile resolves an explicit config file and rejects nonregular files,
// matching the named-project file policy. No config discovery or code runs.
func LocalFile(path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("configuration file path must not be empty")
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("resolve configuration path: %w", err)
	}
	info, err := os.Lstat(absolute)
	if err != nil {
		return "", fmt.Errorf("inspect configuration file %q: %w", absolute, err)
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("configuration file %q must be a regular file; directories and symbolic links are not supported", absolute)
	}
	canonical, err := filepath.EvalSymlinks(absolute)
	if err != nil {
		return "", fmt.Errorf("resolve configuration file: %w", err)
	}
	return canonical, nil
}

func LoadFile(path string) (*Project, error) {
	path, err := LocalFile(path)
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read configuration file %q: %w", path, err)
	}
	project, err := decodeProject(data, filepath.Dir(path))
	if err != nil {
		return nil, fmt.Errorf("configuration file %q: %w", path, err)
	}
	project.SourcePath = path
	return project, nil
}

// CreateLocalProject uses a relative root so the file can be shared in Git.
func CreateLocalProject(name, directory string) (string, error) {
	if err := ValidateProjectName(name); err != nil {
		return "", err
	}
	if !filepath.IsAbs(directory) {
		return "", fmt.Errorf("project directory must be absolute")
	}
	info, err := os.Stat(directory)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", fmt.Errorf("project directory is not a directory")
	}
	return writeProjectFile(name, filepath.Join(directory, LocalFilename), starterProject(name, "."))
}
