package rex

import (
	"context"
	"errors"
	"fmt"
	"math"
	"net/url"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"

	"rexp/internal/workspace"
)

// These fields mirror rex session inspect --json. Keep the binary split tree:
// flattening adjacent groups can change which region a later split subdivides.
type sessionSnapshot struct {
	ID           string           `json:"session_id"`
	ActiveWindow string           `json:"active_window_id"`
	Windows      []windowSnapshot `json:"windows"`
}

type windowSnapshot struct {
	ID     string `json:"window_id"`
	Label  string `json:"label"`
	Focus  string `json:"focused_block_id"`
	Layers []struct {
		Kind   string          `json:"kind"`
		Layout *layoutSnapshot `json:"layout"`
		Blocks []blockSnapshot `json:"blocks"`
	} `json:"layers"`
}

type layoutSnapshot struct {
	Block string `json:"block_id"`
	Split *struct {
		Direction string          `json:"direction"`
		Ratio     *float64        `json:"ratio"`
		Before    *layoutSnapshot `json:"before"`
		After     *layoutSnapshot `json:"after"`
	} `json:"split"`
}

type blockSnapshot struct {
	ID      string `json:"block_id"`
	Label   string `json:"label"`
	Creator string `json:"creator_name"`
	Host    string `json:"host_id"`
}

type ExportResult struct {
	Project  *workspace.Project
	Warnings []string
}

type exportBuilder struct {
	result   ExportResult
	names    map[string]bool
	blocks   map[string]bool
	windows  map[string]bool
	host     string
	count    int
	terminal func(string) (workspace.Layout, error)
}

func (builder *exportBuilder) convert(snapshot sessionSnapshot, name, root string) (ExportResult, error) {
	if snapshot.ID == "" || len(snapshot.Windows) == 0 {
		return ExportResult{}, errors.New("Rex returned a session without an ID or tabs")
	}
	builder.result.Project = &workspace.Project{SchemaVersion: 1, Name: name, Root: root}
	builder.names = map[string]bool{}
	builder.blocks = map[string]bool{}
	builder.windows = map[string]bool{}
	tabNames := map[string]bool{}
	for i, window := range snapshot.Windows {
		if window.ID == "" || builder.windows[window.ID] {
			return ExportResult{}, errors.New("Rex returned missing or duplicate tab IDs")
		}
		builder.windows[window.ID] = true
		if len(window.Layers) != 1 || window.Layers[0].Kind != "tiled" {
			return ExportResult{}, fmt.Errorf("tab %q has unsupported layers; save currently supports one tiled terminal layer per tab", window.Label)
		}
		layer := window.Layers[0]
		metadata := map[string]blockSnapshot{}
		for _, block := range layer.Blocks {
			if block.ID == "" {
				return ExportResult{}, errors.New("Rex returned a pane without an ID")
			}
			if _, exists := metadata[block.ID]; exists {
				return ExportResult{}, fmt.Errorf("duplicate pane ID %q", block.ID)
			}
			metadata[block.ID] = block
		}
		focus := ""
		layout, err := builder.layout(layer.Layout, metadata, window.Focus, &focus, 0)
		if err != nil {
			return ExportResult{}, fmt.Errorf("tab %q: %w", window.Label, err)
		}
		if len(metadata) != 0 {
			return ExportResult{}, fmt.Errorf("tab %q has panes outside its tiled layout", window.Label)
		}
		if window.Focus != "" && focus == "" {
			return ExportResult{}, fmt.Errorf("tab %q focuses a pane outside its layout", window.Label)
		}
		tabName := window.Label
		if strings.TrimSpace(tabName) == "" {
			tabName = fmt.Sprintf("Tab %d", i+1)
		}
		base := tabName
		for suffix := 2; tabNames[tabName]; suffix++ {
			tabName = fmt.Sprintf("%s (%d)", base, suffix)
		}
		tabNames[tabName] = true
		if tabName != window.Label {
			builder.result.Warnings = append(builder.result.Warnings, fmt.Sprintf("Tab %q was saved as %q to give it a unique nonempty name.", window.Label, tabName))
		}
		builder.result.Project.Tabs = append(builder.result.Project.Tabs, workspace.Tab{Name: tabName, Focus: focus, Layout: layout})
	}
	if snapshot.ActiveWindow != "" {
		if !builder.windows[snapshot.ActiveWindow] {
			return ExportResult{}, errors.New("Rex's active tab is missing from the snapshot; retry saving after the session settles")
		}
		if snapshot.ActiveWindow != snapshot.Windows[0].ID {
			builder.result.Warnings = append(builder.result.Warnings, "Tab order was preserved; start initially selects the first tab, which differs from the currently active tab.")
		}
	}
	return builder.result, nil
}

func (builder *exportBuilder) layout(node *layoutSnapshot, metadata map[string]blockSnapshot, focused string, focus *string, depth int) (workspace.Layout, error) {
	if node == nil || depth > 32 {
		return workspace.Layout{}, errors.New("missing layout node or layout nesting exceeds 32 levels")
	}
	if (node.Block == "") == (node.Split == nil) {
		return workspace.Layout{}, errors.New("layout node must contain exactly one pane or split")
	}
	if node.Split != nil {
		split := node.Split
		direction := ""
		switch split.Direction {
		case "horizontal":
			direction = "columns"
		case "vertical":
			direction = "rows"
		default:
			return workspace.Layout{}, fmt.Errorf("unsupported split direction %q", split.Direction)
		}
		if split.Ratio == nil || math.IsNaN(*split.Ratio) || math.IsInf(*split.Ratio, 0) || *split.Ratio <= 0 || *split.Ratio >= 1 {
			return workspace.Layout{}, errors.New("Rex returned an invalid split ratio")
		}
		percentage := int(math.Round(*split.Ratio * 100))
		if percentage < 1 || percentage > 99 {
			return workspace.Layout{}, errors.New("split is too narrow to save as integer percentages between 1 and 99")
		}
		before, err := builder.layout(split.Before, metadata, focused, focus, depth+1)
		if err != nil {
			return workspace.Layout{}, err
		}
		after, err := builder.layout(split.After, metadata, focused, focus, depth+1)
		if err != nil {
			return workspace.Layout{}, err
		}
		return workspace.Layout{Split: direction, Sizes: []int{percentage, 100 - percentage}, Children: []workspace.Layout{before, after}}, nil
	}
	block, exists := metadata[node.Block]
	if !exists || builder.blocks[node.Block] {
		return workspace.Layout{}, fmt.Errorf("pane %q is missing or occurs more than once in the layout", node.Block)
	}
	if block.Creator != "com.superlogical.terminal" {
		return workspace.Layout{}, fmt.Errorf("pane %q has unsupported creator %q; only terminal panes can be saved", block.Label, block.Creator)
	}
	if builder.host != "" && block.Host != "" && builder.host != block.Host {
		return workspace.Layout{}, errors.New("panes on different hosts cannot be represented by a local project configuration")
	}
	if block.Host != "" {
		builder.host = block.Host
	}
	builder.blocks[node.Block] = true
	delete(metadata, node.Block)
	builder.count++
	name := block.Label
	if workspace.ValidateName(name) != nil {
		name = fmt.Sprintf("pane-%d", builder.count)
	}
	base := name
	for suffix := 2; builder.names[name]; suffix++ {
		ending := "-" + strconv.Itoa(suffix)
		name = base
		if len(name)+len(ending) > 64 {
			name = name[:64-len(ending)]
		}
		name += ending
	}
	builder.names[name] = true
	if name != block.Label {
		builder.result.Warnings = append(builder.result.Warnings, fmt.Sprintf("Pane %q was saved as %q to give it a valid unique name.", block.Label, name))
	}
	pane, err := builder.terminal(node.Block)
	if err != nil {
		return workspace.Layout{}, fmt.Errorf("read pane %q: %w", block.Label, err)
	}
	pane.Pane = name
	if node.Block == focused {
		*focus = name
	}
	return pane, nil
}

// Export reads a particular session; it never follows the app's current focus.
// Terminal metadata is queried separately because session inspect only includes
// block identities and layout geometry.
func (client *Client) Export(ctx context.Context, session, name, root string) (ExportResult, error) {
	var snapshot sessionSnapshot
	if err := client.json(ctx, &snapshot, "session", "inspect", "--json", "--", session); err != nil {
		return ExportResult{}, err
	}
	if strings.HasPrefix(session, "session:") && snapshot.ID != session {
		return ExportResult{}, errors.New("Rex returned a different session than requested")
	}
	var directoryWarnings []string
	result, err := (&exportBuilder{terminal: func(block string) (workspace.Layout, error) {
		cwd, reported, err := client.exportDirectory(ctx, snapshot.ID, block)
		if reported {
			directoryWarnings = append(directoryWarnings, fmt.Sprintf("Pane %q has no readable process directory; used its last reported directory.", block))
		}
		return workspace.Layout{Cwd: cwd, Command: []string{}}, err
	}}).convert(snapshot, name, root)
	if err != nil {
		return ExportResult{}, err
	}
	var current sessionSnapshot
	if err := client.json(ctx, &current, "session", "inspect", "--json", "--", snapshot.ID); err != nil {
		return ExportResult{}, err
	}
	if !reflect.DeepEqual(snapshot, current) {
		return ExportResult{}, errors.New("session layout, labels, or focus changed while saving; retry after the session settles")
	}
	result.Warnings = append(result.Warnings, directoryWarnings...)
	result.Warnings = append(result.Warnings, "Rex does not expose complete command arguments. Saved panes open shells; add commands manually to recreate your programs.", "Split sizes were rounded to integer percentages; the restored layout may differ slightly.")
	return result, nil
}

type processSnapshot struct {
	Cwd string `json:"cwd"`
}

func (client *Client) exportDirectory(ctx context.Context, session, block string) (string, bool, error) {
	var processes struct {
		Foreground *processSnapshot  `json:"foreground"`
		Ancestors  []processSnapshot `json:"ancestors"`
		Child      *processSnapshot  `json:"child"`
	}
	if err := client.json(ctx, &processes, "block", "call", "-s", session, "-b", block, "process"); err != nil {
		return "", false, err
	}
	candidates := []*processSnapshot{processes.Foreground}
	for i := range processes.Ancestors {
		candidates = append(candidates, &processes.Ancestors[i])
	}
	candidates = append(candidates, processes.Child)
	for _, process := range candidates {
		if process != nil && process.Cwd != "" {
			if !filepath.IsAbs(process.Cwd) || strings.ContainsRune(process.Cwd, 0) {
				return "", false, errors.New("Rex returned a process directory that is not an absolute native path")
			}
			return filepath.Clean(process.Cwd), false, nil
		}
	}
	var reported struct {
		Pwd string `json:"pwd"`
	}
	if err := client.json(ctx, &reported, "block", "call", "-s", session, "-b", block, "pwd"); err != nil {
		return "", false, err
	}
	cwd := reported.Pwd
	if strings.HasPrefix(cwd, "file:") {
		u, err := url.Parse(cwd)
		if err != nil || u.Scheme != "file" || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
			return "", false, errors.New("Rex returned an invalid working-directory URL")
		}
		cwd = filepath.FromSlash(u.Path)
	}
	if !filepath.IsAbs(cwd) || strings.ContainsRune(cwd, 0) {
		return "", false, errors.New("no absolute pane directory available; cannot save this pane accurately")
	}
	return filepath.Clean(cwd), true, nil
}
