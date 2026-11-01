//go:build !linux && !darwin && !dragonfly && !freebsd && !netbsd && !openbsd && !windows

package programstatus

import "os"

func isTerminal(file *os.File) bool { return false }
