package panerunner

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func environmentValue(environment []string, name string) string {
	for i := len(environment) - 1; i >= 0; i-- {
		key, value, ok := strings.Cut(environment[i], "=")
		if ok && (key == name || runtime.GOOS == "windows" && strings.EqualFold(key, name)) {
			return value
		}
	}
	return ""
}

// exec.Command's usual lookup uses the parent's PATH before Cmd.Env applies.
// Resolve against the effective pane PATH instead, retaining Go's protection
// against implicit execution from a relative PATH entry (ErrDot).
func findExecutable(name string, environment []string) (string, error) {
	if strings.ContainsRune(name, '/') || runtime.GOOS == "windows" && strings.ContainsAny(name, `\:`) {
		return name, nil
	}
	extensions := []string{""}
	if runtime.GOOS == "windows" && filepath.Ext(name) == "" {
		ext := environmentValue(environment, "PATHEXT")
		if ext == "" {
			ext = ".COM;.EXE;.BAT;.CMD"
		}
		extensions = strings.Split(ext, ";")
	}
	var permission error
	for _, directory := range filepath.SplitList(environmentValue(environment, "PATH")) {
		for _, extension := range extensions {
			candidate := filepath.Join(directory, name+extension)
			info, err := os.Stat(candidate)
			if err != nil {
				if !os.IsNotExist(err) {
					permission = err
				}
				continue
			}
			if info.IsDir() || runtime.GOOS != "windows" && info.Mode().Perm()&0o111 == 0 {
				permission = os.ErrPermission
				continue
			}
			if !filepath.IsAbs(candidate) {
				return "", fmt.Errorf("%s: %w", name, exec.ErrDot)
			}
			return candidate, nil
		}
	}
	if permission != nil {
		return "", permission
	}
	return "", exec.ErrNotFound
}
