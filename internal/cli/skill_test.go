package cli

import (
	"os"
	"path/filepath"
	"strings"
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
		{"skill", "remove"}, {"skill", "remove", "other"},
		{"skill", "remove", "../escape"}, {"skill", "remove", "codex", "claude"},
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

func TestSkillRemove(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	t.Setenv("PATH", t.TempDir())
	for _, agent := range []struct{ name, directory string }{{"codex", ".agents"}, {"claude", ".claude"}} {
		t.Run(agent.name, func(t *testing.T) {
			if _, err := runDiscovery(t, "skill", "install", agent.name); err != nil {
				t.Fatal(err)
			}
			path := filepath.Join(home, agent.directory, "skills", "rexp", "SKILL.md")
			keep := filepath.Join(filepath.Dir(path), "keep.md")
			if err := os.WriteFile(keep, []byte("other resource"), 0600); err != nil {
				t.Fatal(err)
			}
			out, err := runDiscovery(t, "skill", "remove", agent.name)
			if err != nil || !strings.Contains(out, "Removed skill at "+path) {
				t.Fatalf("remove: %q %v", out, err)
			}
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Fatalf("skill was not removed: %v", err)
			}
			out, err = runDiscovery(t, "skill", "remove", agent.name)
			if err != nil || !strings.Contains(out, "No skill installed") {
				t.Fatalf("repeat removal: %q %v", out, err)
			}
			if err := os.WriteFile(path, []byte("older or customized skill"), 0600); err != nil {
				t.Fatal(err)
			}
			if _, err := runDiscovery(t, "skill", "remove", agent.name); err == nil {
				t.Fatal("removed differing content without force")
			}
			content, err := os.ReadFile(path)
			if err != nil || string(content) != "older or customized skill" {
				t.Fatalf("differing content was changed: %q %v", content, err)
			}
			if _, err := runDiscovery(t, "skill", "remove", agent.name, "--force"); err != nil {
				t.Fatal(err)
			}
			if _, err := os.Stat(path); !os.IsNotExist(err) {
				t.Fatalf("force did not remove skill: %v", err)
			}
			content, err = os.ReadFile(keep)
			if err != nil || string(content) != "other resource" {
				t.Fatalf("removal changed other files: %q %v", content, err)
			}
		})
	}
}
