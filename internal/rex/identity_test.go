package rex

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/itsjustaplant/rexp/internal/sessionstate"
	"github.com/itsjustaplant/rexp/internal/workspace"
)

type identityRex struct {
	mu                             sync.Mutex
	scope                          sessionstate.Scope
	live                           []StatusSession
	creates, selections, observers int
	fail                           string
}

func (fake *identityRex) run(_ context.Context, args ...string) ([]byte, error) {
	fake.mu.Lock()
	defer fake.mu.Unlock()
	if args[0] == fake.fail {
		return nil, errors.New("injected RPC failure")
	}
	switch args[0] {
	case "whoami":
		return json.Marshal(fake.scope)
	case "ls":
		live := append([]StatusSession{}, fake.live...)
		return json.Marshal(map[string]any{"sessions": live})
	case "new":
		fake.creates++
		id := fmt.Sprintf("session:%d", fake.creates)
		fake.live = append(fake.live, StatusSession{ID: id, Label: args[1]})
		return json.Marshal(map[string]any{"session_id": id, "initial_windows": []any{map[string]any{"block_ids": []string{"block:first"}}}})
	case "session":
		for i := range fake.live {
			if fake.live[i].ID == args[2] {
				fake.live[i].Label = args[3]
			}
		}
	case "kill":
		for i := range fake.live {
			if fake.live[i].ID == args[1] {
				fake.live = append(fake.live[:i], fake.live[i+1:]...)
				break
			}
		}
	case "do":
		fake.selections++
	}
	return nil, nil
}

func (fake *identityRex) client() *Client {
	return &Client{run: fake.run, persistent: true, watchManaged: func(context.Context, *workspace.Project, string, sessionstate.Ref) error {
		fake.mu.Lock()
		defer fake.mu.Unlock()
		fake.observers++
		return nil
	}}
}

func identityFixture(t *testing.T) (*identityRex, *Client, *workspace.Project) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	fake := &identityRex{scope: sessionstate.Scope{HostID: "host", Account: "account"}}
	return fake, fake.client(), projectFixture(t)
}

func TestIdentitySurvivesRenamesAcrossEveryWorkspaceCommand(t *testing.T) {
	fake, client, project := identityFixture(t)
	ctx := context.Background()
	first, err := client.Start(ctx, project, Options{Background: true})
	if err != nil {
		t.Fatal(err)
	}
	fake.live[0].Label = "agent changed the title"
	project.Hooks.BeforeStart = [][]string{{"must-not-run-on-reuse"}}
	project.Hooks.AfterStart = project.Hooks.BeforeStart
	project.Root = filepath.Join(t.TempDir(), "missing")
	second, err := fake.client().Start(ctx, project, Options{Background: true})
	if err != nil || !second.Reused || first.SessionID != second.SessionID {
		t.Fatalf("duplicate startup: %+v %v", second, err)
	}
	status, err := client.Status(ctx, project)
	if err != nil || status.Status != WorkspaceRunning || status.Sessions[0].Label != "agent changed the title" {
		t.Fatalf("renamed status: %+v %v", status, err)
	}
	for _, target := range []string{"demo", "rexp:demo", first.SessionID} {
		session, err := client.Switch(ctx, target, "")
		if err != nil || session.ID != first.SessionID {
			t.Fatalf("switch %q: %+v %v", target, session, err)
		}
	}
	selected, err := client.FindStopTarget(ctx, project)
	if err != nil {
		t.Fatal(err)
	}
	fake.live[0].Label = "renamed during confirmation"
	stopped, err := client.Stop(ctx, project, selected)
	if err != nil || !stopped || fake.creates != 1 || len(fake.live) != 0 {
		t.Fatalf("stop=%v creates=%d err=%v", stopped, fake.creates, err)
	}
	store, _ := client.sessionStore(ctx, project)
	if record, known, err := store.Read(); err != nil || record != nil || !known {
		t.Fatal("stop did not clear association")
	}
}

func TestMissingIdentityNeverFollowsRecycledLabels(t *testing.T) {
	fake, client, project := identityFixture(t)
	ctx := context.Background()
	first, err := client.Start(ctx, project, Options{Background: true})
	if err != nil {
		t.Fatal(err)
	}
	fake.live = []StatusSession{{ID: "session:unrelated", Label: SessionLabel(project)}}
	status, err := client.Status(ctx, project)
	if err != nil || status.Status != WorkspaceMissing {
		t.Fatalf("recycled label was adopted: %+v %v", status, err)
	}
	if _, err := client.Switch(ctx, "demo", ""); err == nil {
		t.Fatal("switched to recycled label")
	}
	store, _ := client.sessionStore(ctx, project)
	if record, _, _ := store.Read(); record != nil {
		t.Fatal("switch did not prune stale record")
	}
	second, err := client.Start(ctx, project, Options{Background: true})
	if err != nil || second.Reused || second.SessionID == first.SessionID || fake.creates != 2 {
		t.Fatalf("replacement: %+v %v", second, err)
	}
	// A delayed event for the old session cannot erase its replacement.
	if err := store.ClearIfSession(ctx, first.SessionID); err != nil {
		t.Fatal(err)
	}
	status, err = client.Status(ctx, project)
	if err != nil || status.Status != WorkspaceRunning || status.Sessions[0].ID != second.SessionID {
		t.Fatal("late event erased replacement")
	}
}

func TestSimultaneousStartsCreateOneSession(t *testing.T) {
	fake, _, project := identityFixture(t)
	var group sync.WaitGroup
	results := make(chan Result, 2)
	failures := make(chan error, 2)
	for range 2 {
		group.Add(1)
		go func() {
			defer group.Done()
			result, err := fake.client().Start(context.Background(), project, Options{Background: true})
			results <- result
			failures <- err
		}()
	}
	group.Wait()
	close(results)
	close(failures)
	for err := range failures {
		if err != nil {
			t.Fatal(err)
		}
	}
	for result := range results {
		if result.SessionID != "session:1" {
			t.Fatalf("wrong session %+v", result)
		}
	}
	if fake.creates != 1 || fake.observers != 1 {
		t.Fatalf("creates=%d observers=%d", fake.creates, fake.observers)
	}
}

func TestInterruptedCreationAndRenamedPartialStartupBlockRetries(t *testing.T) {
	fake, client, project := identityFixture(t)
	ctx := context.Background()
	fake.fail = "new"
	if _, err := client.Start(ctx, project, Options{Background: true}); err == nil {
		t.Fatal("ignored creation error")
	}
	fake.fail = ""
	if _, err := client.Start(ctx, project, Options{Background: true}); err == nil || !strings.Contains(err.Error(), "incomplete") {
		t.Fatalf("retried unknown creation: %v", err)
	}
	status, err := client.Status(ctx, project)
	if err != nil || status.Status != WorkspaceIncomplete || len(status.Sessions) != 0 {
		t.Fatalf("unknown creation: %+v %v", status, err)
	}
	if err := client.Forget(ctx, project); err != nil {
		t.Fatal(err)
	}
	fake.fail = "block"
	if _, err := client.Start(ctx, project, Options{Background: true}); err == nil {
		t.Fatal("ignored layout failure")
	}
	fake.fail = ""
	fake.live[0].Label = "renamed partial startup"
	if _, err := client.Start(ctx, project, Options{Background: true}); err == nil {
		t.Fatal("retried renamed incomplete session")
	}
	if fake.creates != 1 {
		t.Fatal("created duplicate incomplete session")
	}
}

func TestLegacyMigrationAndExplicitBind(t *testing.T) {
	fake, client, project := identityFixture(t)
	ctx := context.Background()
	fake.live = []StatusSession{{ID: "session:legacy", Label: SessionLabel(project)}}
	result, err := client.Start(ctx, project, Options{Background: true})
	if err != nil || !result.Reused || fake.creates != 0 {
		t.Fatalf("legacy migration: %+v %v", result, err)
	}
	fake.live = append(fake.live, StatusSession{ID: "session:chosen", Label: "already renamed"})
	if _, err := client.Bind(ctx, project, "session:chosen", false, false); err == nil {
		t.Fatal("replaced live association without --replace")
	}
	if _, err := client.Bind(ctx, project, "session:chosen", true, false); err != nil {
		t.Fatal(err)
	}
	status, err := client.Status(ctx, project)
	if err != nil || status.Sessions[0].ID != "session:chosen" {
		t.Fatal("bind failed")
	}
	if fake.selections != 0 {
		t.Fatal("bind selected an app session")
	}
}

func TestIdentityScopesLocalFilesHostsAndAccounts(t *testing.T) {
	fake, client, project := identityFixture(t)
	ctx := context.Background()
	first, _ := client.sessionStore(ctx, project)
	project.SourcePath = "/checkout/one/.rexp.yaml"
	localOne, _ := client.sessionStore(ctx, project)
	project.SourcePath = "/checkout/two/.rexp.yaml"
	localTwo, _ := client.sessionStore(ctx, project)
	fake.scope.HostID = "other-host"
	otherHost, _ := client.sessionStore(ctx, project)
	fake.scope.Account = "other-account"
	otherAccount, _ := client.sessionStore(ctx, project)
	seen := map[string]bool{}
	for _, store := range []*sessionstate.Store{first, localOne, localTwo, otherHost, otherAccount} {
		if seen[store.Path()] {
			t.Fatal("identity scope collision")
		}
		seen[store.Path()] = true
	}
	// Record creation is independent of labels for local configurations too.
	fake.live = []StatusSession{{ID: "local", Label: "custom title"}}
	if _, err := client.Bind(ctx, project, "local", false, false); err != nil {
		t.Fatal(err)
	}
	selected, err := client.SwitchProject(ctx, project, "")
	if err != nil || selected.ID != "local" {
		t.Fatalf("local switch: %+v %v", selected, err)
	}
	if _, err := os.Stat(first.Path()); !os.IsNotExist(err) {
		t.Fatal("wrote unrelated global record")
	}
}
