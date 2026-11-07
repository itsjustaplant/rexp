package rex

import (
	"context"
	"errors"

	"github.com/itsjustaplant/rexp/internal/workspace"
)

type WorkspaceState string

const (
	WorkspaceRunning    WorkspaceState = "running"
	WorkspaceMissing    WorkspaceState = "missing"
	WorkspaceIncomplete WorkspaceState = "incomplete"
	WorkspaceAmbiguous  WorkspaceState = "ambiguous"
)

type StatusSession struct {
	ID    string `json:"session_id"`
	Label string `json:"label"`
}

// WorkspaceStatus describes session identity/existence, not program health.
type WorkspaceStatus struct {
	Project       string          `json:"project"`
	ExpectedLabel string          `json:"expected_label"`
	Status        WorkspaceState  `json:"status"`
	Sessions      []StatusSession `json:"sessions"`
	Reason        string          `json:"reason,omitempty"`
}

// Status reads the association and Rex's live list without modifying either.
func (client *Client) Status(ctx context.Context, project *workspace.Project) (WorkspaceStatus, error) {
	if project == nil {
		return WorkspaceStatus{}, errors.New("project is required")
	}
	store, err := client.sessionStore(ctx, project)
	if err != nil {
		return WorkspaceStatus{}, err
	}
	sessions, err := client.sessions(ctx)
	if err != nil {
		return WorkspaceStatus{}, err
	}
	result, err := resolve(project, store, sessions)
	if err != nil {
		return result, err
	}
	return result, ctx.Err()
}
