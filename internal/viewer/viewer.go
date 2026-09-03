package viewer

import (
	"fmt"
	"strings"

	adapters "app.lazygit/internal/adapters"
	utils "app.lazygit/internal/utils"
	textinput "github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	lipgloss "github.com/charmbracelet/lipgloss"
)

type ViewerModel struct {
	database    adapters.Database
	selectedRow int
	table       utils.Table
	layout      utils.ConnectionContainerLayout
	isActive    bool
	content     string

	// currentTable is whatever table the displayed rows came from (seeded
	// from picking a table in the Databases tab's drill-down) - needed to
	// build a real UPDATE statement when editing a cell (see editingCell
	// below). If empty (e.g. the data came from a hand-written query
	// touching multiple tables/joins), edits only update what's shown
	// locally instead of writing to the database, since there's no single
	// table to safely target.
	currentTable string
	// editingCell/editInput back the "e" edit-cell modal - a small
	// popup pre-filled with the hovered cell's current value.
	editingCell bool
	editInput   textinput.Model
}

func InitViewer(database adapters.Database, layout utils.ConnectionContainerLayout, currentTable string) ViewerModel {
	return ViewerModel{
		database:     database,
		layout:       layout,
		isActive:     false,
		table:        utils.InitTable([][]string{}, layout.ViewerWidth, layout.ViewerHeight),
		content:      "",
		currentTable: currentTable,
	}
}

func createTableFromData(data [][]string, layout utils.ConnectionContainerLayout) utils.Table {
	return utils.InitTable(data, layout.ViewerWidth-2, layout.ViewerHeight-2)
}

func (m ViewerModel) Init() tea.Cmd { return nil }

// IsEditingCell reports whether the "e" edit-cell modal is currently open
// (typing a replacement value) - global shortcuts (esc/q to quit, 1/2/3 to
// jump panes) must not be intercepted while true, or canceling the modal
// with esc would quit the whole app instead, and typing a literal "q" into
// the replacement value would quit instead of being typed.
func (m ViewerModel) IsEditingCell() bool { return m.editingCell }

// sqlQuote escapes single quotes for a naive inline SQL literal - good
// enough for the simple "replace this one cell" UPDATE this builds, not a
// substitute for real parameterized queries if this ever needs to handle
// more than a single scalar replacement.
func sqlQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

// buildUpdateQuery constructs an UPDATE for replacing a single cell.
// Assumes the FIRST column of the row is a usable identifier (typically
// "id") to match on - we don't have real schema/primary-key introspection,
// and guessing a DIFFERENT column would risk updating the wrong row(s)
// silently, so this is the one deliberate assumption made explicit here.
func buildUpdateQuery(table string, columns []string, row []string, colIdx int, newValue string) (string, bool) {
	if table == "" || len(columns) == 0 || len(row) == 0 || colIdx < 0 || colIdx >= len(columns) {
		return "", false
	}
	pkCol := columns[0]
	pkVal := row[0]
	targetCol := columns[colIdx]
	return fmt.Sprintf("UPDATE %s SET %s = %s WHERE %s = %s;", table, targetCol, sqlQuote(newValue), pkCol, sqlQuote(pkVal)), true
}

// cellUpdatedMsg carries the outcome of running the UPDATE (or, when there
// was no currentTable to target, is skipped entirely and the local edit
// just applies immediately with no message needed).
type cellUpdatedMsg struct {
	row, col int
	newValue string
	err      error
}

func (m ViewerModel) runCellUpdate(query string, row, col int, newValue string) tea.Cmd {
	return func() tea.Msg {
		_, err := m.database.RunQuery(query)
		return cellUpdatedMsg{row: row, col: col, newValue: newValue, err: err}
	}
}

func (m ViewerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var viewPortCmd tea.Cmd

	if m.editingCell {
		if keyMsg, ok := msg.(tea.KeyMsg); ok {
			switch keyMsg.String() {
			case "esc":
				m.editingCell = false
				return m, nil
			case "enter":
				m.editingCell = false
				newValue := m.editInput.Value()
				row, col := m.table.SelectedRow, m.table.SelectedColumn
				if row < 0 || row >= len(m.table.Rows) {
					return m, nil
				}
				if m.currentTable == "" {
					// No known source table (e.g. a hand-written multi-
					// table query) - update what's shown locally only,
					// nothing to safely write back to.
					m.table.Rows[row][col] = newValue
					return m, nil
				}
				query, ok := buildUpdateQuery(m.currentTable, m.table.Columns, m.table.Rows[row], col, newValue)
				if !ok {
					return m, nil
				}
				return m, m.runCellUpdate(query, row, col, newValue)
			}
			var cmd tea.Cmd
			m.editInput, cmd = m.editInput.Update(keyMsg)
			return m, cmd
		}
		return m, nil
	}

	switch msg := msg.(type) {
	case utils.ViewerStringData:
		m.content = string(msg)
		m.table = utils.InitTable([][]string{}, m.layout.ViewerWidth-2, m.layout.ViewerHeight-2)
	case utils.ViewerTableData:
		m.table = createTableFromData(msg, m.layout)
		m.content = ""
	case utils.ActiveViewChanged:
		m.isActive = string(msg) == "viewer"
	case utils.LayoutUpdated:
		m.layout = utils.ConnectionContainerLayout(msg)
	case cellUpdatedMsg:
		if msg.err != nil {
			m.content = fmt.Sprintf("Failed to save edit: %s", msg.err)
		} else if msg.row >= 0 && msg.row < len(m.table.Rows) {
			m.table.Rows[msg.row][msg.col] = msg.newValue
		}
		return m, nil
	case tea.KeyMsg:
		if msg.String() == "e" && m.table.HasData() {
			row, col := m.table.SelectedRow, m.table.SelectedColumn
			current := ""
			if row >= 0 && row < len(m.table.Rows) && col >= 0 && col < len(m.table.Rows[row]) {
				current = m.table.Rows[row][col]
			}
			input := textinput.New()
			input.SetValue(current)
			input.CursorEnd()
			input.Focus()
			m.editInput = input
			m.editingCell = true
			return m, nil
		}
	}
	m.table, viewPortCmd = m.table.Update(msg)
	return m, viewPortCmd
}

func (m ViewerModel) View() string {
	var content string
	if m.editingCell {
		content = m.renderEditModal()
	} else if m.table.HasData() {
		content = fmt.Sprintf("%s\n", m.table.View())
	} else {
		content = m.content
	}
	return utils.RenderPanel("3 Viewer", content, m.layout.ViewerWidth, m.layout.ViewerHeight, m.isActive)
}

func (m ViewerModel) renderEditModal() string {
	title := "Edit value"
	if m.currentTable != "" {
		title = fmt.Sprintf("Edit value (writes to %s)", m.currentTable)
	} else {
		title = "Edit value (display only - no source table to update)"
	}
	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Padding(1, 2).
		Width(60)
	hint := lipgloss.NewStyle().Faint(true).Render("Save (enter)   Cancel (esc)")
	body := lipgloss.JoinVertical(lipgloss.Left, title, "", m.editInput.View(), "", hint)
	return lipgloss.Place(m.layout.ViewerWidth-2, m.layout.ViewerHeight-2, lipgloss.Center, lipgloss.Center, box.Render(body))
}
