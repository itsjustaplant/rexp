//go:build !darwin && !dragonfly && !freebsd && !linux && !netbsd && !openbsd && !windows

package sessionstate

import (
	"errors"
	"os"
)

func tryLock(*os.File) (bool, error) {
	return false, errors.New("session locking is unsupported on this platform")
}
func unlock(*os.File) {}
