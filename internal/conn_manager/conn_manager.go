package conn_manager

import (
	"fmt"
	"maps"
	"os"
	"slices"

	"app.lazygit/internal/utils"
	"github.com/charmbracelet/lipgloss"
	"golang.org/x/term"

	adapters "app.lazygit/internal/adapters"
	tea "github.com/charmbracelet/bubbletea"
)

type ConnectionManager struct {
	layout                 utils.ConnectionManagerLayout
	list                   tea.Model
	form                   tea.Model
	connections            []adapters.DbConnection
	connectionsByName      map[string]adapters.DbConnection
	selectedConnectionName string
	editingConnection      bool
	connecting             bool
	savingConnection       bool
	showHelp               bool
	connectionError        string

	// Active project state - populated once space/enter loads a project's
	// tables in place (see LoadTablesMsg/OpenActiveConnectionMsg in
	// conn_list.go). Kept here (not in the list) since it holds the live
	// adapters.Database connection, reused rather than reconnecting when
	// the user finally opens the full 3-pane screen from the Tables tab.
	activeDatabase       adapters.Database
	activeConnectionName string
}

type SelectedConnectionMsg adapters.DbConnection
type EditConnectionMsg bool
type ConnectionErrorMsg string

// ConnectedMsg signals a successful connection to AppModel.
type ConnectedMsg struct {
	Database adapters.Database
}
type LayoutUpdated utils.ConnectionManagerLayout
type SavedConnectionsLoaded map[string]adapters.DbConnection
type ConnectionsLoaded []adapters.DbConnection

func setLayout(width int, height int) tea.Cmd {
	return func() tea.Msg {
		return LayoutUpdated(utils.CalculateConnectionManagerLayout(width, height))
	}
}

// quickConnectFromForm connects using whatever's currently typed into the
// form (used while actively editing a connection, as a "test this
// connection right now" shortcut) and jumps straight to the full screen on
// success.
func (m ConnectionManager) quickConnectFromForm() tea.Cmd {
	form := m.form.(ConnectionForm)
	connection := form.toDbConnection()
	return func() tea.Msg {
		database, err := connection.InitConnection()
		if err != nil {
			return ConnectionErrorMsg(fmt.Sprintf("Failed to connect: %s", err))
		}
		return ConnectedMsg{Database: database}
	}
}

// loadTablesForProject connects to the given (saved) connection and fetches
// its tables, without leaving the Connection Manager screen - populates the
// Tables tab in place. Picks the first database InitConnection's target
// returns from GetDatabases() (there's no per-connection "default database"
// concept in DbConnection today).
func (m ConnectionManager) loadTablesForProject(conn adapters.DbConnection) tea.Cmd {
	return func() tea.Msg {
		database, err := conn.InitConnection()
		if err != nil {
			return TablesErrorMsgInternal{ProjectName: conn.Name, Err: err.Error()}
		}
		databases, err := database.GetDatabases()
		if err != nil {
			return TablesErrorMsgInternal{ProjectName: conn.Name, Err: err.Error()}
		}
		if len(databases) == 0 {
			return TablesLoadedMsgInternal{ProjectName: conn.Name, Database: database, Tables: nil}
		}
		tables, err := database.GetTables(databases[0])
		if err != nil {
			return TablesErrorMsgInternal{ProjectName: conn.Name, Err: err.Error()}
		}
		return TablesLoadedMsgInternal{ProjectName: conn.Name, Database: database, Tables: tables}
	}
}

// TablesLoadedMsgInternal / TablesErrorMsgInternal carry the live
// adapters.Database (which conn_list.go can't reference without an import
// cycle concern - it's kept in ConnectionManager instead) alongside the
// fetch result.
type TablesLoadedMsgInternal struct {
	ProjectName string
	Database    adapters.Database
	Tables      []string
}
type TablesErrorMsgInternal struct {
	ProjectName string
	Err         string
}

func (m ConnectionManager) toggleConnectionEdit() tea.Cmd {
	return func() tea.Msg {
		return EditConnectionMsg(m.editingConnection)
	}
}

func loadSavedConnections() tea.Cmd {
	return func() tea.Msg {
		connections, err := getConnections()
		if err != nil {
			return ConnectionErrorMsg(fmt.Sprintf("Failed to load connections: %s", err))
		}
		return SavedConnectionsLoaded(connections)
	}
}

func (m ConnectionManager) loadConnections() tea.Cmd {
	return func() tea.Msg {
		savedConnections := slices.Collect(maps.Values(m.connectionsByName))
		connections := append(savedConnections, initializeNewConnection())
		return ConnectionsLoaded(connections)
	}
}

func (m ConnectionManager) saveConnection() tea.Cmd {
	return func() tea.Msg {
		form := m.form.(ConnectionForm)
		connection := form.toDbConnection()
		if conn, exists := m.connectionsByName[m.selectedConnectionName]; exists {
			delete(m.connectionsByName, conn.Name)
			deleteFromKeyring(conn.Name)
		}
		m.connectionsByName[connection.Name] = connection
		err := saveConnections(m.connectionsByName)
		if err != nil {
			return ConnectionErrorMsg(fmt.Sprintf("Failed to save connection: %s", err))
		}
		return SavedConnectionsLoaded(m.connectionsByName)
	}
}

// hasRealConnectionsCmd tells the list whether there are any real saved
// connections, right after they're (re)loaded.
func hasRealConnectionsCmd(has bool) tea.Cmd {
	return func() tea.Msg {
		return HasRealConnectionsMsg(has)
	}
}

func InitConnectionManager() ConnectionManager {
	var connections []adapters.DbConnection
	width, height, err := term.GetSize(int(os.Stdin.Fd()))
	if err != nil {
		width = 1
		height = 1
	}
	layout := utils.CalculateConnectionManagerLayout(width, height)
	return ConnectionManager{
		layout:                 layout,
		list:                   InitConnectionList(layout),
		form:                   InitConnForm(layout),
		connections:            connections,
		editingConnection:      false,
		connecting:             false,
		connectionError:        "",
		selectedConnectionName: "",
	}
}

func (m ConnectionManager) Init() tea.Cmd {
	return tea.Batch(m.list.Init(), m.form.Init(), loadSavedConnections())
}

func (m ConnectionManager) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var listCmd, formCmd, command tea.Cmd
	m, command = m.handleKeyboardActions(msg)
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		command = setLayout(msg.Width, msg.Height)
	case ConnectionErrorMsg:
		m.connectionError = string(msg)
		m.connecting = false
	case LayoutUpdated:
		m.layout = utils.ConnectionManagerLayout(msg)
	case SavedConnectionsLoaded:
		m.connectionsByName = map[string]adapters.DbConnection(msg)
		command = tea.Batch(m.loadConnections(), hasRealConnectionsCmd(len(m.connectionsByName) > 0))
	case SelectedConnectionMsg:
		conn := adapters.DbConnection(msg)
		m.selectedConnectionName = conn.Name
	case LoadTablesMsg:
		conn := adapters.DbConnection(msg)
		m.connectionError = ""
		loadingMsgCmd := func() tea.Msg {
			return TablesStateMsg{Loading: true, ProjectName: conn.Name}
		}
		command = tea.Batch(loadingMsgCmd, m.loadTablesForProject(conn))
	case TablesLoadedMsgInternal:
		m.activeDatabase = msg.Database
		m.activeConnectionName = msg.ProjectName
		command = func() tea.Msg {
			return TablesStateMsg{ProjectName: msg.ProjectName, Tables: msg.Tables}
		}
	case TablesErrorMsgInternal:
		command = func() tea.Msg {
			return TablesStateMsg{Err: msg.Err, ProjectName: msg.ProjectName}
		}
	case OpenActiveConnectionMsg:
		if m.activeDatabase != nil {
			db := m.activeDatabase
			command = func() tea.Msg {
				return ConnectedMsg{Database: db}
			}
		}
	}

	if !m.editingConnection {
		m.list, listCmd = m.list.Update(msg)
	}
	m.form, formCmd = m.form.Update(msg)
	cmd := tea.Batch(listCmd, formCmd, command)
	return m, cmd
}

func (m ConnectionManager) View() string {
	if m.showHelp {
		helpView := renderHelp(m.layout.HelpWidth, m.layout.HelpHeight)
		return lipgloss.Place(m.layout.ScreenWidth, m.layout.ScreenHeight, lipgloss.Center, lipgloss.Center, helpView)
	}

	return lipgloss.Place(m.layout.ScreenWidth, m.layout.ScreenHeight, lipgloss.Center, lipgloss.Center, m.RenderPanel())
}

// RenderPanel renders just the bordered "Connection Manager" box itself,
// without centering it on the full screen - used by AppModel to hang it on
// the left side alongside empty Editor/Viewer placeholder panels.
func (m ConnectionManager) RenderPanel() string {
	footer := m.buildFooter()

	var body string
	if m.editingConnection {
		// The connection form is a centered popup modal (like LazyCurl's
		// New Project dialog) instead of a permanently-docked side panel -
		// list is hidden underneath it while editing, same as the
		// full-screen help overlay already replaces content elsewhere.
		modalArea := lipgloss.Place(m.layout.WinWidth, m.layout.BodyHeight, lipgloss.Center, lipgloss.Center, m.form.View())
		body = lipgloss.JoinVertical(lipgloss.Top, modalArea, footer)
	} else {
		listView := m.list.View()
		body = lipgloss.JoinVertical(lipgloss.Top, listView, footer)
	}

	// Same panel style as the connected screen (editor/viewer) - title
	// embedded directly in the rounded border, matching LazyCurl, instead
	// of a separate "Connection Manager" text line + rule inside a plain box.
	return utils.RenderPanel("Connection Manager", body, m.layout.WinWidth, m.layout.WinHeight, true)
}

// PanelWidth and PanelHeight expose the Connection Manager panel's own
// rendered dimensions, so AppModel can size the placeholder Editor/Viewer
// panels to line up with it.
func (m ConnectionManager) PanelWidth() int  { return m.layout.WinWidth }
func (m ConnectionManager) PanelHeight() int { return m.layout.WinHeight }

// IsShowingHelp reports whether the full-screen help overlay is active, in
// which case AppModel should just delegate to the normal View() instead of
// composing the hung-left layout (the help screen wants the whole terminal).
func (m ConnectionManager) IsShowingHelp() bool { return m.showHelp }

// IsEditingConnection reports whether the New/Edit Connection modal is
// currently open (typing into Host/Port/Username/etc.) - global pane-jump
// keys (1/2/3, tab) must not be intercepted while true, or typing a literal
// "1" or pressing tab to move between fields would get hijacked instead.
func (m ConnectionManager) IsEditingConnection() bool { return m.editingConnection }

func (m ConnectionManager) handleKeyboardActions(msg tea.Msg) (ConnectionManager, tea.Cmd) {
	var command tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			// While actively editing a connection, Enter is a "test this
			// connection right now" shortcut using whatever's currently
			// typed into the form, jumping straight to the full screen on
			// success. When NOT editing, Enter is owned by the list
			// instead (space/enter on a project row loads its tables in
			// place; enter on a table row opens the full screen using the
			// connection already established for that - see LoadTablesMsg
			// / OpenActiveConnectionMsg in conn_list.go).
			if m.editingConnection {
				m.connecting = true
				m.editingConnection = false
				connectionCommand := m.quickConnectFromForm()
				toggleConnectionCommand := m.toggleConnectionEdit()
				command = tea.Batch(connectionCommand, toggleConnectionCommand)
			}
		case "N":
			// Shift+N: jump straight to a blank "New Connection" and start
			// editing it immediately, same affordance as LazyCurl's
			// Shift+P/Shift+N "create from scratch" shortcuts.
			if !m.editingConnection {
				blank := initializeNewConnection()
				m.selectedConnectionName = ""
				m.editingConnection = true
				m.connecting = false
				m.connectionError = ""
				selectCmd := func() tea.Msg { return SelectedConnectionMsg(blank) }
				command = tea.Batch(selectCmd, m.toggleConnectionEdit())
			}
		case "e":
			if !m.editingConnection {
				m.editingConnection = true
				m.connecting = false
				m.connectionError = ""
				command = m.toggleConnectionEdit()
			}
		case "esc":
			if m.editingConnection {
				m.editingConnection = false
				command = m.toggleConnectionEdit()
			} else if m.showHelp {
				m.showHelp = false
			}
		case "s":
			if !m.editingConnection {
				m.savingConnection = true
				command = m.saveConnection()
			}
		case "?":
			if !m.editingConnection {
				m.showHelp = !m.showHelp
			}
		}
	}

	return m, command
}

func (m ConnectionManager) buildFooter() string {
	var footerContent string
	if m.connectionError != "" {
		footerContent = errorFooter(m.connectionError)
	} else if m.editingConnection {
		footerContent = editFooter()
	} else if m.connecting {
		footerContent = connectingFooter()
	} else if !m.editingConnection {
		footerContent = normalFooter()
	}
	return lipgloss.NewStyle().Width(m.layout.WinWidth).Height(m.layout.FooterHeight).Padding(1).Render(fmt.Sprintf("%s", footerContent))
}
