// Package agentskill distributes the shared rexp skill with the executable.
package agentskill

import (
	_ "embed"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed rexp/SKILL.md
var Content string

// Install writes the bundled skill, preserving existing content unless forced.
// An identical installation is a successful no-op.
func Install(home, agent string, force bool) (string, error) {
	var directory string
	switch agent {
	case "codex":
		directory = ".agents"
	case "claude":
		directory = ".claude"
	default:
		return "", fmt.Errorf("unknown agent %q: choose codex or claude", agent)
	}
	dir := filepath.Join(home, directory, "skills", "rexp")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", fmt.Errorf("create skill directory: %w", err)
	}
	info, err := os.Lstat(dir)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("skill directory must be a regular directory: %s", dir)
	}
	path := filepath.Join(dir, "SKILL.md")
	info, err = os.Lstat(path)
	if err == nil {
		if !info.Mode().IsRegular() {
			return "", fmt.Errorf("skill destination must be a regular file: %s", path)
		}
		content, readErr := os.ReadFile(path)
		if readErr != nil {
			return "", fmt.Errorf("read existing skill: %w", readErr)
		}
		if string(content) == Content {
			return path, nil
		}
		if !force {
			return "", fmt.Errorf("skill already exists at %s; use --force to replace it", path)
		}
	} else if !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("inspect skill destination: %w", err)
	}
	if !force {
		file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
		if err != nil {
			return "", fmt.Errorf("create skill: %w", err)
		}
		_, writeErr := file.WriteString(Content)
		closeErr := file.Close()
		if err := errors.Join(writeErr, closeErr); err != nil {
			return "", fmt.Errorf("write skill: %w", err)
		}
		return path, nil
	}
	// Replace via a sibling file so a failed write preserves the installed skill.
	file, err := os.CreateTemp(dir, ".SKILL-*")
	if err != nil {
		return "", fmt.Errorf("prepare skill replacement: %w", err)
	}
	defer os.Remove(file.Name())
	_, writeErr := file.WriteString(Content)
	closeErr := file.Close()
	if err := errors.Join(writeErr, closeErr); err != nil {
		return "", fmt.Errorf("write skill: %w", err)
	}
	if err := os.Rename(file.Name(), path); err != nil {
		return "", fmt.Errorf("replace skill: %w", err)
	}
	return path, nil
}
