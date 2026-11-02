package rex

import (
	"context"
	"errors"
	"reflect"
	"testing"
)

func TestWorkspaceStatusStates(t *testing.T) {
	for _, test := range []struct {
		name, listing string
		state         WorkspaceState
		matches       int
	}{
		{"running", stopListing("session:ready", "rexp:demo"), WorkspaceRunning, 1},
		{"missing", `{"sessions":[]}`, WorkspaceMissing, 0},
		{"unrelated", stopListing("session:other", "rexp:demo-other"), WorkspaceMissing, 0},
		{"incomplete", stopListing("session:pending", "rexp:demo:starting"), WorkspaceIncomplete, 1},
		{"case-conflict", stopListing("session:case", "rexp:DEMO"), WorkspaceAmbiguous, 1},
		{"pending-case-conflict", stopListing("session:case", "rexp:demo:STARTING"), WorkspaceAmbiguous, 1},
		{"duplicate", `{"sessions":[{"session_id":"session:2","label":"rexp:demo"},{"session_id":"session:1","label":"rexp:demo"}]}`, WorkspaceAmbiguous, 2},
		{"ready-and-pending", `{"sessions":[{"session_id":"session:1","label":"rexp:demo:starting"},{"session_id":"session:2","label":"rexp:demo"}]}`, WorkspaceAmbiguous, 2},
	} {
		t.Run(test.name, func(t *testing.T) {
			fake := &fakeRex{listing: test.listing}
			project := projectFixture(t)
			project.Root = "/no-such-directory"
			result, err := (&Client{run: fake.run}).Status(context.Background(), project)
			if err != nil || result.Status != test.state || len(result.Sessions) != test.matches {
				t.Fatalf("result=%+v err=%v", result, err)
			}
			if result.Project != "demo" || result.ExpectedLabel != "rexp:demo" || result.Sessions == nil {
				t.Fatalf("incomplete result %+v", result)
			}
			if test.state == WorkspaceAmbiguous && result.Reason == "" {
				t.Fatal("missing ambiguity reason")
			}
			if !reflect.DeepEqual(fake.calls, [][]string{{"ls", "--json"}}) {
				t.Fatalf("mutating calls: %v", fake.calls)
			}
			if test.name == "duplicate" && result.Sessions[0].ID != "session:1" {
				t.Fatal("unstable session order")
			}
		})
	}
}

func TestWorkspaceStatusRejectsInvalidResponses(t *testing.T) {
	for _, listing := range []string{`{}`, `null`, `{"sessions":null}`, `{"sessions":{}}`, `not JSON`, `{"sessions":[{"label":"rexp:demo"}]}`} {
		fake := &fakeRex{listing: listing}
		if _, err := (&Client{run: fake.run}).Status(context.Background(), projectFixture(t)); err == nil {
			t.Fatalf("accepted %s", listing)
		}
		if len(fake.calls) != 1 {
			t.Fatal("retried or changed a workspace")
		}
	}
	fake := &fakeRex{fail: "ls"}
	if _, err := (&Client{run: fake.run}).Status(context.Background(), projectFixture(t)); err == nil {
		t.Fatal("converted connection failure to missing")
	}
}

func TestWorkspaceStatusSeparatesLocalAndGlobal(t *testing.T) {
	project := projectFixture(t)
	project.SourcePath = "/checkout/.rexp.yaml"
	fake := &fakeRex{listing: stopListing("session:global", "rexp:demo")}
	client := &Client{run: fake.run}
	result, err := client.Status(context.Background(), project)
	if err != nil || result.Status != WorkspaceMissing {
		t.Fatalf("matched global project: %+v %v", result, err)
	}
	fake.listing = stopListing("session:local", SessionLabel(project))
	result, err = client.Status(context.Background(), project)
	if err != nil || result.Status != WorkspaceRunning || result.Sessions[0].ID != "session:local" {
		t.Fatalf("local status %+v %v", result, err)
	}
}

func TestWorkspaceStatusContextAndNilProject(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	client := &Client{run: func(context.Context, ...string) ([]byte, error) { cancel(); return []byte(`{"sessions":[]}`), nil }}
	if _, err := client.Status(ctx, projectFixture(t)); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancel ignored: %v", err)
	}
	if _, err := client.Status(context.Background(), nil); err == nil {
		t.Fatal("accepted nil project")
	}
}
