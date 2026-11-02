package rex

import (
	"context"
	"errors"
	"sort"
	"strings"

	"rexp/internal/workspace"
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

// Status reads one session-list snapshot. It never selects a session, checks
// local pane directories, executes hooks, or modifies configuration/session state.
func (client *Client) Status(ctx context.Context, project *workspace.Project) (WorkspaceStatus, error) {
	if project == nil {
		return WorkspaceStatus{}, errors.New("project is required")
	}
	var listing struct {
		Sessions *[]StatusSession `json:"sessions"`
	}
	if err := client.json(ctx, &listing, "ls", "--json"); err != nil {
		return WorkspaceStatus{}, err
	}
	if listing.Sessions == nil {
		return WorkspaceStatus{}, errors.New("Rex returned a session listing without a sessions array")
	}
	label := SessionLabel(project)
	result := WorkspaceStatus{Project: project.Name, ExpectedLabel: label, Status: WorkspaceMissing, Sessions: []StatusSession{}}
	collision := false
	for _, session := range *listing.Sessions {
		if !strings.EqualFold(session.Label, label) && !strings.EqualFold(session.Label, label+":starting") {
			continue
		}
		if session.ID == "" {
			return WorkspaceStatus{}, errors.New("Rex returned a matching workspace session without an ID")
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
		result.Reason = "case-conflicting session labels; inspect and rename them in Rex"
	case len(result.Sessions) > 1:
		result.Status = WorkspaceAmbiguous
		result.Reason = "multiple matching sessions; inspect and rename duplicates in Rex"
	case len(result.Sessions) == 1:
		result.Status = WorkspaceRunning
		if result.Sessions[0].Label == label+":starting" {
			result.Status = WorkspaceIncomplete
		}
	}
	return result, ctx.Err()
}
