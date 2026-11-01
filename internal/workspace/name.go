package workspace

import (
	"fmt"
	"regexp"
	"strings"
)

var projectNamePattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_-]{0,63}$`)

// ValidateName checks that a name is portable and safe as a filename.
func ValidateName(name string) error {
	if !projectNamePattern.MatchString(name) {
		return fmt.Errorf("project name must be 1–64 ASCII letters, digits, hyphens, or underscores, starting with a letter or digit")
	}
	upper := strings.ToUpper(name)
	reserved := upper == "CON" || upper == "PRN" || upper == "AUX" || upper == "NUL"
	if len(upper) == 4 && (strings.HasPrefix(upper, "COM") || strings.HasPrefix(upper, "LPT")) && upper[3] >= '1' && upper[3] <= '9' {
		reserved = true
	}
	if reserved {
		return fmt.Errorf("project name %q is reserved on Windows; choose another name", name)
	}
	return nil
}
