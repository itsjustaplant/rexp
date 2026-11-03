// Package demo provides the self-contained panes used by rexp init.
package demo

import (
	"context"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/itsjustaplant/rexp/internal/workspace"
)

const Mode = "__demo"

// Logo is a small ASCII rendering of assets/rexp.png.
const Logo = `
                 ###
              #  #####
             ############
              ###############
           ##################
           ###################
             ################
           #################
            ########                 rexp
         #############
          #############         Your workspace, ready.
 #     #############
  ###################
  ###################
    ################
          ###  ####
         #####  #####

  Named project workspaces for Rex.

  rexp new my-app       Create a workspace
  rexp use my-app       Start and switch
  rexp switch my-app    Switch to a running session
`

// Project builds a demo without touching saved project configurations.
func Project(binary, root string) *workspace.Project {
	return &workspace.Project{
		SchemaVersion: 1, Name: "rexp-intro", Root: root,
		Tabs: []workspace.Tab{{Name: "Welcome to rexp", Focus: "clock", Layout: workspace.Layout{
			Split: "columns", Sizes: []int{50, 50}, Children: []workspace.Layout{
				{Pane: "clock", Cwd: root, Command: []string{binary, Mode, "clock"}},
				{Pane: "rexp", Cwd: root, Command: []string{binary, Mode, "logo"}},
			},
		}}},
	}
}

var digits = [...][]string{
	{" ### ", "#   #", "#   #", "#   #", " ### "},
	{"  #  ", " ##  ", "  #  ", "  #  ", " ### "},
	{" ### ", "    #", " ### ", "#    ", "#####"},
	{"#### ", "    #", " ### ", "    #", "#### "},
	{"#   #", "#   #", "#####", "    #", "    #"},
	{"#####", "#    ", "#### ", "    #", "#### "},
	{" ### ", "#    ", "#### ", "#   #", " ### "},
	{"#####", "    #", "   # ", "  #  ", " #   "},
	{" ### ", "#   #", " ### ", "#   #", " ### "},
	{" ### ", "#   #", " ####", "    #", " ### "},
}

func clockFrame(now time.Time) string {
	var out strings.Builder
	out.WriteString("\n  A little time to make something.\n\n")
	for row := range 5 {
		out.WriteString("  ")
		for _, char := range now.Format("15:04:05") {
			if char == ':' {
				if row == 1 || row == 3 {
					out.WriteString(" # ")
				} else {
					out.WriteString("   ")
				}
			} else {
				out.WriteString(digits[char-'0'][row])
			}
			out.WriteByte(' ')
		}
		out.WriteByte('\n')
	}
	fmt.Fprintf(&out, "\n  %s\n\n  Local time  /  Ctrl+C to stop the clock.\n", now.Format("Monday, 02 January 2006 MST"))
	return out.String()
}

// Run is dispatched before CLI initialization, so panes do not prompt or emit
// lifecycle statuses. The clock restores the cursor when interrupted.
func Run(ctx context.Context, args []string, out io.Writer) error {
	if len(args) != 1 {
		return fmt.Errorf("demo requires clock or logo")
	}
	switch args[0] {
	case "logo":
		_, err := io.WriteString(out, Logo)
		return err
	case "clock":
		if _, err := io.WriteString(out, "\x1b[2J\x1b[H\x1b[?25l"); err != nil {
			return err
		}
		defer io.WriteString(out, "\x1b[?25h\n")
		ticker := time.NewTicker(time.Second)
		defer ticker.Stop()
		for {
			if _, err := fmt.Fprint(out, "\x1b[H", clockFrame(time.Now()), "\x1b[J"); err != nil {
				return err
			}
			select {
			case <-ctx.Done():
				return nil
			case <-ticker.C:
			}
		}
	default:
		return fmt.Errorf("unknown demo pane %q", args[0])
	}
}
