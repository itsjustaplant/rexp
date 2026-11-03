package agentskill

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInstallPreservesNonRegularDestinations(t *testing.T) {
	for _, kind := range []string{"directory", "file-symlink", "directory-symlink"} {
		t.Run(kind, func(t *testing.T) {
			home := t.TempDir()
			dir := filepath.Join(home, ".agents", "skills", "rexp")
			if err := os.MkdirAll(dir, 0700); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(dir, "SKILL.md")
			switch kind {
			case "directory":
				if err := os.Mkdir(path, 0700); err != nil {
					t.Fatal(err)
				}
			case "file-symlink":
				target := filepath.Join(home, "keep.md")
				if err := os.WriteFile(target, []byte("keep"), 0600); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(target, path); err != nil {
					t.Skip(err)
				}
			case "directory-symlink":
				if err := os.Remove(dir); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(t.TempDir(), dir); err != nil {
					t.Skip(err)
				}
			}
			for _, force := range []bool{false, true} {
				if _, err := Install(home, "codex", force); err == nil {
					t.Fatal("accepted nonregular destination")
				}
			}
		})
	}
}
