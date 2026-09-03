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

// LoadTablesMsg is sent when the user presses space/enter on a project row
// in the Projects tab - signals ConnectionManager to connect to that
// connection and fetch its tables, populating the Tables tab in place
// (without leaving the Connection Manager screen).
type LoadTablesMsg adapters.DbConnection

// OpenActiveConnectionMsg is sent when the user presses enter on a table
// row in the (unlocked) Tables tab - signals ConnectionManager to open the
// full 3-pane screen using the connection that was already established to
// load those tables, without reconnecting.
type OpenActiveConnectionMsg struct{}

// TablesStateMsg pushes the Tables tab's data into ConnectionList - owned
// and fetched by ConnectionManager (which has the live adapters.Database),
// but rendered here alongside the Projects list.
type TablesStateMsg struct {
	Loading     bool
	ProjectName string
	Tables      []string
	Err         string
}

// HasRealConnectionsMsg tells the list whether there are any real saved
// connections (excluding the synthetic trailing "New Connection" row), so
// the Projects tab can show its empty state / lock the Tables tab.
type HasRealConnectionsMsg bool

type ConnectionList struct {
	connections             []adapters.DbConnection
	selectedConnectionIndex int
	viewport                viewport.Model
	layout                  utils.ConnectionManagerLayout
	activeTab               string // "projects" | "tables"
	hasRealConnections      bool

	// Tables tab state (populated by ConnectionManager via TablesStateMsg)
	tablesLoading     bool
	tablesProjectName string
	tables            []string
	tablesError       string
	selectedTableIndex int
}

func InitConnectionList(layout utils.ConnectionManagerLayout) ConnectionList {
	viewport := viewport.New(layout.ConnectionListWidth, layout.BodyHeight-2)
	model := ConnectionList{
		connections:             []adapters.DbConnection{},
		selectedConnectionIndex: 0,
		layout:                  layout,
		viewport:                viewport,
		activeTab:               "projects",
	}
	model.viewport.SetContent(model.contentUI())
	return model
}

func (m ConnectionList) changeSelectedConnection(index int) tea.Cmd {
	return func() tea.Msg {
		return SelectedConnectionMsg(m.connections[index])
	}
}

// realConnections returns the connections list without the synthetic
// trailing "New Connection" placeholder row.
func (m ConnectionList) realConnections() []adapters.DbConnection {
	if !m.hasRealConnections {
		return nil
	}
	var real []adapters.DbConnection
	for _, c := range m.connections {
		if c.Name == "New Connection" && c.Host == "" && c.Port == "" && c.Username == "" && c.Url == "" && c.Command == "" {
			continue
		}
		real = append(real, c)
	}
	return real
}

// projectRows returns the Projects tab's display order: every real saved
// connection, sorted alphabetically by alias (Name), plus the synthetic
// "New Connection" row at the end for creating another one.
func (m ConnectionList) projectRows() []adapters.DbConnection {
	rows := append([]adapters.DbConnection{}, m.connections...)
	sort.SliceStable(rows, func(i, j int) bool {
		// Keep the synthetic "New Connection" placeholder pinned last
		iPlaceholder := rows[i].Name == "New Connection" && rows[i].Host == ""
		jPlaceholder := rows[j].Name == "New Connection" && rows[j].Host == ""
		if iPlaceholder != jPlaceholder {
			return jPlaceholder
		}
		return rows[i].Name < rows[j].Name
	})
	return rows
}

// tablesLocked reports whether the Tables tab should be locked/greyed out:
// true if there are no real connections at all, or no project has been
// connected + had its tables loaded yet this session.
func (m ConnectionList) tablesLocked() bool {
	// Locked until an active connection's tables have actually been loaded
	// (or are loading, or failed) - regardless of hasRealConnections. That
	// flag only tracks persisted/SAVED connections, but "quick connect
	// while editing" never saves anything and can still legitimately load
	// and populate this tab - hasRealConnections used to gate this too,
	// which meant a live, table-populated quick-connect stayed locked out
	// forever just because nothing had been saved to disk.
	return m.tablesProjectName == "" && !m.tablesLoading && m.tablesError == ""
}

func (m ConnectionList) moveSelection(delta int) int {
	rows := m.projectRows()
	if len(rows) == 0 {
		return m.selectedConnectionIndex
	}
	current := m.connections[m.selectedConnectionIndex]
	pos := -1
	for i, c := range rows {
		if c == current {
			pos = i
			break
		}
	}
	if pos == -1 {
		pos = 0
	}
	next := pos + delta
	if next < 0 || next >= len(rows) {
		return m.selectedConnectionIndex
	}
	target := rows[next]
	for i, c := range m.connections {
		if c == target {
			return i
		}
	}
	return m.selectedConnectionIndex
}

func (m ConnectionList) projectsUI() string {
	if !m.hasRealConnections {
		emptyStyle := lipgloss.NewStyle().Faint(true).Padding(1, 2)
		return emptyStyle.Render("No projects yet.\n\nPress Shift+N to add one.")
	}

	var lines []string
	normalStyle := lipgloss.NewStyle().Padding(0, 2)
	selectedStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("57")).
		Foreground(lipgloss.Color("229")).
		Padding(0, 2)

	var selected adapters.DbConnection
	if len(m.connections) > 0 {
		selected = m.connections[m.selectedConnectionIndex]
	}

	for _, row := range m.projectRows() {
		if row == selected {
			lines = append(lines, selectedStyle.Render(row.Name))
		} else {
			lines = append(lines, normalStyle.Render(row.Name))
		}
	}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m ConnectionList) tablesUI() string {
	if m.tablesLocked() {
		lockedStyle := lipgloss.NewStyle().Faint(true).Padding(1, 2)
		if !m.hasRealConnections {
			return lockedStyle.Render("🔒 Locked - add a project first (Shift+N).")
		}
		return lockedStyle.Render("🔒 Locked - select a project (space/enter) to load its tables.")
	}

	if m.tablesLoading {
		return lipgloss.NewStyle().Padding(1, 2).Render(fmt.Sprintf("Connecting to %s and loading tables...", m.tablesProjectName))
	}

	if m.tablesError != "" {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("161")).Padding(1, 2).Render(
			fmt.Sprintf("Failed to load tables for %s:\n%s", m.tablesProjectName, m.tablesError),
		)
	}

	if len(m.tables) == 0 {
		return lipgloss.NewStyle().Faint(true).Padding(1, 2).Render(fmt.Sprintf("%s has no tables.", m.tablesProjectName))
	}

	header := lipgloss.NewStyle().Bold(true).Padding(0, 2).Render(fmt.Sprintf("Tables in %s", m.tablesProjectName))
	normalStyle := lipgloss.NewStyle().Padding(0, 2)
	selectedStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("57")).
		Foreground(lipgloss.Color("229")).
		Padding(0, 2)

	lines := []string{header}
	for i, table := range m.tables {
		if i == m.selectedTableIndex {
			lines = append(lines, selectedStyle.Render(table))
		} else {
			lines = append(lines, normalStyle.Render(table))
		}
	}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m ConnectionList) contentUI() string {
	if m.activeTab == "tables" {
		return m.tablesUI()
	}
	return m.projectsUI()
}

// tabsUI renders the "Projects | Tables" tab header. Tables greys out when
// locked so it's visually clear you can't switch into useful content yet.
func (m ConnectionList) tabsUI() string {
	activeStyle := lipgloss.NewStyle().Bold(true).Underline(true)
	inactiveStyle := lipgloss.NewStyle().Faint(true)
	lockedStyle := lipgloss.NewStyle().Faint(true).Strikethrough(false)

	projectsLabel := "Projects"
	tablesLabel := "Tables"
	if m.tablesLocked() {
		tablesLabel = lockedStyle.Render(tablesLabel)
	}

	if m.activeTab == "tables" {
		projectsLabel = inactiveStyle.Render(projectsLabel)
		if !m.tablesLocked() {
			tablesLabel = activeStyle.Render("Tables")
		}
	} else {
		projectsLabel = activeStyle.Render(projectsLabel)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, projectsLabel, "  ", tablesLabel)
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
			if m.activeTab == "tables" && !m.tablesLocked() && len(m.tables) > 0 {
				if m.selectedTableIndex > 0 {
					m.selectedTableIndex--
				}
			} else if m.activeTab == "projects" {
				m.selectedConnectionIndex = m.moveSelection(-1)
				cmd = m.changeSelectedConnection(m.selectedConnectionIndex)
			}
			m.viewport.SetContent(m.contentUI())
		case "down", "j":
			if m.activeTab == "tables" && !m.tablesLocked() && len(m.tables) > 0 {
				if m.selectedTableIndex < len(m.tables)-1 {
					m.selectedTableIndex++
				}
			} else if m.activeTab == "projects" {
				m.selectedConnectionIndex = m.moveSelection(1)
				cmd = m.changeSelectedConnection(m.selectedConnectionIndex)
			}
			m.viewport.SetContent(m.contentUI())
		case "H", "L":
			// Shift+H/Shift+L switch Projects/Tables sub-tabs, matching
			// LazyCurl's Shift+H/L Collections/Envs convention. Plain tab
			// is reserved globally for cycling between the Projects/
			// Editor/Viewer panes instead (see AppModel).
			if m.activeTab == "tables" {
				m.activeTab = "projects"
			} else {
				m.activeTab = "tables"
			}
			m.viewport.SetContent(m.contentUI())
		case "enter", " ":
			if m.activeTab == "projects" && m.hasRealConnections {
				rows := m.projectRows()
				if m.selectedConnectionIndex >= 0 && m.selectedConnectionIndex < len(m.connections) {
					selected := m.connections[m.selectedConnectionIndex]
					// Don't try to "load tables" for the synthetic New
					// Connection placeholder row.
					isPlaceholder := selected.Name == "New Connection" && selected.Host == ""
					if !isPlaceholder && len(rows) > 0 {
						cmd = func() tea.Msg { return LoadTablesMsg(selected) }
					}
				}
			} else if m.activeTab == "tables" && !m.tablesLocked() && len(m.tables) > 0 {
				cmd = func() tea.Msg { return OpenActiveConnectionMsg{} }
			}
		}
	case SelectedConnectionMsg:
		// Keep the list's own selection index in sync too - this message
		// can originate from elsewhere (e.g. Shift+N jumping straight to
		// the "New Connection" row), not just from this list's own cursor
		// movement.
		target := adapters.DbConnection(msg)
		for i, c := range m.connections {
			if c == target {
				m.selectedConnectionIndex = i
				break
			}
		}
		m.viewport.SetContent(m.contentUI())
	case LayoutUpdated:
		m.layout = utils.ConnectionManagerLayout(msg)
	case ConnectionsLoaded:
		m.connections = []adapters.DbConnection(msg)
		m.viewport.SetContent(m.contentUI())
		cmd = m.changeSelectedConnection(m.selectedConnectionIndex)
	case HasRealConnectionsMsg:
		m.hasRealConnections = bool(msg)
		m.viewport.SetContent(m.contentUI())
	case TablesStateMsg:
		m.tablesLoading = msg.Loading
		m.tablesProjectName = msg.ProjectName
		m.tables = msg.Tables
		m.tablesError = msg.Err
		m.selectedTableIndex = 0
		m.viewport.SetContent(m.contentUI())
	}
	m.viewport, viewPortCmd = m.viewport.Update(msg)
	return m, tea.Batch(cmd, viewPortCmd)
}

func (m ConnectionList) View() string {
	info := fmt.Sprintf("Current rows: %d\nTotal rows: %d", m.viewport.VisibleLineCount(), m.viewport.TotalLineCount())
	return fmt.Sprintf("%s\n%s\n%s", m.tabsUI(), m.viewport.View(), info)
}
