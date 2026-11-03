// Package shellsetup installs opt-in user shell completion without replacing
// existing startup files or requiring extra dependencies.
package shellsetup

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const begin = "# >>> rexp completion >>>"
const end = "# <<< rexp completion <<<"

type File struct {
	Path string
	Body string
}

// Plan describes the files shown to the user before requesting permission.
func Plan(home, shell string) ([]File, error) {
	switch shell {
	case "zsh":
		dir := os.Getenv("ZDOTDIR")
		if dir == "" {
			dir = home
		}
		path, err := filepath.Abs(filepath.Join(dir, ".zshrc"))
		if err != nil {
			return nil, err
		}
		return []File{{path, `if command -v rexp >/dev/null 2>&1; then
  autoload -Uz compinit
  if ! (( $+functions[compdef] )); then
    compinit
  fi
  source <(rexp completion zsh)
fi`}}, nil
	case "bash":
		// .profile may also be read by a POSIX shell. Keep its syntax portable
		// and evaluate the Bash-specific script only when running under Bash.
		body := `if [ -n "${BASH_VERSION:-}" ] && command -v rexp >/dev/null 2>&1; then
  case ":${SHELLOPTS:-}:" in
    *:posix:*) ;;
    *) eval "$(rexp completion bash)" ;;
  esac
fi`
		login := filepath.Join(home, ".bash_profile")
		for _, name := range []string{".bash_profile", ".bash_login", ".profile"} {
			path := filepath.Join(home, name)
			if _, err := os.Lstat(path); err == nil {
				login = path
				break
			} else if !errors.Is(err, os.ErrNotExist) {
				return nil, fmt.Errorf("inspect Bash startup file: %w", err)
			}
		}
		return []File{{filepath.Join(home, ".bashrc"), body}, {login, body}}, nil
	case "fish":
		dir := os.Getenv("XDG_CONFIG_HOME")
		if !filepath.IsAbs(dir) {
			dir = filepath.Join(home, ".config")
		}
		return []File{{filepath.Join(dir, "fish", "completions", "rexp.fish"), "if type -q rexp\n  rexp completion fish | source\nend"}}, nil
	default:
		return nil, fmt.Errorf("automatic completion setup supports zsh, bash, and fish; use rexp completion <shell> for manual setup")
	}
}

// pending preserves existing contents and treats an existing complete managed
// block as installed, including user edits within it.
func pending(file File) ([]byte, bool, error) {
	info, err := os.Lstat(file.Path)
	if errors.Is(err, os.ErrNotExist) {
		return []byte(begin + "\n" + file.Body + "\n" + end + "\n"), false, nil
	}
	if err != nil {
		return nil, false, err
	}
	if !info.Mode().IsRegular() {
		return nil, false, fmt.Errorf("shell file must be a regular file (symlinks are not modified): %s", file.Path)
	}
	data, err := os.ReadFile(file.Path)
	if err != nil {
		return nil, false, err
	}
	text := string(data)
	starts, ends := strings.Count(text, begin), strings.Count(text, end)
	if starts != 0 || ends != 0 {
		if starts != 1 || ends != 1 || strings.Index(text, begin) > strings.Index(text, end) {
			return nil, false, fmt.Errorf("incomplete rexp completion block in %s; repair it before rerunning init", file.Path)
		}
		return nil, true, nil
	}
	// Fish completion files belong to this command name; preserve unfamiliar ones.
	if filepath.Ext(file.Path) == ".fish" && len(data) > 0 {
		return nil, false, fmt.Errorf("existing Fish completion file was preserved: %s; review it before rerunning init", file.Path)
	}
	return []byte("\n" + begin + "\n" + file.Body + "\n" + end + "\n"), true, nil
}

func Install(files []File) error {
	// Validate every destination before modifying any of them.
	data := make([][]byte, len(files))
	exists := make([]bool, len(files))
	for i, file := range files {
		var err error
		data[i], exists[i], err = pending(file)
		if err != nil {
			return err
		}
	}
	for i, file := range files {
		if data[i] == nil {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(file.Path), 0700); err != nil {
			return err
		}
		flags := os.O_WRONLY | os.O_APPEND
		if !exists[i] {
			flags |= os.O_CREATE | os.O_EXCL
		}
		out, err := os.OpenFile(file.Path, flags, 0600)
		if err != nil {
			return fmt.Errorf("open shell file %s: %w", file.Path, err)
		}
		_, writeErr := out.Write(data[i])
		if err := errors.Join(writeErr, out.Close()); err != nil {
			return fmt.Errorf("write shell file %s: %w", file.Path, err)
		}
	}
	return nil
}
