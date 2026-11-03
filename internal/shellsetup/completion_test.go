package shellsetup

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestCompletionPreservesStartupFilesAndIsIdempotent(t *testing.T) {
	for _, shell := range []string{"zsh", "bash", "fish"} {
		t.Run(shell, func(t *testing.T) {
			home := t.TempDir()
			t.Setenv("ZDOTDIR", "")
			t.Setenv("XDG_CONFIG_HOME", "")
			files, err := Plan(home, shell)
			if err != nil {
				t.Fatal(err)
			}
			if shell != "fish" {
				for _, file := range files {
					if err := os.WriteFile(file.Path, []byte("# keep my settings\nexport EDITOR=vi"), 0640); err != nil {
						t.Fatal(err)
					}
				}
			}
			if err := Install(files); err != nil {
				t.Fatal(err)
			}
			for _, file := range files {
				before, err := os.ReadFile(file.Path)
				if err != nil {
					t.Fatal(err)
				}
				if shell != "fish" && !strings.HasPrefix(string(before), "# keep my settings\nexport EDITOR=vi\n") {
					t.Fatalf("lost original contents: %q", before)
				}
				if runtime.GOOS != "windows" && shell != "fish" {
					info, _ := os.Stat(file.Path)
					if info.Mode().Perm() != 0640 {
						t.Fatal("changed startup file permissions")
					}
				}
				if err := Install(files); err != nil {
					t.Fatal(err)
				}
				after, _ := os.ReadFile(file.Path)
				if string(after) != string(before) {
					t.Fatal("repeat install duplicated completion")
				}
				if executable, err := exec.LookPath(shell); err == nil {
					if output, err := exec.Command(executable, "-n", file.Path).CombinedOutput(); err != nil {
						t.Fatalf("shell syntax: %v %s", err, output)
					}
				}
			}
		})
	}
}

func TestCompletionHonorsShellDirectoriesAndBashLoginPrecedence(t *testing.T) {
	home := t.TempDir()
	t.Setenv("ZDOTDIR", filepath.Join(home, "zsh"))
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(home, "xdg"))
	files, _ := Plan(home, "zsh")
	if files[0].Path != filepath.Join(home, "zsh", ".zshrc") {
		t.Fatalf("ZDOTDIR ignored: %v", files)
	}
	files, _ = Plan(home, "fish")
	if files[0].Path != filepath.Join(home, "xdg", "fish", "completions", "rexp.fish") {
		t.Fatalf("XDG_CONFIG_HOME ignored: %v", files)
	}
	for _, name := range []string{".profile", ".bash_login", ".bash_profile"} {
		if err := os.WriteFile(filepath.Join(home, name), nil, 0600); err != nil {
			t.Fatal(err)
		}
		files, _ = Plan(home, "bash")
		if files[1].Path != filepath.Join(home, name) {
			t.Fatalf("wrong login startup file: %v", files)
		}
		if name == ".profile" {
			if err := Install(files); err != nil {
				t.Fatal(err)
			}
			if sh, err := exec.LookPath("sh"); err == nil {
				if out, err := exec.Command(sh, files[1].Path).CombinedOutput(); err != nil {
					t.Fatalf("Bash setup broke POSIX profile: %s %v", out, err)
				}
			}
		}
	}
}

func TestCompletionRejectsUnsafeOrConflictingFilesBeforeWriting(t *testing.T) {
	for _, conflict := range []string{"directory", "symlink", "broken-block", "custom-fish"} {
		t.Run(conflict, func(t *testing.T) {
			home := t.TempDir()
			first := filepath.Join(home, "untouched")
			path := filepath.Join(home, "bad")
			switch conflict {
			case "directory":
				if err := os.Mkdir(path, 0700); err != nil {
					t.Fatal(err)
				}
			case "symlink":
				target := filepath.Join(home, "target")
				if err := os.WriteFile(target, []byte("keep"), 0600); err != nil {
					t.Fatal(err)
				}
				if err := os.Symlink(target, path); err != nil {
					t.Skip(err)
				}
			case "broken-block":
				if err := os.WriteFile(path, []byte(begin), 0600); err != nil {
					t.Fatal(err)
				}
			case "custom-fish":
				path += ".fish"
				if err := os.WriteFile(path, []byte("# custom completion"), 0600); err != nil {
					t.Fatal(err)
				}
			}
			if err := Install([]File{{first, "test"}, {path, "test"}}); err == nil {
				t.Fatal("modified conflicting shell file")
			}
			if _, err := os.Stat(first); !os.IsNotExist(err) {
				t.Fatal("partially applied invalid completion plan")
			}
		})
	}
}
