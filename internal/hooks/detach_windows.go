package hooks

import (
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"syscall"
)

func detach(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: 0x00000200 | 0x00000008}
}

func killDetached(cmd *exec.Cmd) {
	// Terminate the observer's Rex child too when subscription fails.
	root := os.Getenv("SystemRoot")
	if filepath.IsAbs(root) {
		_ = exec.Command(filepath.Join(root, "System32", "taskkill.exe"), "/PID", strconv.Itoa(cmd.Process.Pid), "/T", "/F").Run()
	}
	_ = cmd.Process.Kill()
}
