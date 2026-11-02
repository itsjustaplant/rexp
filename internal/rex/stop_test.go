package rex

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"rexp/internal/workspace"
)

func stopListing(id, label string) string {
	return fmt.Sprintf(`{"sessions":[{"session_id":%q,"label":%q}]}`, id, label)
}

func TestStopClosesOnlyExactWorkspaceID(t *testing.T) {
	for _, pending := range []bool{false, true} {
		t.Run(fmt.Sprint(pending), func(t *testing.T) {
			project := projectFixture(t)
			project.Root = "/missing/source"
			label := SessionLabel(project)
			if pending {
				label += ":starting"
			}
			fake := &fakeRex{listing: stopListing("session:target", label)}
			client := &Client{run: fake.run}
			target, err := client.FindStopTarget(context.Background(), project)
			if err != nil {
				t.Fatal(err)
			}
			stopped, err := client.Stop(context.Background(), project, target)
			if err != nil || !stopped {
				t.Fatalf("stopped=%v err=%v", stopped, err)
			}
			want := [][]string{{"ls", "--json"}, {"ls", "--json"}, {"kill", "session:target"}}
			if !reflect.DeepEqual(fake.calls, want) {
				t.Fatalf("unexpected calls: %v", fake.calls)
			}
		})
	}
}

func TestStopRejectsAmbiguousSessions(t *testing.T) {
	for _, listing := range []string{
		`{"sessions":[{"session_id":"session:1","label":"rexp:DEMO"}]}`,
		`{"sessions":[{"session_id":"session:1","label":"rexp:demo:STARTING"}]}`,
		`{"sessions":[{"session_id":"session:1","label":"rexp:demo"},{"session_id":"session:2","label":"rexp:demo"}]}`,
		`{"sessions":[{"session_id":"session:1","label":"rexp:demo"},{"session_id":"session:2","label":"rexp:demo:starting"}]}`,
		`{"sessions":[{"label":"rexp:demo"}]}`,
		`invalid json`,
	} {
		fake := &fakeRex{listing: listing}
		client := &Client{run: fake.run}
		if _, err := client.FindStopTarget(context.Background(), projectFixture(t)); err == nil {
			t.Fatalf("accepted %s", listing)
		}
		if len(fake.calls) != 1 {
			t.Fatalf("mutated ambiguous workspace: %v", fake.calls)
		}
	}
}

func TestStopRechecksAfterConfirmation(t *testing.T) {
	for _, test := range []struct {
		name, listing string
		wantError     bool
	}{
		{"closed", `{"sessions":[]}`, false},
		{"replacement", stopListing("session:replacement", "rexp:demo"), true},
		{"renamed", stopListing("session:original", "different"), false},
		{"pending-changed", stopListing("session:original", "rexp:demo:starting"), true},
	} {
		t.Run(test.name, func(t *testing.T) {
			fake := &fakeRex{listing: test.listing}
			client := &Client{run: fake.run}
			stopped, err := client.Stop(context.Background(), projectFixture(t), StopTarget{ID: "session:original", Label: "rexp:demo"})
			if stopped || (err != nil) != test.wantError {
				t.Fatalf("stopped=%v err=%v", stopped, err)
			}
			if len(fake.calls) != 1 {
				t.Fatal("killed a stale or replacement target")
			}
		})
	}
}

func TestStopLocalIdentityAndMissingWorkspace(t *testing.T) {
	project := projectFixture(t)
	project.SourcePath = "/local/.rexp.yaml"
	fake := &fakeRex{listing: stopListing("session:global", "rexp:demo")}
	client := &Client{run: fake.run}
	target, err := client.FindStopTarget(context.Background(), project)
	if err != nil || target.ID != "" {
		t.Fatalf("matched global workspace: %+v %v", target, err)
	}
	fake.listing = stopListing("session:local", SessionLabel(project))
	target, err = client.FindStopTarget(context.Background(), project)
	if err != nil || target.ID != "session:local" {
		t.Fatalf("local target %+v %v", target, err)
	}
}

func TestStopFailureIsNotRetried(t *testing.T) {
	fake := &fakeRex{listing: stopListing("session:target", "rexp:demo"), fail: "kill"}
	stopped, err := (&Client{run: fake.run}).Stop(context.Background(), projectFixture(t), StopTarget{ID: "session:target", Label: "rexp:demo"})
	if stopped || err == nil || !strings.Contains(err.Error(), "inspect Rex before retrying") {
		t.Fatalf("stopped=%v err=%v", stopped, err)
	}
	if len(fake.calls) != 2 {
		t.Fatalf("retried kill: %v", fake.calls)
	}
}

func TestStopCancellationPreventsKill(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	client := &Client{run: func(_ context.Context, args ...string) ([]byte, error) {
		if args[0] != "ls" {
			t.Fatal("kill after cancellation")
		}
		cancel()
		return []byte(stopListing("session:target", "rexp:demo")), nil
	}}
	stopped, err := client.Stop(ctx, &workspace.Project{Name: "demo"}, StopTarget{ID: "session:target", Label: "rexp:demo"})
	if stopped || !errors.Is(err, context.Canceled) {
		t.Fatalf("stopped=%v err=%v", stopped, err)
	}
}
