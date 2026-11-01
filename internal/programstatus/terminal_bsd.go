//go:build darwin || dragonfly || freebsd || netbsd || openbsd

package programstatus

import (
	"golang.org/x/sys/unix"
	"os"
)

func isTerminal(file *os.File) bool {
	_, err := unix.IoctlGetTermios(int(file.Fd()), unix.TIOCGETA)
	return err == nil
}
