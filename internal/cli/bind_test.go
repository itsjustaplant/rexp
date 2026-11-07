package cli

import (
	"os"
	"strings"
	"testing"

	"github.com/itsjustaplant/rexp/internal/config"
)

func TestBindCLIAdoptsRenamedSessionAndForgetsWithoutClosing(t *testing.T) {
	calls := switchFixture(t)
	if _, err := config.CreateProject("demo", t.TempDir()); err != nil {
		t.Fatal(err)
	}
	t.Setenv("REXP_SWITCH_LISTING", `{"sessions":[{"session_id":"session:chosen","label":"already renamed"}]}`)
	out, err := runDiscovery(t, "bind", "demo", "--session", "session:chosen")
	if err != nil || !strings.Contains(out, "Bound project") {
		t.Fatalf("bind: %s %v", out, err)
	}
	data, _ := os.ReadFile(calls)
	if string(data) != "ls --json\n" {
		t.Fatalf("binding modified Rex: %s", data)
	}
	if _, err := runDiscovery(t, "switch", "demo"); err != nil {
		t.Fatal(err)
	}
	out, err = runDiscovery(t, "status", "demo", "--json")
	if err != nil || !strings.Contains(out, `"session_id":"session:chosen"`) {
		t.Fatalf("status: %s %v", out, err)
	}
	if _, err := runDiscovery(t, "bind", "demo", "--session", "already renamed"); err == nil {
		t.Fatal("accepted a label instead of a full ID")
	}
	out, err = runDiscovery(t, "bind", "demo", "--forget")
	if err != nil || !strings.Contains(out, "no Rex session was closed") {
		t.Fatalf("forget: %s %v", out, err)
	}
	// The old ID may still be alive and acquire the original display label;
	// explicit forgetting prevents legacy discovery from adopting it again.
	t.Setenv("REXP_SWITCH_LISTING", `{"sessions":[{"session_id":"session:chosen","label":"rexp:demo"}]}`)
	out, err = runDiscovery(t, "status", "demo", "--json")
	if err != nil || !strings.Contains(out, `"status":"missing"`) {
		t.Fatalf("forgotten workspace: %s %v", out, err)
	}
	data, _ = os.ReadFile(calls)
	if strings.Contains(string(data), "kill") {
		t.Fatal("forget closed a session")
	}
}

func TestBindCLIArgumentRules(t *testing.T) {
	calls := switchFixture(t)
	for _, args := range [][]string{
		{"bind"}, {"bind", "demo"}, {"bind", "demo", "--session", "id", "--forget"},
		{"bind", "demo", "--replace", "--forget"}, {"bind", "demo", "--incomplete", "--forget"},
		{"bind", "demo", "--file", "config.yaml", "--session", "id"},
	} {
		if _, err := runDiscovery(t, args...); err == nil {
			t.Fatalf("accepted %v", args)
		}
	}
	if _, err := os.Stat(calls); !os.IsNotExist(err) {
		t.Fatal("invalid binding contacted Rex")
	}
}
