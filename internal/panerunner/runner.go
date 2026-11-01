// Package panerunner runs ordered argv commands within one Rex terminal. It
// inherits the pane's terminal and directory, without shell interpolation.
package panerunner

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"runtime"

	"rexp/internal/workspace"
)

const Mode = "__run-pane"

type Payload struct {
	Version  int               `json:"version"`
	Commands [][]string        `json:"commands"`
	Env      map[string]string `json:"env,omitempty"`
	Shell    bool              `json:"shell,omitempty"`
}

func (payload Payload) validate() error {
	if payload.Version != 1 {
		return errors.New("unsupported pane runner payload version")
	}
	if payload.Shell && len(payload.Commands) > 0 {
		return errors.New("pane runner shell and commands are mutually exclusive")
	}
	if !payload.Shell && len(payload.Commands) == 0 {
		return errors.New("pane runner requires at least one command")
	}
	if err := workspace.ValidateEnvironment(payload.Env, "env"); err != nil {
		return err
	}
	for i, command := range payload.Commands {
		if err := workspace.ValidateCommand(command, fmt.Sprintf("commands[%d]", i)); err != nil {
			return err
		}
	}
	return nil
}

// Invocation uses the same built binary, available at this absolute path on
// the local Rex host. No dependence on the user's PATH for rexp itself.
func Invocation(commands [][]string) ([]string, error) {
	return InvocationWithEnvironment(commands, nil, false)
}

func InvocationWithEnvironment(commands [][]string, env map[string]string, shell bool) ([]string, error) {
	payload := Payload{Version: 1, Commands: commands, Env: env, Shell: shell}
	if err := payload.validate(); err != nil {
		return nil, err
	}
	executable, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("locate pane runner: %w", err)
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	return []string{executable, Mode, base64.RawURLEncoding.EncodeToString(data)}, nil
}

func Decode(encoded string) (Payload, error) {
	var payload Payload
	data, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return payload, errors.New("invalid pane runner payload encoding")
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&payload); err != nil {
		return payload, errors.New("invalid pane runner payload JSON")
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return payload, errors.New("pane runner payload must contain one JSON document")
	}
	return payload, payload.validate()
}

// Run never initializes rexp's configuration or sends its lifecycle reports.
// A finite sequence exits after success; --keep-open preserves its output.
func Run(ctx context.Context, args []string, stdin io.Reader, stdout, stderr io.Writer) (int, error) {
	if len(args) != 1 {
		return 2, errors.New("invalid pane runner invocation")
	}
	payload, err := Decode(args[0])
	if err != nil {
		return 2, err
	}
	environment := workspace.OverlayEnvironment(os.Environ(), payload.Env)
	commands := payload.Commands
	if payload.Shell {
		shell := environmentValue(environment, "SHELL")
		if runtime.GOOS == "windows" {
			shell = environmentValue(environment, "COMSPEC")
			if shell == "" {
				shell = "cmd.exe"
			}
		} else if shell == "" {
			shell = "/bin/sh"
		}
		commands = [][]string{{shell}}
	}
	for i, command := range commands {
		if err := ctx.Err(); err != nil {
			return 130, err
		}
		program, lookupErr := findExecutable(command[0], environment)
		process := exec.CommandContext(ctx, program, command[1:]...)
		process.Args[0] = command[0]
		process.Env = environment
		process.Stdin, process.Stdout, process.Stderr = stdin, stdout, stderr
		runErr := lookupErr
		if runErr == nil {
			runErr = process.Run()
		}
		if err := runErr; err != nil {
			if ctx.Err() != nil {
				return 130, ctx.Err()
			}
			code := 126
			var exited *exec.ExitError
			if errors.As(err, &exited) {
				code = exited.ExitCode()
				if code < 1 {
					code = 1
				}
			} else if errors.Is(err, exec.ErrNotFound) || errors.Is(err, os.ErrNotExist) {
				code = 127
			}
			// Avoid echoing argv: arguments may contain credentials or payloads.
			return code, fmt.Errorf("pane command %d failed (exit %d)", i+1, code)
		}
	}
	return 0, nil
}
