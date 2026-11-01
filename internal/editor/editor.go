// Package editor opens project files in a user-selected external editor.
package editor

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"unicode"

	"rexp/internal/config"
)

// Resolve selects an editor without evaluating a shell command. An explicit
// override also allows fixing projects when the tool preferences are invalid.
func Resolve(override string, explicit bool) ([]string, error) {
	if explicit {
		return parse(override)
	}
	preference, err := config.EditorPreference()
	if err != nil {
		return nil, err
	}
	for _, command := range []string{preference, os.Getenv("VISUAL"), os.Getenv("EDITOR")} {
		if strings.TrimSpace(command) != "" {
			return parse(command)
		}
	}
	if runtime.GOOS == "windows" {
		return []string{"notepad.exe"}, nil
	}
	return []string{"vi"}, nil
}

// Open connects the editor to the caller's terminal and waits for it to exit.
// GUI editors must be configured with their own wait option, e.g. code --wait.
func Open(ctx context.Context, command []string, path string, stdin io.Reader, stdout, stderr io.Writer) error {
	if len(command) == 0 || command[0] == "" {
		return errors.New("editor command must name a program")
	}
	args := append(append([]string(nil), command[1:]...), path)
	process := exec.CommandContext(ctx, command[0], args...)
	process.Stdin, process.Stdout, process.Stderr = stdin, stdout, stderr
	if err := process.Run(); err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return fmt.Errorf("editor %q was not found on PATH; use --editor or configure VISUAL/EDITOR", command[0])
		}
		return fmt.Errorf("editor %q failed: %w", command[0], err)
	}
	return nil
}

// parse supports quoted arguments and escaped quotes/spaces. Backslashes before
// ordinary characters remain literal for Windows paths. No expansion, pipes,
// redirection, or command substitution is performed.
func parse(command string) ([]string, error) {
	if strings.ContainsRune(command, 0) {
		return nil, errors.New("editor command must not contain NUL")
	}
	var args []string
	var word strings.Builder
	var quote rune
	started := false
	runes := []rune(command)
	for i := 0; i < len(runes); i++ {
		c := runes[i]
		if quote == '\'' {
			if c == '\'' {
				quote = 0
			} else {
				word.WriteRune(c)
			}
			continue
		}
		if c == '\\' {
			if i+1 == len(runes) {
				return nil, errors.New("editor command ends with an incomplete escape")
			}
			next := runes[i+1]
			if next == '\\' || next == '"' || quote == 0 && (next == '\'' || unicode.IsSpace(next)) {
				word.WriteRune(next)
				i++
			} else {
				word.WriteRune(c)
			}
			started = true
			continue
		}
		if quote == '"' {
			if c == '"' {
				quote = 0
			} else {
				word.WriteRune(c)
			}
			continue
		}
		if c == '\'' || c == '"' {
			quote = c
			started = true
			continue
		}
		if unicode.IsSpace(c) {
			if started {
				args = append(args, word.String())
				word.Reset()
				started = false
			}
			continue
		}
		word.WriteRune(c)
		started = true
	}
	if quote != 0 {
		return nil, errors.New("editor command contains an unclosed quote")
	}
	if started {
		args = append(args, word.String())
	}
	if len(args) == 0 || args[0] == "" {
		return nil, errors.New("editor command must name a program")
	}
	return args, nil
}
