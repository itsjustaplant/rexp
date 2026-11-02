//go:build !unix && !windows

package hooks

import "os/exec"

func detach(cmd *exec.Cmd) {}

func killDetached(cmd *exec.Cmd) { _ = cmd.Process.Kill() }
