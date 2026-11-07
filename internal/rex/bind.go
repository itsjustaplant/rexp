package rex

import (
	"context"
	"errors"
	"fmt"

	"github.com/itsjustaplant/rexp/internal/workspace"
)

// Bind explicitly associates an existing ID without renaming it, launching
// programs, or installing user hooks on that session.
func (client *Client) Bind(ctx context.Context, project *workspace.Project, sessionID string, replace, incomplete bool) (StatusSession, error) {
	if project == nil || sessionID == "" {
		return StatusSession{}, errors.New("project and full session ID are required")
	}
	store, err := client.sessionStore(ctx, project)
	if err != nil {
		return StatusSession{}, err
	}
	if store == nil {
		return StatusSession{}, errors.New("persistent session storage is required")
	}
	release, err := store.Lock(ctx)
	if err != nil {
		return StatusSession{}, err
	}
	defer release()
	sessions, err := client.sessions(ctx)
	if err != nil {
		return StatusSession{}, err
	}
	var selected StatusSession
	for _, session := range sessions {
		if session.ID == sessionID {
			selected = session
			break
		}
	}
	if selected.ID == "" {
		return selected, fmt.Errorf("no live session has full ID %q", sessionID)
	}
	record, _, err := store.Read()
	if err != nil {
		return selected, err
	}
	if record != nil && record.Session != "" && record.Session != selected.ID && !replace {
		return selected, fmt.Errorf("workspace is already associated with %s; use --replace to change the association without closing it", record.Session)
	}
	state := "ready"
	if incomplete {
		state = "starting"
	}
	if err := store.Save(selected.ID, state); err != nil {
		return selected, err
	}
	if record == nil || record.Session != selected.ID {
		if err := client.observe(ctx, &workspace.Project{Name: project.Name, Root: project.Root}, selected.ID, store); err != nil {
			return selected, fmt.Errorf("association saved, but close observer failed: %w", err)
		}
	}
	return selected, nil
}

// Forget is an explicit recovery operation. It never closes a Rex session.
func (client *Client) Forget(ctx context.Context, project *workspace.Project) error {
	store, err := client.sessionStore(ctx, project)
	if err != nil {
		return err
	}
	if store == nil {
		return errors.New("persistent session storage is required")
	}
	release, err := store.Lock(ctx)
	if err != nil {
		return err
	}
	defer release()
	return store.Clear()
}
