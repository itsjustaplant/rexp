package programstatus

import (
	"golang.org/x/sys/windows"
	"os"
)

func isTerminal(file *os.File) bool {
	var mode uint32
	return windows.GetConsoleMode(windows.Handle(file.Fd()), &mode) == nil
}
