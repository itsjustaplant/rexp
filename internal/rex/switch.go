package rex

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// Switch selects an existing session by full ID, label, or global project
// shorthand. It never loads configurations, creates sessions, or runs hooks.
func (client *Client) Switch(ctx context.Context, target, appClient string) (StatusSession, error) {
	if strings.TrimSpace(target) == "" {
		return StatusSession{}, errors.New("a session name or ID is required")
	}
	var listing struct {
		Sessions *[]StatusSession `json:"sessions"`
	}
	if err := client.json(ctx, &listing, "ls", "--json"); err != nil {
		return StatusSession{}, err
	}
	if listing.Sessions == nil {
		return StatusSession{}, errors.New("Rex returned a session listing without a sessions array")
	}
	seen := map[string]bool{}
	var byID StatusSession
	for _, session := range *listing.Sessions {
		if session.ID == "" || seen[session.ID] {
			return StatusSession{}, errors.New("Rex returned missing or duplicate session IDs")
		}
		seen[session.ID] = true
		if session.ID == target {
			byID = session
		}
	}
	selected := byID
	if selected.ID == "" {
		for _, session := range *listing.Sessions {
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
