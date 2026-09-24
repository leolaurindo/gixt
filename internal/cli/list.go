package cli

import (
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"

	"golang.org/x/term"

	"github.com/spf13/cobra"

	"github.com/leolaurindo/gixt/internal/gist"
	"github.com/leolaurindo/gixt/internal/known"
)

func newListCmd() *cobra.Command {
	c := &cobra.Command{
		Use:   "list [owner]",
		Short: "list remembered gists or an owner's gists",
		Args:  cobra.MaximumNArgs(1),
		RunE:  listGists,
	}
	c.Flags().Int("limit", 30, "maximum number of remote gists to list")
	c.Flags().Bool("all", false, "list all remote gists")
	c.Flags().Bool("verbose", false, "show gist descriptions")
	return c
}

func listGists(cmd *cobra.Command, args []string) error {
	all := mustBool(cmd, "all")
	verbose := mustBool(cmd, "verbose")
	limit, _ := cmd.Flags().GetInt("limit")
	if len(args) == 0 {
		if cmd.Flags().Changed("limit") || all {
			return fmt.Errorf("--limit and --all require an owner")
		}
		paths, err := ensurePaths()
		if err != nil {
			return err
		}
		st, err := known.Load(paths.KnownFile)
		if err != nil {
			return err
		}
		return writeList(st.Sorted(), verbose)
	}
	if cmd.Flags().Changed("limit") && all {
		return fmt.Errorf("--limit and --all are mutually exclusive")
	}
	if !all && limit <= 0 {
		return fmt.Errorf("--limit must be positive")
	}

	paths, err := ensurePaths()
	if err != nil {
		return err
	}
	client := gist.New(loadToken(paths.AuthFile))
	items, err := listOwner(cmd, client, args[0], limit, all)
	if err != nil {
		return err
	}
	entries := make([]known.Entry, 0, len(items))
	for _, item := range items {
		entries = append(entries, toKnownEntryFromList(item))
	}
	return writeList(known.Store{Entries: entries}.Sorted(), verbose)
}

func listOwner(cmd *cobra.Command, client *gist.Client, owner string, limit int, all bool) ([]gist.ListItem, error) {
	if all {
		return client.ListForOwner(cmd.Context(), owner, 100, 0)
	}
	perPage, maxPages := limit, 1
	if perPage > 100 {
		perPage = 100
		maxPages = (limit + perPage - 1) / perPage
	}
	items, err := client.ListForOwner(cmd.Context(), owner, perPage, maxPages)
	if err != nil {
		return nil, err
	}
	if len(items) > limit {
		items = items[:limit]
	}
	return items, nil
}

func writeList(entries []known.Entry, verbose bool) error {
	isTTY := term.IsTerminal(int(os.Stdout.Fd()))
	color := isTTY && os.Getenv("NO_COLOR") == ""
	tableWidth := 100
	if isTTY {
		if width, _, err := term.GetSize(int(os.Stdout.Fd())); err == nil && width > 0 {
			tableWidth = width
		}
	}
	return renderList(os.Stdout, entries, color, verbose, tableWidth)
}

func renderList(w io.Writer, entries []known.Entry, color, verbose bool, tableWidth int) error {
	headers := []string{"NAME", "OWNER", "ID"}
	if verbose {
		headers = append(headers, "DESCRIPTION")
	}
	rows := make([][]string, len(entries))
	widths := make([]int, len(headers))
	for i, header := range headers {
		widths[i] = utf8.RuneCountInString(header)
	}
	descriptions := make([][]string, len(entries))
	for i, e := range entries {
		name := e.Alias
		if name == "" && len(e.Filenames) > 0 {
			name = e.Filenames[0]
		}
		if name == "" {
			name = "(unnamed)"
		}
		owner := e.Owner
		if owner != "" {
			owner = "@" + owner
		}
		rows[i] = []string{name, owner, e.ID}
		for col, value := range rows[i] {
			if width := utf8.RuneCountInString(value); width > widths[col] {
				widths[col] = width
			}
		}
	}
	if verbose {
		descriptionWidth := tableWidth - widths[0] - widths[1] - widths[2] - 13
		if descriptionWidth < widths[3] {
			descriptionWidth = widths[3]
		}
		for i, e := range entries {
			description := strings.Join(strings.Fields(e.Description), " ")
			descriptions[i] = wrapDescription(description, descriptionWidth)
			for _, line := range descriptions[i] {
				if width := utf8.RuneCountInString(line); width > widths[3] {
					widths[3] = width
				}
			}
		}
	}

	drawBorder := func(left, join, right string) error {
		if _, err := fmt.Fprint(w, left); err != nil {
			return err
		}
		for i, width := range widths {
			if i > 0 {
				if _, err := fmt.Fprint(w, join); err != nil {
					return err
				}
			}
			if _, err := fmt.Fprint(w, strings.Repeat("─", width+2)); err != nil {
				return err
			}
		}
		_, err := fmt.Fprintln(w, right)
		return err
	}
	drawRow := func(values []string, colorName bool) error {
		if _, err := fmt.Fprint(w, "│"); err != nil {
			return err
		}
		for i, value := range values {
			if _, err := fmt.Fprint(w, " "); err != nil {
				return err
			}
			if color && colorName && i == 0 {
				value = "\033[36m" + value + "\033[0m"
			}
			if _, err := fmt.Fprint(w, value, strings.Repeat(" ", widths[i]-utf8.RuneCountInString(values[i])), " │"); err != nil {
				return err
			}
		}
		_, err := fmt.Fprintln(w)
		return err
	}

	if err := drawBorder("┌", "┬", "┐"); err != nil {
		return err
	}
	if err := drawRow(headers, false); err != nil {
		return err
	}
	if err := drawBorder("├", "┼", "┤"); err != nil {
		return err
	}
	for i, row := range rows {
		if !verbose {
			if err := drawRow(row, true); err != nil {
				return err
			}
			continue
		}
		for lineIndex, description := range descriptions[i] {
			values := []string{"", "", "", description}
			if lineIndex == 0 {
				values[0], values[1], values[2] = row[0], row[1], row[2]
			}
			if err := drawRow(values, lineIndex == 0); err != nil {
				return err
			}
		}
	}
	return drawBorder("└", "┴", "┘")
}

func wrapDescription(text string, width int) []string {
	if width < 1 {
		width = 1
	}
	words := strings.Fields(text)
	if len(words) == 0 {
		return []string{""}
	}
	var lines []string
	var line strings.Builder
	lineWidth := 0
	flush := func() {
		lines = append(lines, line.String())
		line.Reset()
		lineWidth = 0
	}
	for _, word := range words {
		runes := []rune(word)
		if lineWidth > 0 && lineWidth+1+len(runes) <= width {
			line.WriteByte(' ')
			line.WriteString(word)
			lineWidth += 1 + len(runes)
			continue
		}
		if lineWidth > 0 {
			flush()
		}
		for len(runes) > width {
			line.WriteString(string(runes[:width]))
			flush()
			runes = runes[width:]
		}
		line.WriteString(string(runes))
		lineWidth = len(runes)
	}
	if lineWidth > 0 || len(lines) == 0 {
		flush()
	}
	return lines
}
