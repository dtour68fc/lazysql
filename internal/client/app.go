package client

import (
	adapters "app.lazygit/internal/adapters"
	conn_manager "app.lazygit/internal/conn_manager"
	utils "app.lazygit/internal/utils"
	tea "github.com/charmbracelet/bubbletea"
	lipgloss "github.com/charmbracelet/lipgloss"
)

type AppModel struct {
	current_view tea.Model
	width        int
	height       int
}

func StartApp() {
	tea.NewProgram(initModel(), tea.WithAltScreen()).Run()
}

func initModel() AppModel {
	return AppModel{
		current_view: conn_manager.InitConnectionManager(),
	}
}

func (m AppModel) Init() tea.Cmd {
	return m.current_view.Init()
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	m.current_view, cmd = m.current_view.Update(msg)
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		}
	case conn_manager.ConnectedMsg:
		database := msg.(adapters.Database)
		m.current_view = InitConnectionContainer(database)
		cmd = m.current_view.Init()
	}

	return m, cmd
}

func (m AppModel) View() string {
	// Once connected, ConnectionContainerModel already renders its own full
	// 3-pane (explorer/editor/viewer) layout - nothing to compose here.
	if cm, ok := m.current_view.(conn_manager.ConnectionManager); ok && !cm.IsShowingHelp() {
		return m.renderPreConnectLayout(cm)
	}
	return m.current_view.View()
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

	editorHeight := panelHeight / 2
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
	return lipgloss.Place(m.width, m.height, lipgloss.Left, lipgloss.Center, body)
}
