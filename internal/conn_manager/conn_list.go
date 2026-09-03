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

// LoadDatabasesMsg is sent when the user presses space/enter on a project
// (server) row in the Projects tab - signals ConnectionManager to connect
// to that server and fetch the FULL list of its databases (like the old
// Explorer's top-level list), populating the Databases tab in place
// (without leaving the Connection Manager screen). No guessing which one
// has your tables - you see all of them and pick.
type LoadDatabasesMsg adapters.DbConnection

// OpenDatabaseMsg is sent when the user presses enter on a specific
// database row in the (unlocked) Databases tab - signals ConnectionManager
// to open the full editor/viewer screen targeting exactly that database,
// using the connection that was already established to list them (no
// reconnecting).
type OpenDatabaseMsg struct {
	DatabaseName string
}

// DatabasesStateMsg pushes the Databases tab's data into ConnectionList -
// owned and fetched by ConnectionManager (which has the live
// adapters.Database), but rendered here alongside the Projects list.
type DatabasesStateMsg struct {
	Loading     bool
	ProjectName string
	Databases   []string
	Err         string
}

// HasRealConnectionsMsg tells the list whether there are any real saved
// connections (excluding the synthetic trailing "New Connection" row), so
// the Projects tab can show its empty state / lock the Databases tab.
type HasRealConnectionsMsg bool

type ConnectionList struct {
	connections             []adapters.DbConnection
	selectedConnectionIndex int
	viewport                viewport.Model
	layout                  utils.ConnectionManagerLayout
	activeTab               string // "projects" | "databases"
	hasRealConnections      bool

	// Databases tab state (populated by ConnectionManager via
	// DatabasesStateMsg) - the full list of databases on the connected
	// server (project), not an auto-guessed single one.
	databasesLoading      bool
	databasesProjectName  string
	databases             []string
	databasesError        string
	selectedDatabaseIndex int
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
// connection (server), sorted alphabetically by alias (Name), plus the
// synthetic "New Connection" row at the end for creating another one.
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

// databasesLocked reports whether the Databases tab should be locked/greyed
// out: true if there are no real connections at all, or no server has been
// connected to (its database list loaded) yet this session.
func (m ConnectionList) databasesLocked() bool {
	return m.databasesProjectName == "" && !m.databasesLoading && m.databasesError == ""
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

func (m ConnectionList) databasesUI() string {
	if m.databasesLocked() {
		lockedStyle := lipgloss.NewStyle().Faint(true).Padding(1, 2)
		if !m.hasRealConnections {
			return lockedStyle.Render("🔒 Locked - add a project first (Shift+N).")
		}
		return lockedStyle.Render("🔒 Locked - select a project (space/enter) to load its databases.")
	}

	if m.databasesLoading {
		return lipgloss.NewStyle().Padding(1, 2).Render(fmt.Sprintf("Connecting to %s and loading databases...", m.databasesProjectName))
	}

	if m.databasesError != "" {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("161")).Padding(1, 2).Render(
			fmt.Sprintf("Failed to load databases for %s:\n%s", m.databasesProjectName, m.databasesError),
		)
	}

	if len(m.databases) == 0 {
		return lipgloss.NewStyle().Faint(true).Padding(1, 2).Render(fmt.Sprintf("%s has no databases.", m.databasesProjectName))
	}

	header := lipgloss.NewStyle().Bold(true).Padding(0, 2).Render(fmt.Sprintf("Databases on %s", m.databasesProjectName))
	normalStyle := lipgloss.NewStyle().Padding(0, 2)
	selectedStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("57")).
		Foreground(lipgloss.Color("229")).
		Padding(0, 2)

	lines := []string{header}
	for i, database := range m.databases {
		if i == m.selectedDatabaseIndex {
			lines = append(lines, selectedStyle.Render(database))
		} else {
			lines = append(lines, normalStyle.Render(database))
		}
	}
	return lipgloss.JoinVertical(lipgloss.Left, lines...)
}

func (m ConnectionList) contentUI() string {
	if m.activeTab == "databases" {
		return m.databasesUI()
	}
	return m.projectsUI()
}

// tabsUI renders the "Projects | Databases" tab header. Databases greys out
// when locked so it's visually clear you can't switch into useful content
// yet.
func (m ConnectionList) tabsUI() string {
	activeStyle := lipgloss.NewStyle().Bold(true).Underline(true)
	inactiveStyle := lipgloss.NewStyle().Faint(true)
	lockedStyle := lipgloss.NewStyle().Faint(true)

	projectsLabel := "Projects"
	databasesLabel := "Databases"
	if m.databasesLocked() {
		databasesLabel = lockedStyle.Render(databasesLabel)
	}

	if m.activeTab == "databases" {
		projectsLabel = inactiveStyle.Render(projectsLabel)
		if !m.databasesLocked() {
			databasesLabel = activeStyle.Render("Databases")
		}
	} else {
		projectsLabel = activeStyle.Render(projectsLabel)
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, projectsLabel, "  ", databasesLabel)
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
			if m.activeTab == "databases" && !m.databasesLocked() && len(m.databases) > 0 {
				if m.selectedDatabaseIndex > 0 {
					m.selectedDatabaseIndex--
				}
			} else if m.activeTab == "projects" {
				m.selectedConnectionIndex = m.moveSelection(-1)
				cmd = m.changeSelectedConnection(m.selectedConnectionIndex)
			}
			m.viewport.SetContent(m.contentUI())
		case "down", "j":
			if m.activeTab == "databases" && !m.databasesLocked() && len(m.databases) > 0 {
				if m.selectedDatabaseIndex < len(m.databases)-1 {
					m.selectedDatabaseIndex++
				}
			} else if m.activeTab == "projects" {
				m.selectedConnectionIndex = m.moveSelection(1)
				cmd = m.changeSelectedConnection(m.selectedConnectionIndex)
			}
			m.viewport.SetContent(m.contentUI())
		case "H", "L":
			// Shift+H/Shift+L switch Projects/Databases sub-tabs, matching
			// LazyCurl's Shift+H/L Collections/Envs convention. Plain tab
			// is reserved globally for cycling between the Projects/
			// Editor/Viewer panes instead (see AppModel).
			if m.activeTab == "databases" {
				m.activeTab = "projects"
			} else {
				m.activeTab = "databases"
			}
			m.viewport.SetContent(m.contentUI())
		case "enter", " ":
			if m.activeTab == "projects" && m.hasRealConnections {
				rows := m.projectRows()
				if m.selectedConnectionIndex >= 0 && m.selectedConnectionIndex < len(m.connections) {
					selected := m.connections[m.selectedConnectionIndex]
					// Don't try to "load databases" for the synthetic New
					// Connection placeholder row.
					isPlaceholder := selected.Name == "New Connection" && selected.Host == ""
					if !isPlaceholder && len(rows) > 0 {
						cmd = func() tea.Msg { return LoadDatabasesMsg(selected) }
					}
				}
			} else if m.activeTab == "databases" && !m.databasesLocked() && len(m.databases) > 0 {
				chosen := m.databases[m.selectedDatabaseIndex]
				cmd = func() tea.Msg { return OpenDatabaseMsg{DatabaseName: chosen} }
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
	case DatabasesStateMsg:
		m.databasesLoading = msg.Loading
		m.databasesProjectName = msg.ProjectName
		m.databases = msg.Databases
		m.databasesError = msg.Err
		m.selectedDatabaseIndex = 0
		m.viewport.SetContent(m.contentUI())
	}
	m.viewport, viewPortCmd = m.viewport.Update(msg)
	return m, tea.Batch(cmd, viewPortCmd)
}

func (m ConnectionList) View() string {
	info := fmt.Sprintf("Current rows: %d\nTotal rows: %d", m.viewport.VisibleLineCount(), m.viewport.TotalLineCount())
	return fmt.Sprintf("%s\n%s\n%s", m.tabsUI(), m.viewport.View(), info)
}
