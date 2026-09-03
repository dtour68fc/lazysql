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

	// Active project (server) state - populated once space/enter loads a
	// server's full database list in place (see LoadDatabasesMsg/
	// OpenDatabaseMsg in conn_list.go). Kept here (not in the list) since
	// it holds the live adapters.Database connection, reused rather than
	// reconnecting when the user finally opens the full editor/viewer
	// screen for a specific database.
	activeDatabase       adapters.Database
	activeConnectionName string

	// Session restore (see session.go) - saved every time you connect to
	// a project/database/table, restored automatically on the next
	// launch. sessionRestoreAttempted makes sure this only fires once,
	// since SavedConnectionsLoaded and SessionLoadedMsg race (both are
	// independent async Init() cmds, arriving in either order) - whichever
	// arrives second is what actually kicks off the restore-connect.
	pendingRestoreSession   SessionState
	connectionsLoaded       bool
	sessionRestoreAttempted bool
}

type SelectedConnectionMsg adapters.DbConnection
type EditConnectionMsg bool
type ConnectionErrorMsg string

// ConnectedMsg signals a successful connection to AppModel. ProjectName
// labels the Databases tab once we're connected - if empty (the "quick
// connect while editing" shortcut doesn't necessarily have a saved name), a
// generic label is used instead. DatabaseName is which specific database
// the editor's queries should target (see DbConnection.Database) - empty
// means whatever InitPostgres/InitMySQL default to (the admin db).
type ConnectedMsg struct {
	Database     adapters.Database
	ProjectName  string
	DatabaseName string
	// AutoRunQuery, when non-empty, gets seeded into the editor's buffer
	// and run immediately once the connected screen opens - used by
	// picking a table in the Databases tab's drill-down (see
	// OpenTableMsg) to jump straight to its "SELECT *" results, same as
	// opening a file in a netrw/oil.nvim style explorer.
	AutoRunQuery string
	// Table, when non-empty, seeds the editor's "current table" - lets
	// the query shorthand DSL (":sa", ":d", etc) default to this table
	// when you don't type one explicitly, e.g. just ":sa" instead of
	// ":sa users" while you're already looking at users.
	Table string
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
	name := connection.Name
	if name == "" || name == "New Connection" {
		name = "Quick Connection"
	}
	return func() tea.Msg {
		database, err := connection.InitConnection()
		if err != nil {
			return ConnectionErrorMsg(fmt.Sprintf("Failed to connect: %s", err))
		}
		return ConnectedMsg{Database: database, ProjectName: name, DatabaseName: connection.Database}
	}
}

// loadDatabasesForServer connects to the given (saved) connection/server and
// fetches its FULL list of databases, without leaving the Connection
// Manager screen - populates the Databases tab in place. No guessing which
// one has your tables - every database on the server is listed, same as
// the old Explorer pane's top level, and you explicitly pick one.
func (m ConnectionManager) loadDatabasesForServer(conn adapters.DbConnection) tea.Cmd {
	return func() tea.Msg {
		database, err := conn.InitConnection()
		if err != nil {
			return DatabasesErrorMsgInternal{ProjectName: conn.Name, Err: err.Error()}
		}
		return fetchDatabasesMsg(conn.Name, database)
	}
}

// fetchDatabasesForLiveConnection is loadDatabasesForServer's counterpart
// for a connection that's already established (e.g. the "quick connect
// while editing" shortcut) - lists its databases without reconnecting.
func fetchDatabasesForLiveConnection(name string, database adapters.Database) tea.Cmd {
	return func() tea.Msg {
		return fetchDatabasesMsg(name, database)
	}
}

// fetchDatabasesMsg does the actual GetDatabases work shared by both
// connect paths - the full list, no guessing or filtering.
func fetchDatabasesMsg(name string, database adapters.Database) tea.Msg {
	databases, err := database.GetDatabases()
	if err != nil {
		return DatabasesErrorMsgInternal{ProjectName: name, Err: err.Error()}
	}
	return DatabasesLoadedMsgInternal{ProjectName: name, Database: database, Databases: databases}
}

// DatabasesLoadedMsgInternal / DatabasesErrorMsgInternal carry the live
// adapters.Database (which conn_list.go can't reference without an import
// cycle concern - it's kept in ConnectionManager instead) alongside the
// fetch result.
type DatabasesLoadedMsgInternal struct {
	ProjectName string
	Database    adapters.Database
	Databases   []string
}
type DatabasesErrorMsgInternal struct {
	ProjectName string
	Err         string
}

// fetchTablesForDatabase lists the tables of a specific database on the
// already-live connection (switching the connection's active database as a
// side effect, same as GetTables always did) - no reconnecting needed, this
// reuses m.activeDatabase the same way fetchDatabasesForLiveConnection
// does.
func fetchTablesForDatabase(databaseName string, database adapters.Database) tea.Cmd {
	return func() tea.Msg {
		tables, err := database.GetTables(databaseName)
		if err != nil {
			return TablesStateMsg{DatabaseName: databaseName, Err: err.Error()}
		}
		return TablesStateMsg{DatabaseName: databaseName, Tables: tables}
	}
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

// deleteConnection removes a saved connection (pressing "d" on its row in
// the Projects tab) - persisted to disk and the keyring immediately, same
// as saving one.
func (m ConnectionManager) deleteConnection(name string) tea.Cmd {
	return func() tea.Msg {
		delete(m.connectionsByName, name)
		deleteFromKeyring(name)
		err := saveConnections(m.connectionsByName)
		if err != nil {
			return ConnectionErrorMsg(fmt.Sprintf("Failed to delete connection: %s", err))
		}
		return SavedConnectionsLoaded(m.connectionsByName)
	}
}

// performDump runs the actual table/database dump (see dump.go) - kept
// here rather than in conn_list.go since it needs the live
// adapters.Database connection, which only ConnectionManager holds.
func (m ConnectionManager) performDump(req DumpRequestMsg) tea.Cmd {
	db := m.activeDatabase
	return func() tea.Msg {
		var path string
		var err error
		if req.Kind == "table" {
			path, err = dumpTable(db, req.TableName, req.Folder)
		} else {
			path, err = dumpDatabase(db, req.DatabaseName, req.Folder)
		}
		if err != nil {
			return DumpResultMsg{Err: err.Error()}
		}
		return DumpResultMsg{Path: path}
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
	return tea.Batch(m.list.Init(), m.form.Init(), loadSavedConnections(), loadSessionCmd())
}

// SessionLoadedMsg carries whatever was saved from the last time you
// quit (see session.go) - empty (zero value) if there was no session file
// yet (brand new install).
type SessionLoadedMsg SessionState

func loadSessionCmd() tea.Cmd {
	return func() tea.Msg {
		s, _ := loadSession()
		return SessionLoadedMsg(s)
	}
}

// tryRestoreSession kicks off reconnecting to the last-used project (and,
// once its databases load, opening the last-used table) - guarded so it
// only ever fires once, and only once BOTH the saved connections and the
// saved session have actually loaded (they're independent async Init()
// cmds racing in either order).
func (m *ConnectionManager) tryRestoreSession() tea.Cmd {
	if m.sessionRestoreAttempted || !m.connectionsLoaded || m.pendingRestoreSession.ProjectName == "" {
		return nil
	}
	m.sessionRestoreAttempted = true
	conn, exists := m.connectionsByName[m.pendingRestoreSession.ProjectName]
	if !exists {
		return nil
	}
	return func() tea.Msg { return LoadDatabasesMsg(conn) }
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
	case ConnectedMsg:
		// A connect attempt just succeeded. This message is really
		// consumed by AppModel (which creates/shows the connected
		// Editor/Viewer), but it's still broadcast to this Update() too -
		// without this case, m.connecting stays stuck true forever,
		// showing "Connecting..." permanently in this panel's footer. That
		// silently didn't matter back when the whole Connection Manager
		// screen got hidden the moment you connected, but now that it's
		// permanently visible alongside Editor/Viewer, a stuck flag like
		// this is immediately visible instead of quietly irrelevant.
		m.connecting = false
		m.connectionError = ""
		// Also populate the Databases tab, regardless of which of the two
		// connect paths got us here (the Projects->Databases flow already
		// does this itself, but the "quick connect while editing"
		// shortcut never did - you'd end up connected with a live editor,
		// but Databases staying empty/locked forever since nothing ever
		// told it what was connected). Reuses the already-live Database -
		// no need to reconnect just to list them. Skipped when we already
		// have this exact project's database list loaded (picking a
		// project, then a table, then another table from the same
		// project all re-fire ConnectedMsg - no need to redo the fetch
		// and flicker the Databases tab back to "Loading..." every time).
		alreadyLoaded := m.activeConnectionName == msg.ProjectName && m.activeDatabase != nil
		m.activeDatabase = msg.Database
		m.activeConnectionName = msg.ProjectName
		if !alreadyLoaded {
			loadingMsgCmd := func() tea.Msg {
				return DatabasesStateMsg{Loading: true, ProjectName: msg.ProjectName}
			}
			command = tea.Batch(loadingMsgCmd, fetchDatabasesForLiveConnection(msg.ProjectName, msg.Database))
		}
	case LayoutUpdated:
		m.layout = utils.ConnectionManagerLayout(msg)
	case SavedConnectionsLoaded:
		m.connectionsByName = map[string]adapters.DbConnection(msg)
		m.connectionsLoaded = true
		command = tea.Batch(m.loadConnections(), hasRealConnectionsCmd(len(m.connectionsByName) > 0), m.tryRestoreSession())
	case SessionLoadedMsg:
		m.pendingRestoreSession = SessionState(msg)
		command = m.tryRestoreSession()
	case SelectedConnectionMsg:
		conn := adapters.DbConnection(msg)
		m.selectedConnectionName = conn.Name
	case LoadDatabasesMsg:
		conn := adapters.DbConnection(msg)
		m.connectionError = ""
		loadingMsgCmd := func() tea.Msg {
			return DatabasesStateMsg{Loading: true, ProjectName: conn.Name}
		}
		command = tea.Batch(loadingMsgCmd, m.loadDatabasesForServer(conn))
	case DeleteConnectionMsg:
		command = m.deleteConnection(msg.Name)
	case DatabasesLoadedMsgInternal:
		m.activeDatabase = msg.Database
		m.activeConnectionName = msg.ProjectName
		databasesCmd := func() tea.Msg {
			return DatabasesStateMsg{ProjectName: msg.ProjectName, Databases: msg.Databases}
		}
		// Picking a project already connects under the hood (to be able
		// to fetch its database list) - it used to stop there, leaving
		// the editor/viewer stuck on "No connection yet" placeholders
		// until you drilled all the way down to a specific table. Now
		// that the connection is proven live (databases fetched
		// successfully), make the editor/viewer live too, right away.
		// AppModel never steals focus on ConnectedMsg regardless of
		// which of the connect paths fired it, so this doesn't yank you
		// off the Databases tab you just opened either.
		connectedCmd := func() tea.Msg {
			return ConnectedMsg{Database: msg.Database, ProjectName: msg.ProjectName}
		}
		// If this load was kicked off to restore last session (see
		// SessionLoadedMsg/tryRestoreSession above), immediately continue
		// on to opening the saved table too, instead of leaving you on
		// the Databases tab having to pick it again yourself. Otherwise,
		// remember this as the new restore point in case you quit before
		// opening a specific table (session.go).
		var continueCmd tea.Cmd
		if m.pendingRestoreSession.ProjectName == msg.ProjectName && m.pendingRestoreSession.TableName != "" {
			restore := m.pendingRestoreSession
			m.pendingRestoreSession = SessionState{}
			continueCmd = func() tea.Msg {
				return OpenTableMsg{DatabaseName: restore.DatabaseName, TableName: restore.TableName}
			}
		} else {
			saveSession(SessionState{ProjectName: msg.ProjectName})
		}
		command = tea.Batch(databasesCmd, connectedCmd, continueCmd)
	case DatabasesErrorMsgInternal:
		command = func() tea.Msg {
			return DatabasesStateMsg{Err: msg.Err, ProjectName: msg.ProjectName}
		}
	case LoadTablesMsg:
		if m.activeDatabase != nil {
			dbName := msg.DatabaseName
			loadingMsgCmd := func() tea.Msg {
				return TablesStateMsg{Loading: true, DatabaseName: dbName}
			}
			command = tea.Batch(loadingMsgCmd, fetchTablesForDatabase(dbName, m.activeDatabase))
		}
	case OpenTableMsg:
		if m.activeDatabase != nil {
			db := m.activeDatabase
			name := m.activeConnectionName
			dbName := msg.DatabaseName
			tableName := msg.TableName
			saveSession(SessionState{ProjectName: name, DatabaseName: dbName, TableName: tableName})
			command = func() tea.Msg {
				return ConnectedMsg{
					Database:     db,
					ProjectName:  name,
					DatabaseName: dbName,
					AutoRunQuery: fmt.Sprintf("SELECT * FROM %s;", tableName),
					Table:        tableName,
				}
			}
		}
	case DumpRequestMsg:
		if m.activeDatabase != nil {
			command = m.performDump(msg)
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
	if m.editingConnection {
		// Full-screen takeover, same as the help overlay - the modal
		// needs 50-70 cols, more than the (now much narrower) Connection
		// Manager panel has room to give it without overflowing past its
		// own border.
		return lipgloss.Place(m.layout.ScreenWidth, m.layout.ScreenHeight, lipgloss.Center, lipgloss.Center, m.form.View())
	}

	return lipgloss.Place(m.layout.ScreenWidth, m.layout.ScreenHeight, lipgloss.Center, lipgloss.Center, m.RenderPanel())
}

// RenderPanel renders just the bordered "Connection Manager" box itself,
// without centering it on the full screen - used by AppModel to hang it on
// the left side alongside empty Editor/Viewer placeholder panels. Not used
// while editingConnection is true - AppModel takes the full-screen modal
// path (View()) instead, since the modal needs more width than this narrow
// panel can offer without overflowing its own border.
func (m ConnectionManager) RenderPanel() string {
	footer := m.buildFooter()
	listView := m.list.View()
	body := lipgloss.JoinVertical(lipgloss.Top, listView, footer)

	// Same panel style as the connected screen (editor/viewer) - title
	// embedded directly in the rounded border, matching LazyCurl, instead
	// of a separate "Connection Manager" text line + rule inside a plain box.
	return utils.RenderPanel("1 Connection", body, m.layout.WinWidth, m.layout.WinHeight, true)
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

// IsInTablesDrilldown reports whether the Databases tab is currently
// showing the tables-of-a-specific-database drill-down - used to gate the
// global esc "quit the app" shortcut, which must not fire here since esc
// already means "back out to the database list" in this state.
func (m ConnectionManager) IsInTablesDrilldown() bool {
	if list, ok := m.list.(ConnectionList); ok {
		return list.InTablesDrilldown()
	}
	return false
}

// IsDumping reports whether the Ctrl+D "dump to .sql" folder-path modal is
// currently open - see ConnectionList.IsDumping.
func (m ConnectionManager) IsDumping() bool {
	if list, ok := m.list.(ConnectionList); ok {
		return list.IsDumping()
	}
	return false
}

func (m ConnectionManager) handleKeyboardActions(msg tea.Msg) (ConnectionManager, tea.Cmd) {
	var command tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "enter":
			// While actively editing a connection, Enter both saves it
			// (same as pressing 's') AND tests connecting right now,
			// jumping straight to the full screen on success. Used to
			// only connect, never save - which meant the connection you
			// just typed in and successfully connected to never showed up
			// in the Projects list afterward, only surviving for the rest
			// of this session. Saving unconditionally (even if the
			// connect attempt fails) means you don't lose what you typed
			// either way. When NOT editing, Enter is owned by the list
			// instead (space/enter on a project row loads its full
			// database list in place; enter on a database row lists its
			// tables; enter on a table row opens the full screen with a
			// "SELECT *" already run - see LoadDatabasesMsg /
			// LoadTablesMsg / OpenTableMsg in conn_list.go).
			if m.editingConnection {
				m.connecting = true
				m.editingConnection = false
				connectionCommand := m.quickConnectFromForm()
				saveCommand := m.saveConnection()
				toggleConnectionCommand := m.toggleConnectionEdit()
				command = tea.Batch(connectionCommand, saveCommand, toggleConnectionCommand)
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
