package utils

import (
	"strconv"

	"github.com/charmbracelet/x/ansi"
	"slices"
	"sort"
	"strings"

	viewport "github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	lipgloss "github.com/charmbracelet/lipgloss"
)

// escapeCell replaces control characters that would break the table layout.
func escapeCell(s string) string {
	r := strings.NewReplacer(
		"\n", "\\n",
		"\r", "\\r",
		"\t", "\\t",
	)
	return r.Replace(s)
}

var MIN_COLUMN_WIDTH = 100

type Table struct {
	Columns             []string
	Rows                [][]string
	SelectedRow         int
	SelectedColumn      int
	ColumnsStyle        lipgloss.Style
	SelectedRowStyle    lipgloss.Style
	SelectedColumnStyle lipgloss.Style
	SelectedCellStyle   lipgloss.Style
	// MarkedStyle colors rows/columns marked with "a" - deliberately a
	// different color from SelectedRowStyle/SelectedColumnStyle/
	// SelectedCellStyle (the cursor/hover highlight) so "this is where my
	// cursor currently is" and "this is what I've marked for the row
	// view" are visually distinguishable at a glance instead of looking
	// identical.
	MarkedStyle lipgloss.Style
	Viewport    viewport.Model

	// MarkedRows tracks rows marked with "a" for the multi-select vertical
	// row view ("r") - if empty when "r" is pressed, just the currently
	// hovered row is shown instead.
	MarkedRows map[int]bool
	// MarkedColumns tracks columns marked with "a" (alongside the row
	// under the cursor - "a" marks both at once) - if empty when "r" is
	// pressed, every column is shown instead of just some of them. Lets
	// you narrow the vertical row view down to only the fields you
	// actually care about instead of every column in the row.
	MarkedColumns map[int]bool
	// RowView, when true, renders the marked (or just hovered) row(s)
	// vertically as field: value pairs instead of the normal grid - like
	// psql's \x / expanded display, for rows too wide/many-columned to
	// read comfortably side-by-side.
	RowView bool

	columnWidths []int
}

func InitTable(data [][]string, width int, height int) Table {
	viewport := viewport.New(width-2, height-2)
	var rows [][]string
	var cols []string

	if len(data) > 1 {
		rows = data[1:]
		cols = data[0]
	} else {
		rows = [][]string{}
		cols = []string{}
	}

	table := Table{
		Columns:             cols,
		Rows:                rows,
		SelectedRow:         0,
		SelectedColumn:      0,
		ColumnsStyle:        lipgloss.NewStyle().Bold(true),
		SelectedRowStyle:    lipgloss.NewStyle().Background(lipgloss.Color("57")).Foreground(lipgloss.Color("229")),
		SelectedColumnStyle: lipgloss.NewStyle().Background(lipgloss.Color("60")).Foreground(lipgloss.Color("229")),
		SelectedCellStyle:   lipgloss.NewStyle().Background(lipgloss.Color("63")).Foreground(lipgloss.Color("229")),
		// A different shade of the same purple/blue family as the hover
		// highlights above (57/60/63), not an unrelated color like orange -
		// still clearly a different shade so marked vs hovered don't look
		// identical, without clashing with the rest of the app's palette.
		MarkedStyle: lipgloss.NewStyle().Background(lipgloss.Color("97")).Foreground(lipgloss.Color("255")),
		Viewport:    viewport,
		MarkedRows:          map[int]bool{},
		MarkedColumns:       map[int]bool{},
		columnWidths:        calculateColumnWidths(cols, rows),
	}
	content := table.renderColumns() + "\n" + table.renderRows()
	table.Viewport.SetContent(content)
	return table
}

func (t Table) Update(msg tea.Msg) (Table, tea.Cmd) {
	var viewportCmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "l", "right":
			if t.SelectedColumn < len(t.Columns)-1 {
				t.Viewport.ScrollRight(t.selectedColumnWidth())
				t.SelectedColumn++
			}
		case "h", "left":
			if t.SelectedColumn > 0 {
				t.SelectedColumn--
				t.Viewport.ScrollLeft(t.selectedColumnWidth())
			}
		case "j", "down":
			if t.SelectedRow < len(t.Rows)-1 {
				t.SelectedRow++
			}
		case "k", "up":
			if t.SelectedRow > 0 {
				t.SelectedRow--
			}
		case "a":
			// Mark/unmark the hovered row AND column, for the multi-select
			// vertical view ("r") - if no rows are marked when you press
			// r, it just shows whichever one is currently hovered
			// instead; same idea for columns, showing every column
			// instead of just some of them.
			if len(t.Rows) > 0 {
				if t.MarkedRows[t.SelectedRow] {
					delete(t.MarkedRows, t.SelectedRow)
				} else {
					t.MarkedRows[t.SelectedRow] = true
				}
			}
			if len(t.Columns) > 0 {
				if t.MarkedColumns[t.SelectedColumn] {
					delete(t.MarkedColumns, t.SelectedColumn)
				} else {
					t.MarkedColumns[t.SelectedColumn] = true
				}
			}
		case "r":
			t.RowView = !t.RowView
		case "A":
			t.sortByColumn(true)
		case "D":
			t.sortByColumn(false)
		}
	case LayoutUpdated:
		layout := ConnectionContainerLayout(msg)
		t.Viewport.Width = (layout.ViewerWidth - 4)
		t.Viewport.Height = (layout.ViewerHeight - 4)
	}
	content := t.renderContent()
	t.Viewport.SetContent(content)
	t.Viewport, viewportCmd = t.Viewport.Update(msg)
	return t, viewportCmd
}

func (t Table) renderContent() string {
	if t.RowView {
		return t.renderRowView()
	}
	return t.renderColumns() + "\n" + t.renderRows()
}

func (t Table) View() string {
	return t.Viewport.View()
}

func (t Table) HasData() bool {
	return len(t.Columns) > 0 && len(t.Rows) > 0
}

func calculateColumnWidths(cols []string, rows [][]string) []int {
	widths := make([]int, len(cols))
	for i, col := range cols {
		widths[i] = slices.Max([]int{widths[i], len(col) + 2})
	}
	for _, row := range rows {
		for j, cell := range row {
			widths[j] = slices.Min([]int{
				MIN_COLUMN_WIDTH,
				slices.Max([]int{widths[j], len(cell) + 2}),
			})
		}
	}
	return widths
}

func (t Table) renderColumns() string {
	var columns []string
	for i, col := range t.Columns {
		style := t.ColumnsStyle.
			Width(t.columnWidths[i]).
			Padding(0, 1, 0, 1)

		if t.MarkedColumns[i] {
			style = style.Inherit(t.MarkedStyle)
		}
		if i == t.SelectedColumn {
			// Hover takes precedence over marked - you should always be
			// able to see where your cursor actually is, even on a
			// column you've also marked.
			style = style.Inherit(t.SelectedColumnStyle)
		}
		columns = append(columns, style.Render(col))
	}
	return lipgloss.JoinHorizontal(lipgloss.Left, columns...)
}

func (t Table) renderRows() string {
	var rows []string
	for i, row := range t.Rows {
		var columns []string
		for j, cell := range row {
			style := lipgloss.NewStyle().Width(t.columnWidths[j]).Padding(0, 1, 0, 1)
			marked := t.MarkedRows[i] || t.MarkedColumns[j]
			hovered := i == t.SelectedRow || j == t.SelectedColumn
			switch {
			case i == t.SelectedRow && j == t.SelectedColumn:
				style = style.Inherit(t.SelectedCellStyle)
			case marked && !hovered:
				style = style.Inherit(t.MarkedStyle)
			case i == t.SelectedRow:
				style = style.Inherit(t.SelectedRowStyle)
			case j == t.SelectedColumn:
				style = style.Inherit(t.SelectedColumnStyle)
			}
			columns = append(columns, style.Render(ansi.Truncate(escapeCell(cell), t.columnWidths[j]-2, "…")))
		}
		rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Left, columns...))
	}
	return lipgloss.JoinVertical(lipgloss.Left, rows...)
}

func (t Table) selectedColumnWidth() int {
	if len(t.columnWidths) == 0 {
		return 0
	}
	return t.columnWidths[t.SelectedColumn]
}

// sortByColumn sorts t.Rows in place by whatever column is currently
// highlighted (t.SelectedColumn) - numeric-aware (compares as numbers when
// both sides parse as one, e.g. so "10" sorts after "9" instead of before
// it lexically), falling back to a plain string comparison otherwise.
// Clears MarkedRows since row indices shift after sorting - the marks
// would otherwise silently point at the wrong rows.
func (t *Table) sortByColumn(ascending bool) {
	col := t.SelectedColumn
	if col < 0 || col >= len(t.Columns) || len(t.Rows) == 0 {
		return
	}
	sort.SliceStable(t.Rows, func(i, j int) bool {
		a, b := t.Rows[i][col], t.Rows[j][col]
		if less, ok := lessValue(a, b); ok {
			if ascending {
				return less
			}
			return !less
		}
		return false
	})
	t.MarkedRows = map[int]bool{}
	t.SelectedRow = 0
}

// lessValue compares two cell values, numerically if both parse as
// numbers, otherwise as plain strings. ok is false only when a==b (no
// ordering to report either way).
func lessValue(a, b string) (less bool, ok bool) {
	if a == b {
		return false, false
	}
	af, aErr := strconv.ParseFloat(strings.TrimSpace(a), 64)
	bf, bErr := strconv.ParseFloat(strings.TrimSpace(b), 64)
	if aErr == nil && bErr == nil {
		return af < bf, true
	}
	return a < b, true
}

// rowViewRows returns which row indices renderRowView should show: every
// marked row (in original order) if any are marked, otherwise just
// whichever one is currently hovered.
func (t Table) rowViewRows() []int {
	if len(t.MarkedRows) == 0 {
		if len(t.Rows) == 0 {
			return nil
		}
		return []int{t.SelectedRow}
	}
	rows := make([]int, 0, len(t.MarkedRows))
	for i := range t.MarkedRows {
		rows = append(rows, i)
	}
	sort.Ints(rows)
	return rows
}

// rowViewColumns returns which column indices renderRowView should show:
// every marked column (in original order) if any are marked, otherwise
// every column - same "marked, else fall back to everything" rule as
// rowViewRows.
func (t Table) rowViewColumns() []int {
	if len(t.MarkedColumns) == 0 {
		cols := make([]int, len(t.Columns))
		for i := range t.Columns {
			cols[i] = i
		}
		return cols
	}
	cols := make([]int, 0, len(t.MarkedColumns))
	for i := range t.MarkedColumns {
		cols = append(cols, i)
	}
	sort.Ints(cols)
	return cols
}

// renderRowView renders the selected/marked row(s) vertically as
// "column: value" pairs (like psql's \x expanded display) instead of the
// normal side-by-side grid - useful once a row has more columns than
// comfortably fit on screen, or payload values too wide to read truncated.
func (t Table) renderRowView() string {
	rowIndices := t.rowViewRows()
	if len(rowIndices) == 0 {
		return ""
	}
	colIndices := t.rowViewColumns()
	labelStyle := t.ColumnsStyle
	var blocks []string
	for _, rowIdx := range rowIndices {
		row := t.Rows[rowIdx]
		var lines []string
		for _, colIdx := range colIndices {
			col := t.Columns[colIdx]
			value := ""
			if colIdx < len(row) {
				value = row[colIdx]
			}
			lines = append(lines, lipgloss.JoinHorizontal(lipgloss.Top, labelStyle.Render(col+": "), value))
		}
		blocks = append(blocks, lipgloss.JoinVertical(lipgloss.Left, lines...))
	}
	separator := "\n" + strings.Repeat("-", 40) + "\n"
	return strings.Join(blocks, separator)
}
