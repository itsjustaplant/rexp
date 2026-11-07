package rex

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/itsjustaplant/rexp/internal/sessionstate"
	"github.com/itsjustaplant/rexp/internal/workspace"
)

// Switch selects an existing session by full ID, label, or global project
// shorthand. It never loads configurations, creates sessions, or runs hooks.
func (client *Client) Switch(ctx context.Context, target, appClient string) (StatusSession, error) {
	if strings.TrimSpace(target) == "" {
		return StatusSession{}, errors.New("a session name or ID is required")
	}
	sessions, err := client.sessions(ctx)
	if err != nil {
		return StatusSession{}, err
	}
	var byID StatusSession
	for _, session := range sessions {
		if session.ID == target {
			byID = session
		}
	}
	selected := byID
	var store *sessionstate.Store
	var project *workspace.Project
	if selected.ID == "" && client.persistent {
		name := strings.TrimPrefix(target, "rexp:")
		if workspace.ValidateName(name) == nil {
			project = &workspace.Project{Name: name}
			store, err = client.sessionStore(ctx, project)
			if err != nil {
				return selected, err
			}
			release, lockErr := lockStore(ctx, store)
			if lockErr != nil {
				return selected, lockErr
			}
			defer release()
			// Re-read live sessions after acquiring the startup lock.
			sessions, err = client.sessions(ctx)
			if err != nil {
				return selected, err
			}
			record, known, readErr := store.Read()
			if readErr != nil {
				return selected, readErr
			}
			if record != nil || known {
				status, resolveErr := resolve(project, store, sessions)
				if resolveErr != nil {
					return selected, resolveErr
				}
				if len(status.Sessions) == 0 {
					if status.Status == WorkspaceMissing {
						if err := pruneMissing(store); err != nil {
							return selected, err
						}
					}
					return selected, workspaceSelectionError(name, status.Reason)
				}
				selected = status.Sessions[0]
			}
		}
	}
	if selected.ID == "" {
		for _, session := range sessions {
			if !strings.EqualFold(session.Label, target) && !strings.EqualFold(session.Label, "rexp:"+target) {
				continue
			}
			if session.Label != target && session.Label != "rexp:"+target {
				return StatusSession{}, fmt.Errorf("session label %q differs only in case from %q; use its full session ID", session.Label, target)
			}
			if selected.ID != "" {
				return StatusSession{}, fmt.Errorf("multiple sessions match %q; use a full session ID from rex ls --json", target)
			}
			selected = session
		}
	}
	if selected.ID == "" {
		return StatusSession{}, fmt.Errorf("no running Rex session matches %q; switch does not start workspaces", target)
	}
	if store != nil && selected.Label == SessionLabel(project) {
		status := WorkspaceStatus{Status: WorkspaceRunning, Sessions: []StatusSession{selected}}
		if err := client.adopt(ctx, project, status, store); err != nil {
			return selected, err
		}
	}
	if err := ctx.Err(); err != nil {
		return StatusSession{}, err
	}
	// Select by the observed ID so a replacement with the same label cannot be
	// selected if this session disappears. A failed selection is never retried.
	if err := client.selectSessionID(ctx, selected.ID, appClient); err != nil {
		return StatusSession{}, fmt.Errorf("could not switch to Rex session %s: %w; enable Remote Control or specify --client", selected.ID, err)
	}
	return selected, nil
}

// SwitchProject handles local configuration identities without guessing a
// project name from its display label or path hash.
func (client *Client) SwitchProject(ctx context.Context, project *workspace.Project, appClient string) (StatusSession, error) {
	store, err := client.sessionStore(ctx, project)
	if err != nil {
		return StatusSession{}, err
	}
	release, err := lockStore(ctx, store)
	if err != nil {
		return StatusSession{}, err
	}
	defer release()
	sessions, err := client.sessions(ctx)
	if err != nil {
		return StatusSession{}, err
	}
	status, err := resolve(project, store, sessions)
	if err != nil {
		return StatusSession{}, err
	}
	if status.Status == WorkspaceAmbiguous {
		return StatusSession{}, errors.New(status.Reason)
	}
	if len(status.Sessions) != 1 {
		if store != nil && status.Status == WorkspaceMissing {
			if err := pruneMissing(store); err != nil {
				return StatusSession{}, err
			}
		}
		return StatusSession{}, workspaceSelectionError(project.Name, status.Reason)
	}
	if err := client.adopt(ctx, project, status, store); err != nil {
		return StatusSession{}, err
	}
	session := status.Sessions[0]
	if err := client.selectSessionID(ctx, session.ID, appClient); err != nil {
		return session, err
	}
	return session, nil
}

func workspaceSelectionError(name, reason string) error {
	message := fmt.Sprintf("no running Rex session for workspace %q", name)
	if reason != "" {
		message += "; " + reason
	}
	return errors.New(message)
}
