package conn_manager

import (
	"fmt"
	"sort"

	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"

	adapters "app.lazygit/internal/adapters"
	utils "app.lazygit/internal/utils"
	tea "github.com/charmbracelet/bubbletea"
)

// UngroupedProject is the bucket label used for connections with no Project
// set (legacy connections created before project grouping existed) - same
// convention as LazyCurl's environment grouping.
const UngroupedProject = "Ungrouped"

type ConnectionList struct {
	connections             []adapters.DbConnection
	selectedConnectionIndex int
	viewport                viewport.Model
	layout                  utils.ConnectionManagerLayout
	activeTab               string // "connections" | "projects"
}

func InitConnectionList(layout utils.ConnectionManagerLayout) ConnectionList {
	viewport := viewport.New(layout.ConnectionListWidth, layout.BodyHeight-2)
	model := ConnectionList{
		connections:             []adapters.DbConnection{},
		selectedConnectionIndex: 0,
		layout:                  layout,
		viewport:                viewport,
		activeTab:               "connections",
	}
	model.viewport.SetContent(model.connectionsUI())
	return model
}

func (m ConnectionList) changeSelectedConnection(index int) tea.Cmd {
	return func() tea.Msg {
		return SelectedConnectionMsg(m.connections[index])
	}
}

// connectionRow is either a project header (not selectable) or a real
// connection row, in display order for whichever tab is active.
type connectionRow struct {
	isHeader bool
	label    string
	conn     adapters.DbConnection
}

// buildRows returns the display order for the active tab. In "connections"
// mode it's just the flat original order (unchanged from before this
// existed). In "projects" mode, connections are grouped under a header for
// their Project field (or "Ungrouped"), sorted alphabetically both by
// project and by connection name within each project.
func (m ConnectionList) buildRows() []connectionRow {
	if m.activeTab != "projects" {
		rows := make([]connectionRow, len(m.connections))
		for i, c := range m.connections {
			rows[i] = connectionRow{conn: c}
		}
		return rows
	}

	byProject := make(map[string][]adapters.DbConnection)
	for _, c := range m.connections {
		p := c.Project
		if p == "" {
			p = UngroupedProject
		}
		byProject[p] = append(byProject[p], c)
	}

	projectNames := make([]string, 0, len(byProject))
	for p := range byProject {
		projectNames = append(projectNames, p)
	}
	sort.Strings(projectNames)

	var rows []connectionRow
	for _, p := range projectNames {
		conns := byProject[p]
		sort.Slice(conns, func(i, j int) bool { return conns[i].Name < conns[j].Name })
		rows = append(rows, connectionRow{isHeader: true, label: p})
		for _, c := range conns {
			rows = append(rows, connectionRow{conn: c})
		}
	}
	return rows
}

// moveSelection moves the logical selection (an index into m.connections,
// identity-based) forward/backward through the CURRENT tab's display order,
// skipping over project header rows. Returns the same index unchanged if
// there's nowhere to move (already at the first/last connection).
func (m ConnectionList) moveSelection(delta int) int {
	if len(m.connections) == 0 {
		return m.selectedConnectionIndex
	}
	rows := m.buildRows()
	current := m.connections[m.selectedConnectionIndex]

	pos := -1
	for i, r := range rows {
		if !r.isHeader && r.conn == current {
			pos = i
			break
		}
	}
	if pos == -1 {
		return m.selectedConnectionIndex
	}

	next := pos
	for {
		next += delta
		if next < 0 || next >= len(rows) {
			return m.selectedConnectionIndex // no more connection rows that way
		}
		if !rows[next].isHeader {
			break
		}
	}

	target := rows[next].conn
	for i, c := range m.connections {
		if c == target {
			return i
		}
	}
	return m.selectedConnectionIndex
}

func (m ConnectionList) connectionsUI() string {
	var lines []string
	normalStyle := lipgloss.NewStyle().
		Padding(0, 2)
	selectedStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("57")).
		Foreground(lipgloss.Color("229")).
		Padding(0, 2)
	headerStyle := lipgloss.NewStyle().
		Bold(true).
		Padding(0, 1)

	var selected adapters.DbConnection
	if len(m.connections) > 0 {
		selected = m.connections[m.selectedConnectionIndex]
	}

	for _, row := range m.buildRows() {
		if row.isHeader {
			lines = append(lines, headerStyle.Render(row.label))
			continue
		}
		indent := ""
		if m.activeTab == "projects" {
			indent = "  "
		}
		if row.conn == selected {
			lines = append(lines, selectedStyle.Render(indent+row.conn.Name))
		} else {
			lines = append(lines, normalStyle.Render(indent+row.conn.Name))
		}
	}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

// tabsUI renders the "Connections | Projects" tab header, same idea as
// LazyCurl's Collections/Envs tabs.
func (m ConnectionList) tabsUI() string {
	activeStyle := lipgloss.NewStyle().Bold(true).Underline(true)
	inactiveStyle := lipgloss.NewStyle().Faint(true)

	connectionsLabel := "Connections"
	projectsLabel := "Projects"
	if m.activeTab == "projects" {
		connectionsLabel = inactiveStyle.Render(connectionsLabel)
		projectsLabel = activeStyle.Render(projectsLabel)
	} else {
		connectionsLabel = activeStyle.Render(connectionsLabel)
		projectsLabel = inactiveStyle.Render(projectsLabel)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, connectionsLabel, "  ", projectsLabel)
}

func (m ConnectionList) Init() tea.Cmd {
	return nil
}

func (m ConnectionList) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var viewPortCmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			m.selectedConnectionIndex = m.moveSelection(-1)
			cmd = m.changeSelectedConnection(m.selectedConnectionIndex)
			m.viewport.SetContent(m.connectionsUI())
		case "down", "j":
			m.selectedConnectionIndex = m.moveSelection(1)
			cmd = m.changeSelectedConnection(m.selectedConnectionIndex)
			m.viewport.SetContent(m.connectionsUI())
		case "tab":
			if m.activeTab == "projects" {
				m.activeTab = "connections"
			} else {
				m.activeTab = "projects"
			}
			m.viewport.SetContent(m.connectionsUI())
		}
	case SelectedConnectionMsg:
		m.viewport.SetContent(m.connectionsUI())
	case LayoutUpdated:
		m.layout = utils.ConnectionManagerLayout(msg)
	case ConnectionsLoaded:
		m.connections = []adapters.DbConnection(msg)
		m.viewport.SetContent(m.connectionsUI())
		cmd = m.changeSelectedConnection(m.selectedConnectionIndex)
	}
	m.viewport, viewPortCmd = m.viewport.Update(msg)
	return m, tea.Batch(cmd, viewPortCmd)
}

func (m ConnectionList) View() string {
	info := fmt.Sprintf("Current rows: %d\nTotal rows: %d", m.viewport.VisibleLineCount(), m.viewport.TotalLineCount())
	return fmt.Sprintf("%s\n%s\n%s", m.tabsUI(), m.viewport.View(), info)
}
