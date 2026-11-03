package rex

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"
)

func TestSwitchResolvesExistingSessions(t *testing.T) {
	listing := `{"sessions":[{"session_id":"session:project","label":"rexp:demo"},{"session_id":"session:plain","label":"Personal work"},{"session_id":"session:pending","label":"rexp:other:starting"}]}`
	for _, test := range []struct{ target, id, label string }{
		{"demo", "session:project", "rexp:demo"},
		{"rexp:demo", "session:project", "rexp:demo"},
		{"session:project", "session:project", "rexp:demo"},
		{"Personal work", "session:plain", "Personal work"},
		{"rexp:other:starting", "session:pending", "rexp:other:starting"},
	} {
		t.Run(test.target, func(t *testing.T) {
			fake := &fakeRex{listing: listing}
			session, err := (&Client{run: fake.run}).Switch(context.Background(), test.target, "macbook")
			if err != nil || session.ID != test.id || session.Label != test.label {
				t.Fatalf("result=%+v err=%v", session, err)
			}
			want := [][]string{{"ls", "--json"}, {"-C", "macbook", "do", "session.select", "session_id=" + test.id}}
			if !reflect.DeepEqual(fake.calls, want) {
				t.Fatalf("unexpected calls: %v", fake.calls)
			}
		})
	}
}

func TestSwitchMissingAndAmbiguousTargetsDoNotMutate(t *testing.T) {
	for _, test := range []struct{ name, listing, hint string }{
		{"missing", `{"sessions":[]}`, "no running Rex session"},
		{"incomplete", `{"sessions":[{"session_id":"s1","label":"rexp:demo:starting"}]}`, "no running Rex session"},
		{"duplicates", `{"sessions":[{"session_id":"s1","label":"rexp:demo"},{"session_id":"s2","label":"rexp:demo"}]}`, "multiple sessions"},
		{"raw-and-project", `{"sessions":[{"session_id":"s1","label":"demo"},{"session_id":"s2","label":"rexp:demo"}]}`, "multiple sessions"},
		{"case", `{"sessions":[{"session_id":"s1","label":"rexp:DEMO"}]}`, "differs only in case"},
		{"case-conflict", `{"sessions":[{"session_id":"s1","label":"rexp:demo"},{"session_id":"s2","label":"rexp:DEMO"}]}`, "differs only in case"},
	} {
		t.Run(test.name, func(t *testing.T) {
			fake := &fakeRex{listing: test.listing}
			if _, err := (&Client{run: fake.run}).Switch(context.Background(), "demo", ""); err == nil || !strings.Contains(err.Error(), test.hint) {
				t.Fatalf("expected %q: %v", test.hint, err)
			}
			if !reflect.DeepEqual(fake.calls, [][]string{{"ls", "--json"}}) {
				t.Fatalf("mutating requests: %v", fake.calls)
			}
		})
	}
}

func TestSwitchIDDisambiguatesAndTakesPrecedence(t *testing.T) {
	fake := &fakeRex{listing: `{"sessions":[{"session_id":"session:1","label":"rexp:demo"},{"session_id":"session:2","label":"rexp:demo"},{"session_id":"session:3","label":"session:1"}]}`}
	session, err := (&Client{run: fake.run}).Switch(context.Background(), "session:1", "")
	if err != nil || session.ID != "session:1" {
		t.Fatalf("result=%+v err=%v", session, err)
	}
	if !reflect.DeepEqual(fake.calls[1], []string{"do", "session.select", "session_id=session:1"}) {
		t.Fatalf("selection: %v", fake.calls)
	}
}

func TestSwitchRejectsBadResponsesAndFailures(t *testing.T) {
	for _, listing := range []string{`{}`, `null`, `{"sessions":null}`, `{"sessions":{}}`, `not JSON`, `{"sessions":[{"label":"demo"}]}`, `{"sessions":[{"session_id":"s1"},{"session_id":"s1"}]}`} {
		fake := &fakeRex{listing: listing}
		if _, err := (&Client{run: fake.run}).Switch(context.Background(), "demo", ""); err == nil {
			t.Fatalf("accepted %s", listing)
		}
		if len(fake.calls) != 1 {
			t.Fatalf("continued after malformed response: %v", fake.calls)
		}
	}
	for _, failure := range []string{"ls", "do"} {
		fake := &fakeRex{listing: `{"sessions":[{"session_id":"s1","label":"rexp:demo"}]}`, fail: failure}
		if _, err := (&Client{run: fake.run}).Switch(context.Background(), "demo", ""); err == nil {
			t.Fatal("ignored Rex failure")
		}
		if len(fake.calls) > 2 {
			t.Fatalf("retried failed switch: %v", fake.calls)
		}
	}
}

func TestSwitchCancellationAndEmptyTarget(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	client := &Client{run: func(context.Context, ...string) ([]byte, error) {
		cancel()
		return []byte(`{"sessions":[{"session_id":"s1","label":"demo"}]}`), nil
	}}
	if _, err := client.Switch(ctx, "demo", ""); !errors.Is(err, context.Canceled) {
		t.Fatalf("cancellation ignored: %v", err)
	}
	fake := &fakeRex{}
	if _, err := (&Client{run: fake.run}).Switch(context.Background(), " ", ""); err == nil || len(fake.calls) != 0 {
		t.Fatalf("empty target contacted Rex: %v", err)
	}
}
