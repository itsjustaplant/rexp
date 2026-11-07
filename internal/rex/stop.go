package rex

import (
	"context"
	"errors"
	"fmt"

	"github.com/itsjustaplant/rexp/internal/workspace"
)

// StopTarget identifies the specific session the user approved closing.
type StopTarget struct{ ID, Label string }

func stopTarget(status WorkspaceStatus) (StopTarget, error) {
	if status.Status == WorkspaceMissing {
		return StopTarget{}, nil
	}
	if status.Status == WorkspaceAmbiguous {
		return StopTarget{}, errors.New(status.Reason)
	}
	if len(status.Sessions) != 1 {
		return StopTarget{}, errors.New(status.Reason)
	}
	return StopTarget{ID: status.Sessions[0].ID, Label: status.Sessions[0].Label}, nil
}

// FindStopTarget uses the recorded ID even after the session has been renamed.
func (client *Client) FindStopTarget(ctx context.Context, project *workspace.Project) (StopTarget, error) {
	status, err := client.Status(ctx, project)
	if err != nil {
		return StopTarget{}, err
	}
	return stopTarget(status)
}

// Stop rechecks the association under the same lock as startup. Title changes
// do not invalidate confirmation; a different session ID always does.
func (client *Client) Stop(ctx context.Context, project *workspace.Project, expected StopTarget) (bool, error) {
	if expected.ID == "" {
		return false, errors.New("a selected workspace session is required")
	}
	if project == nil {
		return false, errors.New("project is required")
	}
	store, err := client.sessionStore(ctx, project)
	if err != nil {
		return false, err
	}
	release, err := lockStore(ctx, store)
	if err != nil {
		return false, err
	}
	defer release()
	sessions, err := client.sessions(ctx)
	if err != nil {
		return false, err
	}
	status, err := resolve(project, store, sessions)
	if err != nil {
		return false, err
	}
	current, err := stopTarget(status)
	if err != nil {
		return false, err
	}
	if current.ID == "" {
		return false, pruneMissing(store)
	}
	if current.ID != expected.ID {
		return false, errors.New("workspace session changed while waiting; run stop again to select and confirm the current session")
	}
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if err := client.command(ctx, "kill", expected.ID); err != nil {
		return false, fmt.Errorf("stop session %s: %w; inspect Rex before retrying", expected.ID, err)
	}
	if store != nil {
		if err := store.Clear(); err != nil {
			return true, fmt.Errorf("session stopped, but could not clear its record: %w", err)
		}
	}
	return true, nil
}
