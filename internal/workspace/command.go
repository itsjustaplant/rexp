package workspace

import (
	"fmt"
	"strings"
)

func ValidateCommand(command []string, path string) error {
	if len(command) == 0 {
		return fmt.Errorf("%s must contain a program", path)
	}
	if strings.TrimSpace(command[0]) == "" {
		return fmt.Errorf("%s[0] must name a program", path)
	}
	for i, arg := range command {
		if strings.ContainsRune(arg, 0) {
			return fmt.Errorf("%s[%d] must not contain NUL", path, i)
		}
	}
	return nil
}
