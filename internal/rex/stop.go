package rex

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"rexp/internal/workspace"
)

// StopTarget identifies one complete or incomplete workspace session.
// An empty ID means the workspace is not running.
type StopTarget struct {
	ID    string
	Label string
}

// FindStopTarget never guesses between duplicate or case-conflicting labels.
// It does not require pane directories to exist and never executes hooks.
func (client *Client) FindStopTarget(ctx context.Context, project *workspace.Project) (StopTarget, error) {
	if project == nil {
		return StopTarget{}, errors.New("project is required")
	}
	label := SessionLabel(project)
	var listing struct {
		Sessions []struct {
			ID    string `json:"session_id"`
			Label string `json:"label"`
		} `json:"sessions"`
	}
	if err := client.json(ctx, &listing, "ls", "--json"); err != nil {
		return StopTarget{}, err
	}
	var target StopTarget
	for _, session := range listing.Sessions {
		if !strings.EqualFold(session.Label, label) && !strings.EqualFold(session.Label, label+":starting") {
			continue
		}
		if session.Label != label && session.Label != label+":starting" {
			return StopTarget{}, fmt.Errorf("session label %q differs only in case from project %q; rename it before stopping", session.Label, label)
		}
		if session.ID == "" {
			return StopTarget{}, errors.New("Rex returned a workspace session without an ID")
		}
		if target.ID != "" {
			return StopTarget{}, fmt.Errorf("multiple sessions match project %q; inspect and rename duplicates in Rex before stopping", project.Name)
		}
		target = StopTarget{ID: session.ID, Label: session.Label}
	}
	return target, nil
}

// Stop rechecks identity after confirmation, then closes only the selected ID.
// A replacement session must receive its own confirmation. No kill is retried.
func (client *Client) Stop(ctx context.Context, project *workspace.Project, expected StopTarget) (bool, error) {
	if expected.ID == "" {
		return false, errors.New("a selected workspace session is required")
	}
	current, err := client.FindStopTarget(ctx, project)
	if err != nil {
		return false, err
	}
	if current.ID == "" {
		return false, nil
	}
	if current != expected {
		return false, errors.New("workspace session changed while waiting; run stop again to select and confirm the current session")
	}
	if err := ctx.Err(); err != nil {
		return false, err
	}
	if err := client.command(ctx, "kill", expected.ID); err != nil {
		return false, fmt.Errorf("stop session %s: %w; inspect Rex before retrying", expected.ID, err)
	}
	return true, nil
}
