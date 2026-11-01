package workspace

import (
	"fmt"
	"regexp"
	"runtime"
	"sort"
	"strings"
)

var environmentName = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func environmentKey(key string) string {
	if runtime.GOOS == "windows" {
		return strings.ToUpper(key)
	}
	return key
}

func ValidateEnvironment(env map[string]string, path string) error {
	seen := map[string]bool{}
	for _, key := range sortedKeys(env) {
		if !environmentName.MatchString(key) {
			return fmt.Errorf("%s: invalid environment variable name %q", path, key)
		}
		if strings.ContainsRune(env[key], 0) {
			return fmt.Errorf("%s.%s must not contain NUL", path, key)
		}
		canonical := environmentKey(key)
		if seen[canonical] {
			return fmt.Errorf("%s: duplicate environment variable %q", path, key)
		}
		seen[canonical] = true
	}
	return nil
}

// MergeEnvironment copies scopes from broadest to most specific. Windows
// variable names are case-insensitive; values and input maps remain unchanged.
func MergeEnvironment(scopes ...map[string]string) map[string]string {
	result := map[string]string{}
	names := map[string]string{}
	for _, scope := range scopes {
		for _, key := range sortedKeys(scope) {
			canonical := environmentKey(key)
			if previous, ok := names[canonical]; ok {
				delete(result, previous)
			}
			names[canonical] = key
			result[key] = scope[key]
		}
	}
	return result
}

// OverlayEnvironment retains the inherited launch environment and replaces
// only configured variables. It does not modify the caller's process state.
func OverlayEnvironment(base []string, overrides map[string]string) []string {
	keys := map[string]bool{}
	for key := range overrides {
		keys[environmentKey(key)] = true
	}
	result := make([]string, 0, len(base)+len(overrides))
	for _, entry := range base {
		start := 0
		if strings.HasPrefix(entry, "=") {
			start = 1
		} // Windows hidden drive variables.
		separator := strings.IndexByte(entry[start:], '=')
		if separator < 0 {
			continue
		}
		if !keys[environmentKey(entry[:start+separator])] {
			result = append(result, entry)
		}
	}
	for _, key := range sortedKeys(overrides) {
		result = append(result, key+"="+overrides[key])
	}
	return result
}

func sortedKeys(env map[string]string) []string {
	keys := make([]string, 0, len(env))
	for key := range env {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
