// Package rex launches workspaces through the installed Rex CLI.
package rex

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"rexp/internal/panerunner"
	"rexp/internal/workspace"
)

type Client struct {
	run func(context.Context, ...string) ([]byte, error)
}
type Options struct {
	Background bool
	Client     string
}
type Result struct {
	SessionID string
	Reused    bool
}

func NewClient() *Client { return &Client{run: execute} }

func execute(ctx context.Context, args ...string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "rex", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if err != nil {
		if errors.Is(err, exec.ErrNotFound) {
			return nil, errors.New("Rex CLI not found on PATH; install Rex before starting a workspace")
		}
		return nil, fmt.Errorf("rex %s: %w: %s", args[0], err, strings.TrimSpace(stderr.String()))
	}
	return output, nil
}

// Start expects a validated project with resolved pane directories. Mutating
// calls are never retried after errors; partial workspaces remain inspectable.
func (client *Client) Start(ctx context.Context, project *workspace.Project, options Options) (Result, error) {
	if err := preflight(project); err != nil {
		return Result{}, err
	}
	label := "rexp:" + project.Name
	pending := label + ":starting"
	var listing struct {
		Sessions []struct {
			ID    string `json:"session_id"`
			Label string `json:"label"`
		} `json:"sessions"`
	}
	if err := client.json(ctx, &listing, "ls", "--json"); err != nil {
		return Result{}, err
	}
	result := Result{}
	for _, session := range listing.Sessions {
		if strings.EqualFold(session.Label, pending) {
			return Result{}, fmt.Errorf("session %s has an incomplete startup (%q); inspect it in Rex before closing it or renaming it", session.ID, session.Label)
		}
		if !strings.EqualFold(session.Label, label) {
			continue
		}
		if session.Label != label {
			return Result{}, fmt.Errorf("session label %q differs only in case from %q; rename it to avoid selecting the wrong project", session.Label, label)
		}
		if result.SessionID != "" {
			return Result{}, fmt.Errorf("multiple sessions are named %q; rename duplicates in Rex before starting", label)
		}
		if session.ID == "" {
			return Result{}, errors.New("Rex returned a session without an ID")
		}
		result = Result{SessionID: session.ID, Reused: true}
	}
	if result.Reused {
		return result, client.selectSession(ctx, result, options)
	}
	commands, err := launchCommands(project)
	if err != nil {
		return Result{}, err
	}

	first := firstPane(project.Tabs[0].Layout)
	args := withCommand([]string{"new", pending, "--window", project.Tabs[0].Name, "--cwd", first.Cwd, "--keep-open", "--json"}, commands[first.Pane])
	var created struct {
		SessionID string `json:"session_id"`
		Windows   []struct {
			Blocks []string `json:"block_ids"`
		} `json:"initial_windows"`
	}
	if err := client.json(ctx, &created, args...); err != nil {
		return Result{}, fmt.Errorf("create workspace: %w; check Rex for %q before retrying", err, pending)
	}
	result.SessionID = created.SessionID
	if result.SessionID == "" || len(created.Windows) == 0 || len(created.Windows[0].Blocks) == 0 || created.Windows[0].Blocks[0] == "" {
		return result, fmt.Errorf("Rex returned incomplete session creation data; inspect %q before retrying", pending)
	}
	blocks := map[string]string{first.Pane: created.Windows[0].Blocks[0]}
	fail := func(err error) (Result, error) {
		return result, fmt.Errorf("startup incomplete for session %s (%s): %w; existing panes were left running", result.SessionID, pending, err)
	}
	for i, tab := range project.Tabs {
		anchor := blocks[firstPane(tab.Layout).Pane]
		if i > 0 {
			leaf := firstPane(tab.Layout)
			// Set the user-provided label in a separate rename call, where --
			// makes names beginning with a hyphen unambiguously positional.
			args := withCommand([]string{"window", "new", "-s", result.SessionID, "--cwd", leaf.Cwd, "--keep-open", "--focus=false", "--json"}, commands[leaf.Pane])
			var window struct {
				ID     string   `json:"window_id"`
				Blocks []string `json:"block_ids"`
			}
			if err := client.json(ctx, &window, args...); err != nil {
				return fail(err)
			}
			if window.ID == "" || len(window.Blocks) == 0 || window.Blocks[0] == "" {
				return fail(errors.New("Rex returned a tab without a window or pane ID"))
			}
			if err := client.command(ctx, "window", "rename", "-s", result.SessionID, "--", window.ID, tab.Name); err != nil {
				return fail(err)
			}
			anchor = window.Blocks[0]
			blocks[leaf.Pane] = anchor
		}
		if err := client.command(ctx, "block", "rename", "-s", result.SessionID, anchor, firstPane(tab.Layout).Pane); err != nil {
			return fail(err)
		}
		if err := client.build(ctx, result.SessionID, tab.Layout, anchor, blocks, commands); err != nil {
			return fail(err)
		}
		if err := client.command(ctx, "focus", "-s", result.SessionID, blocks[tab.Focus]); err != nil {
			return fail(err)
		}
	}
	if err := client.command(ctx, "focus", "-s", result.SessionID, blocks[project.InitialTab().Focus]); err != nil {
		return fail(err)
	}
	if err := client.command(ctx, "session", "rename", result.SessionID, label); err != nil {
		return fail(err)
	}
	return result, client.selectSession(ctx, result, options)
}

func (client *Client) build(ctx context.Context, session string, node workspace.Layout, anchor string, blocks map[string]string, commands map[string][]string) error {
	if node.Pane != "" {
		return nil
	}
	anchors := make([]string, len(node.Children))
	anchors[0] = anchor
	shares := node.Shares()
	remaining := 1.0
	direction := "right"
	if node.Split == "rows" {
		direction = "below"
	}
	// Split off each remaining region before subdividing any child group.
	for i := 1; i < len(node.Children); i++ {
		next := remaining - shares[i-1]
		ratio := int(math.Round(100 * next / remaining))
		leaf := firstPane(node.Children[i])
		args := withCommand([]string{"split", "-s", session, "-b", anchors[i-1], "--split", direction, "--ratio", strconv.Itoa(ratio), "--cwd", leaf.Cwd, "--label", leaf.Pane, "--keep-open", "--focus=false", "--json"}, commands[leaf.Pane])
		var split struct {
			Blocks []string `json:"block_ids"`
		}
		if err := client.json(ctx, &split, args...); err != nil {
			return err
		}
		if len(split.Blocks) == 0 || split.Blocks[0] == "" {
			return errors.New("Rex returned a split without a pane ID")
		}
		anchors[i] = split.Blocks[0]
		blocks[leaf.Pane] = anchors[i]
		remaining = next
	}
	for i, child := range node.Children {
		if err := client.build(ctx, session, child, anchors[i], blocks, commands); err != nil {
			return err
		}
	}
	return nil
}

func launchCommands(project *workspace.Project) (map[string][]string, error) {
	commands := map[string][]string{}
	var walk func(workspace.Layout) error
	walk = func(node workspace.Layout) error {
		if node.Pane != "" {
			command := node.Command
			if len(node.Commands) > 0 {
				var err error
				command, err = panerunner.Invocation(node.Commands)
				if err != nil {
					return fmt.Errorf("prepare pane %q: %w", node.Pane, err)
				}
			}
			commands[node.Pane] = command
			return nil
		}
		for _, child := range node.Children {
			if err := walk(child); err != nil {
				return err
			}
		}
		return nil
	}
	for _, tab := range project.Tabs {
		if err := walk(tab.Layout); err != nil {
			return nil, err
		}
	}
	return commands, nil
}

func preflight(project *workspace.Project) error {
	if project == nil || len(project.Tabs) == 0 {
		return errors.New("project has no tabs")
	}
	checkDir := func(path string) error {
		info, err := os.Stat(path)
		if err != nil {
			return fmt.Errorf("workspace directory %q: %w", path, err)
		}
		if !info.IsDir() {
			return fmt.Errorf("workspace directory %q is not a directory", path)
		}
		return nil
	}
	if err := checkDir(project.Root); err != nil {
		return err
	}
	var walk func(workspace.Layout) error
	walk = func(node workspace.Layout) error {
		if node.Pane != "" {
			return checkDir(node.Cwd)
		}
		if len(node.Children) < 2 {
			return errors.New("layout split has fewer than two children")
		}
		remaining := 1.0
		shares := node.Shares()
		for i := 1; i < len(shares); i++ {
			next := remaining - shares[i-1]
			ratio := math.Round(100 * next / remaining)
			if ratio < 1 || ratio > 99 {
				return errors.New("layout split cannot be represented by Rex's integer percentage ratios; use fewer children or larger shares")
			}
			remaining = next
		}
		for _, child := range node.Children {
			if err := walk(child); err != nil {
				return err
			}
		}
		return nil
	}
	for _, tab := range project.Tabs {
		if err := walk(tab.Layout); err != nil {
			return err
		}
	}
	return nil
}

func firstPane(node workspace.Layout) workspace.Layout {
	for node.Pane == "" {
		node = node.Children[0]
	}
	return node
}

func withCommand(args, command []string) []string {
	if len(command) != 0 {
		args = append(args, "--")
		args = append(args, command...)
	}
	return args
}

func (client *Client) selectSession(ctx context.Context, result Result, options Options) error {
	if options.Background {
		return nil
	}
	args := []string{}
	if options.Client != "" {
		args = append(args, "-C", options.Client)
	}
	args = append(args, "do", "session.select", "session_id="+result.SessionID)
	if err := client.command(ctx, args...); err != nil {
		return fmt.Errorf("session %s is ready, but could not select it in Rex: %w; enable Remote Control or use --client/--background", result.SessionID, err)
	}
	return nil
}

func (client *Client) command(ctx context.Context, args ...string) error {
	_, err := client.run(ctx, args...)
	return err
}
func (client *Client) json(ctx context.Context, target any, args ...string) error {
	data, err := client.run(ctx, args...)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("decode Rex %s response: %w", args[0], err)
	}
	return nil
}
