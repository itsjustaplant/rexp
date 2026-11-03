package cli

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/itsjustaplant/rexp/internal/agentskill"
)

func TestSkillShowAndInstall(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("PATH", t.TempDir()) // Neither operation needs Rex or an agent executable.
	out, err := runDiscovery(t, "skill", "show")
	if err != nil || out != agentskill.Content {
		t.Fatalf("show: %q %v", out, err)
	}
	for _, agent := range []struct{ name, directory string }{{"codex", ".agents"}, {"claude", ".claude"}} {
		t.Run(agent.name, func(t *testing.T) {
			path := filepath.Join(home, agent.directory, "skills", "rexp", "SKILL.md")
			for range 2 {
				if _, err := runDiscovery(t, "skill", "install", agent.name); err != nil {
					t.Fatal(err)
				}
			}
			content, err := os.ReadFile(path)
			if err != nil || string(content) != out {
				t.Fatalf("installed skill differs from show: %v", err)
			}
			if err := os.WriteFile(path, []byte("user customization"), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := runDiscovery(t, "skill", "install", agent.name); err == nil {
				t.Fatal("replaced a customized skill without force")
			}
			content, _ = os.ReadFile(path)
			if string(content) != "user customization" {
				t.Fatal("changed customized skill")
			}
			if _, err := runDiscovery(t, "skill", "install", agent.name, "--force"); err != nil {
				t.Fatal(err)
			}
			content, _ = os.ReadFile(path)
			if string(content) != out {
				t.Fatal("force did not update skill")
			}
		})
	}
}

func TestSkillInvalidArguments(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	for _, args := range [][]string{
		{"skill", "show", "extra"}, {"skill", "install"},
		{"skill", "install", "other"}, {"skill", "install", "../escape"},
		{"skill", "install", "codex", "claude"},
	} {
		if _, err := runDiscovery(t, args...); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
	entries, err := os.ReadDir(home)
	if err != nil || len(entries) != 0 {
		t.Fatalf("invalid invocation wrote files: %v %v", entries, err)
	}
}
