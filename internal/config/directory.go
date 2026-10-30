// Package config manages rexp's configuration directory.
package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// Initialize creates .rexp and its projects directory in the user's home.
// Existing directories and their contents are preserved.
func Initialize() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("find user home directory: %w", err)
	}
	projects := filepath.Join(home, ".rexp", "projects")
	if err := os.MkdirAll(projects, 0o700); err != nil {
		return fmt.Errorf("initialize rexp directory %q: %w", projects, err)
	}
	return nil
}
