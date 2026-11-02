package cli

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"text/tabwriter"
	"unicode"

	"github.com/itsjustaplant/rexp/internal/config"
	"github.com/spf13/cobra"
)

type projectOutput struct{ long, json bool }

func (output *projectOutput) bind(cmd *cobra.Command) {
	cmd.Flags().BoolVarP(&output.long, "long", "l", false, "Show configuration paths, roots, tab/pane counts, and validity")
	cmd.Flags().BoolVar(&output.json, "json", false, "Output configuration metadata as a JSON array")
	cmd.MarkFlagsMutuallyExclusive("long", "json")
}

func (output projectOutput) write(cmd *cobra.Command, names []string, empty string) error {
	out := cmd.OutOrStdout()
	if output.json {
		return json.NewEncoder(out).Encode(config.SummarizeProjects(names))
	}
	if len(names) == 0 {
		_, err := fmt.Fprintln(out, empty)
		return err
	}
	if !output.long {
		for _, name := range names {
			if _, err := fmt.Fprintln(out, name); err != nil {
				return err
			}
		}
		return nil
	}
	table := tabwriter.NewWriter(out, 0, 4, 2, ' ', 0)
	if _, err := fmt.Fprintln(table, "NAME\tROOT\tTABS\tPANES\tSTATUS\tPATH\tERROR"); err != nil {
		return err
	}
	for _, summary := range config.SummarizeProjects(names) {
		status, tabs, panes, root := "invalid", "-", "-", "-"
		if summary.Valid {
			status = "valid"
			tabs = strconv.Itoa(summary.Tabs)
			panes = strconv.Itoa(summary.Panes)
			root = summary.Root
		}
		if _, err := fmt.Fprintf(table, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n", summary.Name, tableText(root), tabs, panes, status, tableText(summary.Path), tableText(summary.Error)); err != nil {
			return err
		}
	}
	return table.Flush()
}

// Paths/YAML errors can contain control characters; keep one entry per row
// and prevent terminal escape sequences in user-controlled configuration data.
func tableText(value string) string {
	if strings.ContainsFunc(value, func(char rune) bool { return !unicode.IsPrint(char) }) {
		return strconv.QuoteToGraphic(value)
	}
	return value
}
