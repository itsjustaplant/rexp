package rex

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/itsjustaplant/rexp/internal/hooks"
	"github.com/itsjustaplant/rexp/internal/sessionstate"
	"github.com/itsjustaplant/rexp/internal/workspace"
)

func workspaceKey(project *workspace.Project) string {
	if project.SourcePath != "" {
		return "file:" + project.SourcePath
	}
	return "global:" + project.Name
}

func (client *Client) sessionStore(ctx context.Context, project *workspace.Project) (*sessionstate.Store, error) {
	if project == nil {
		return nil, errors.New("project is required")
	}
	if !client.persistent {
		return nil, nil // Transport-only clients can exercise legacy resolution.
	}
	var scope sessionstate.Scope
	if err := client.json(ctx, &scope, "whoami", "--json"); err != nil {
		return nil, err
	}
	if scope.HostID == "" || scope.Account == "" {
		return nil, errors.New("Rex did not return host_id and account; update Rex to use persistent workspace identities")
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}
	return sessionstate.Open(home, sessionstate.Ref{Scope: scope, Workspace: workspaceKey(project)})
}

func lockStore(ctx context.Context, store *sessionstate.Store) (func(), error) {
	if store == nil {
		return func() {}, nil
	}
	return store.Lock(ctx)
}

func (client *Client) sessions(ctx context.Context) ([]StatusSession, error) {
	var listing struct {
		Sessions *[]StatusSession `json:"sessions"`
	}
	if err := client.json(ctx, &listing, "ls", "--json"); err != nil {
		return nil, err
	}
	if listing.Sessions == nil {
		return nil, errors.New("Rex returned a session listing without a sessions array")
	}
	seen := map[string]bool{}
	for _, session := range *listing.Sessions {
		if session.ID == "" || seen[session.ID] {
			return nil, errors.New("Rex returned missing or duplicate session IDs")
		}
		seen[session.ID] = true
	}
	return *listing.Sessions, ctx.Err()
}

// resolve treats a stored ID as authoritative. A known workspace never falls
// back to a recycled label after its recorded session disappears.
func resolve(project *workspace.Project, store *sessionstate.Store, sessions []StatusSession) (WorkspaceStatus, error) {
	label := SessionLabel(project)
	result := WorkspaceStatus{Project: project.Name, ExpectedLabel: label, Status: WorkspaceMissing, Sessions: []StatusSession{}}
	if store != nil {
		record, known, err := store.Read()
		if err != nil {
			return result, err
		}
		if record != nil {
			if record.State == "creating" {
				result.Status = WorkspaceIncomplete
				result.Reason = "session creation was interrupted before its ID was recorded; inspect Rex, then use rexp bind --session <id> or rexp bind --forget with this workspace target"
				return result, nil
			}
			for _, session := range sessions {
				if session.ID == record.Session {
					result.Sessions = append(result.Sessions, session)
					result.Status = WorkspaceRunning
					if record.State == "starting" {
						result.Status = WorkspaceIncomplete
						result.Reason = fmt.Sprintf("session %s has an incomplete startup", record.Session)
					}
					return result, nil
				}
			}
			return result, nil
		}
		if known {
			return result, nil
		}
	}
	// One-time migration for workspaces created before persistent records.
	collision := false
	for _, session := range sessions {
		if !strings.EqualFold(session.Label, label) && !strings.EqualFold(session.Label, label+":starting") {
			continue
		}
		result.Sessions = append(result.Sessions, session)
		if session.Label != label && session.Label != label+":starting" {
			collision = true
		}
	}
	sort.Slice(result.Sessions, func(i, j int) bool {
		a, b := result.Sessions[i], result.Sessions[j]
		if a.Label != b.Label {
			return a.Label < b.Label
		}
		return a.ID < b.ID
	})
	switch {
	case collision:
		result.Status = WorkspaceAmbiguous
		result.Reason = "case-conflicting session labels; inspect Rex and bind the intended full session ID"
	case len(result.Sessions) > 1:
		result.Status = WorkspaceAmbiguous
		result.Reason = "multiple matching sessions; inspect Rex and bind the intended full session ID"
	case len(result.Sessions) == 1:
		result.Status = WorkspaceRunning
		if result.Sessions[0].Label == label+":starting" {
			result.Status = WorkspaceIncomplete
		}
	}
	return result, nil
}

func pruneMissing(store *sessionstate.Store) error {
	if store == nil {
		return nil
	}
	record, _, err := store.Read()
	if err != nil || record == nil {
		return err
	}
	return store.Clear()
}

func (client *Client) observe(ctx context.Context, project *workspace.Project, session string, store *sessionstate.Store) error {
	if client.watchManaged != nil {
		return client.watchManaged(ctx, project, session, store.Ref)
	}
	return hooks.WatchManaged(ctx, project, session, store.Ref)
}

// adopt records a legacy association without installing user close hooks on an
// already running session. The internal observer only clears its record.
func (client *Client) adopt(ctx context.Context, project *workspace.Project, result WorkspaceStatus, store *sessionstate.Store) error {
	if store == nil {
		return nil
	}
	record, _, err := store.Read()
	if err != nil || record != nil {
		return err
	}
	state := "ready"
	if result.Status == WorkspaceIncomplete {
		state = "starting"
	}
	session := result.Sessions[0].ID
	if err := store.Save(session, state); err != nil {
		return err
	}
	if err := client.observe(ctx, &workspace.Project{Name: project.Name, Root: project.Root}, session, store); err != nil {
		return fmt.Errorf("workspace association saved, but close observer failed: %w", err)
	}
	return nil
}
