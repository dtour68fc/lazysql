package client

import (
	adapters "app.lazygit/internal/adapters"
	conn_manager "app.lazygit/internal/conn_manager"
	utils "app.lazygit/internal/utils"
	tea "github.com/charmbracelet/bubbletea"
	lipgloss "github.com/charmbracelet/lipgloss"
)

type AppModel struct {
	// connectionManager stays alive for the whole program run (never
	// destroyed on connect) so switching back to it with ctrl+p restores
	// exactly where you left off - same Projects list, same loaded Tables
	// tab state - instead of losing it and starting over.
	connectionManager   tea.Model
	connectionContainer tea.Model // nil until a connection succeeds at least once
	showingContainer    bool
	width               int
	height              int
}

func StartApp() {
	tea.NewProgram(initModel(), tea.WithAltScreen()).Run()
}

func initModel() AppModel {
	return AppModel{
		connectionManager: conn_manager.InitConnectionManager(),
	}
}

func (m AppModel) Init() tea.Cmd {
	return m.connectionManager.Init()
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "ctrl+p":
			// Switch back to the Projects list (Connection Manager)
			// without losing it or the connected screen - go pick a
			// different connection, or ctrl+p again... actually there's
			// no need to toggle back to the container from here; opening
			// a NEW connection (or the same one) re-enters it via
			// ConnectedMsg same as the first time.
			if m.showingContainer {
				m.showingContainer = false
				return m, nil
			}
		}
	case conn_manager.ConnectedMsg:
		database := msg.(adapters.Database)
		m.connectionContainer = InitConnectionContainer(database)
		m.showingContainer = true
		return m, m.connectionContainer.Init()
	}

	// Keep whichever screen is actually visible fully updated. The hidden
	// one only needs tea.WindowSizeMsg to stay correctly sized for later -
	// forwarding every message (especially keystrokes) to it while hidden
	// would leak them: e.g. pressing 's' while writing a SQL query would
	// also silently trigger "save connection" in the hidden Connection
	// Manager, or 'j'/'k' would move its list cursor out from under you.
	var cmCmd, ccCmd tea.Cmd
	if m.showingContainer {
		if _, ok := msg.(tea.WindowSizeMsg); ok {
			m.connectionManager, cmCmd = m.connectionManager.Update(msg)
		}
		if m.connectionContainer != nil {
			m.connectionContainer, ccCmd = m.connectionContainer.Update(msg)
		}
	} else {
		m.connectionManager, cmCmd = m.connectionManager.Update(msg)
	}

	return m, tea.Batch(cmCmd, ccCmd)
}

func (m AppModel) View() string {
	if m.showingContainer && m.connectionContainer != nil {
		// Already connected - ConnectionContainerModel renders its own
		// full 3-pane (explorer/editor/viewer) layout.
		return m.connectionContainer.View()
	}

	cm, ok := m.connectionManager.(conn_manager.ConnectionManager)
	if !ok || cm.IsShowingHelp() {
		return m.connectionManager.View()
	}
	return m.renderPreConnectLayout(cm)
}

// renderPreConnectLayout hangs the Connection Manager panel on the left,
// same as Explorer sits on the left of the connected screen, with empty
// Editor/Viewer placeholder panels on the right showing there's no query or
// data yet - so the app looks like its final 3-pane layout from the moment
// it starts, instead of a single box floating in the middle of the screen.
func (m AppModel) renderPreConnectLayout(cm conn_manager.ConnectionManager) string {
	left := cm.RenderPanel()
	panelHeight := cm.PanelHeight()

	rightWidth := m.width - cm.PanelWidth()
	if rightWidth < utils.EXPLORER_MIN_WIDTH {
		rightWidth = utils.EXPLORER_MIN_WIDTH
	}

	editorHeight := panelHeight * 25 / 100
	viewerHeight := panelHeight - editorHeight

	editorPlaceholder := utils.RenderPanel(
		"2 Editor",
		"No connection yet.\n\nConnect to a database (left) to start writing a query.",
		rightWidth, editorHeight, false,
	)
	viewerPlaceholder := utils.RenderPanel(
		"3 Viewer",
		"No data yet.",
		rightWidth, viewerHeight, false,
	)
	right := lipgloss.JoinVertical(lipgloss.Left, editorPlaceholder, viewerPlaceholder)

	body := lipgloss.JoinHorizontal(lipgloss.Top, left, right)

	if m.width <= 0 || m.height <= 0 {
		return body
	}
	// Top-aligned rather than centered - the panel now fills the full
	// available height itself, so there's nothing left to center; Top
	// keeps any rounding-error slack (e.g. tiny terminals hitting the
	// MIN_HEIGHT floor) at the bottom instead of splitting it top/bottom.
	return lipgloss.Place(m.width, m.height, lipgloss.Left, lipgloss.Top, body)
}
