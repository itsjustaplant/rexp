package programstatus

import (
	"golang.org/x/sys/unix"
	"os"
)

func isTerminal(file *os.File) bool {
	_, err := unix.IoctlGetTermios(int(file.Fd()), unix.TCGETS)
	return err == nil
}
