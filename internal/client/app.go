package client

import (
	"fmt"
	"os"

	conn_manager "app.lazygit/internal/conn_manager"
	utils "app.lazygit/internal/utils"
	tea "github.com/charmbracelet/bubbletea"
	lipgloss "github.com/charmbracelet/lipgloss"
)

// AppModel keeps the Connection Manager (Projects/Databases) permanently
// visible as pane 1, with Editor (pane 2) and Viewer (pane 3) on the right -
// always all three, never a full-screen toggle between "browsing projects"
// and "writing a query". Connecting to a project just makes Editor/Viewer
// go from placeholder to live; it never hides the Connection Manager.
type AppModel struct {
	connectionManager   conn_manager.ConnectionManager
	connectionContainer *ConnectionContainerModel // nil until a connection succeeds at least once
	activePane          string                    // "manager" | "editor" | "viewer"
	width               int
	height              int
}

func StartApp() {
	tea.NewProgram(initModel(), tea.WithAltScreen()).Run()
}

func initModel() AppModel {
	return AppModel{
		connectionManager: conn_manager.InitConnectionManager(),
		activePane:        "manager",
	}
}

func (m AppModel) Init() tea.Cmd {
	return m.connectionManager.Init()
}

// rightWidth computes how much horizontal space Editor/Viewer get, given
// whatever the Connection Manager panel ends up sizing itself to.
func (m AppModel) rightWidth() int {
	w := m.width - m.connectionManager.PanelWidth()
	if w < 20 {
		w = 20
	}
	return w
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
		}

		canJumpPanes := !m.connectionManager.IsEditingConnection() &&
			!m.connectionManager.IsShowingHelp() &&
			!(m.activePane == "editor" && m.connectionContainer != nil && m.connectionContainer.IsEditorCapturingInput())

		if canJumpPanes {
			switch msg.String() {
			case "1", "ctrl+p":
				m.activePane = "manager"
				return m, nil
			case "2":
				m.activePane = "editor"
				return m.applyActiveViewChanged("editor")
			case "3":
				m.activePane = "viewer"
				return m.applyActiveViewChanged("viewer")
			case "tab":
				return m.cyclePane(1)
			case "shift+tab":
				return m.cyclePane(-1)
			}
		}
	case conn_manager.ConnectedMsg:
		cc := InitConnectionContainer(msg.Database, msg.AutoRunQuery)
		m.connectionContainer = &cc
		// Never yank focus onto the editor just because a connection
		// succeeded (whether that's picking a project, opening a table,
		// or quick-connecting from the form) - the editor/viewer go live
		// in the background, but you stay wherever you already were and
		// jump over yourself with 2/tab whenever you're ready.
		nextPane := m.activePane
		var sizeCmd tea.Cmd
		if m.width > 0 {
			updated, cmd := m.connectionContainer.Update(tea.WindowSizeMsg{Width: m.rightWidth(), Height: m.height})
			*m.connectionContainer = updated
			sizeCmd = cmd
		}
		// Also forward ConnectedMsg to the Connection Manager itself -
		// this branch used to return early without doing that at all,
		// which meant ConnectionManager.Update()'s own "case ConnectedMsg"
		// (added specifically to reset its connecting flag) never
		// actually ran. It kept showing "Connecting..." forever even
		// though the connection had already succeeded and you'd been
		// bounced straight into the (now live) editor.
		updatedCM, cmCmd := m.connectionManager.Update(msg)
		m.connectionManager = updatedCM.(conn_manager.ConnectionManager)
		return m, tea.Batch(m.connectionContainer.Init(), sizeCmd, m.applyActiveViewChangedCmd(nextPane), cmCmd)
	}

	return m.routeToActivePane(msg)
}

// applyActiveViewChanged tells the connected screen's editor/viewer which
// one is focused (so their border highlights correctly).
func (m AppModel) applyActiveViewChanged(view string) (tea.Model, tea.Cmd) {
	return m, m.applyActiveViewChangedCmd(view)
}

func (m AppModel) applyActiveViewChangedCmd(view string) tea.Cmd {
	if m.connectionContainer == nil {
		return nil
	}
	return func() tea.Msg { return utils.ActiveViewChanged(view) }
}

// cyclePane moves focus forward (delta=1) or backward (delta=-1) through
// all three panes: Projects/Databases (manager) -> Editor -> Viewer -> back to
// manager. Global, always active regardless of which pane currently has
// focus - the Connection Manager's OWN Projects/Databases sub-tab switch uses
// shift+h/shift+l instead (see conn_list.go), matching LazyCurl's
// Collections/Envs convention, so plain tab is free for this.
func (m AppModel) cyclePane(delta int) (tea.Model, tea.Cmd) {
	order := []string{"manager", "editor", "viewer"}
	idx := 0
	for i, p := range order {
		if p == m.activePane {
			idx = i
			break
		}
	}
	idx = (idx + delta + len(order)) % len(order)
	m.activePane = order[idx]
	if m.activePane != "manager" {
		return m.applyActiveViewChanged(m.activePane)
	}
	return m, nil
}

// routeToActivePane sends msg (if non-nil) to whichever pane currently has
// focus, and ALWAYS keeps the Connection Manager and the connected screen's
// editor/viewer updated for non-keystroke messages (async data loads,
// resize, etc.) - only raw keystrokes are exclusive to the focused pane, so
// e.g. pressing 's' while writing a query doesn't also silently trigger
// "save connection" in the (still-visible, but unfocused) Connection
// Manager pane.
func (m AppModel) routeToActivePane(msg tea.Msg) (tea.Model, tea.Cmd) {
	_, isKey := msg.(tea.KeyMsg)

	if !isKey {
		var cmCmd, ccCmd tea.Cmd
		var updatedCM tea.Model
		updatedCM, cmCmd = m.connectionManager.Update(msg)
		m.connectionManager = updatedCM.(conn_manager.ConnectionManager)
		if m.connectionContainer != nil {
			// Split the shared width/height message into the narrower
			// right-hand size the connected screen actually gets.
			if wsMsg, ok := msg.(tea.WindowSizeMsg); ok {
				wsMsg = tea.WindowSizeMsg{Width: m.rightWidth(), Height: wsMsg.Height}
				updated, cmd := m.connectionContainer.Update(wsMsg)
				*m.connectionContainer = updated
				ccCmd = cmd
			} else {
				updated, cmd := m.connectionContainer.Update(msg)
				*m.connectionContainer = updated
				ccCmd = cmd
			}
		}
		return m, tea.Batch(cmCmd, ccCmd)
	}

	switch m.activePane {
	case "manager":
		updated, cmd := m.connectionManager.Update(msg)
		m.connectionManager = updated.(conn_manager.ConnectionManager)
		return m, cmd
	case "editor":
		if m.connectionContainer == nil {
			return m, nil
		}
		updated, cmd := m.connectionContainer.UpdateEditor(msg)
		*m.connectionContainer = updated
		return m, cmd
	case "viewer":
		if m.connectionContainer == nil {
			return m, nil
		}
		updated, cmd := m.connectionContainer.UpdateViewer(msg)
		*m.connectionContainer = updated
		return m, cmd
	}
	return m, nil
}

func (m AppModel) View() string {
	if m.connectionManager.IsShowingHelp() {
		return m.connectionManager.View()
	}

	left := m.connectionManager.RenderPanel()
	panelHeight := m.connectionManager.PanelHeight()
	rw := m.rightWidth()

	var editorView, viewerView string
	if m.connectionContainer != nil {
		editorView = m.connectionContainer.EditorView()
		viewerView = m.connectionContainer.ViewerView()
	} else {
		editorHeight := panelHeight * 25 / 100
		viewerHeight := panelHeight - editorHeight
		editorView = utils.RenderPanel(
			"2 Editor",
			"No connection yet.\n\nConnect to a database (pane 1) to start writing a query.",
			rw, editorHeight, m.activePane == "editor",
		)
		viewerView = utils.RenderPanel(
			"3 Viewer",
			"No data yet.",
			rw, viewerHeight, m.activePane == "viewer",
		)
	}

	right := lipgloss.JoinVertical(lipgloss.Left, editorView, viewerView)
	body := lipgloss.JoinHorizontal(lipgloss.Top, left, right)
	footer := m.buildFooter()

	full := lipgloss.JoinVertical(lipgloss.Left, body, footer)

	if m.width <= 0 || m.height <= 0 {
		return full
	}
	return lipgloss.Place(m.width, m.height, lipgloss.Left, lipgloss.Top, full)
}

func (m AppModel) buildFooter() string {
	badgeLabel := map[string]string{
		"manager": "1 PROJECTS",
		"editor":  "2 EDITOR",
		"viewer":  "3 VIEWER",
	}[m.activePane]
	badgeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#1e1e2e")).
		Background(lipgloss.Color("141")).
		Bold(true).
		Padding(0, 1)
	badge := badgeStyle.Render(badgeLabel)

	universal := "1/2/3: jump pane, tab/shift+tab: cycle panes"
	var specific string
	switch m.activePane {
	case "manager":
		specific = "enter/space: select, esc/h: back, shift+n: new project, e: edit, s: save, j/k: navigate, shift+h/l: projects/databases"
	case "editor":
		specific = "ctrl+r or ctrl+s: run query"
	case "viewer":
		specific = "j/k: rows, h/l: columns"
	}

	bindings := fmt.Sprintf("%s | %s", universal, specific)
	pid := fmt.Sprintf("PID: %d", os.Getpid())

	left := lipgloss.JoinHorizontal(lipgloss.Top,
		badge,
		lipgloss.NewStyle().Padding(0, 1).Render(bindings),
	)
	right := lipgloss.NewStyle().Padding(0, 1).Render(pid)

	spacerWidth := m.width - lipgloss.Width(left) - lipgloss.Width(right)
	if spacerWidth < 0 {
		spacerWidth = 0
	}
	spacer := lipgloss.NewStyle().Width(spacerWidth).Render("")

	return lipgloss.JoinHorizontal(lipgloss.Top, left, spacer, right)
}
