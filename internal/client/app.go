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

// bodyHeight is how much vertical space the Connection Manager/Editor/
// Viewer panels get to share, reserving exactly one row at the bottom for
// AppModel's own global footer. Sub-components used to get the FULL raw
// terminal height with nothing held back for that footer, which meant the
// footer's row pushed the whole composed view one row taller than the
// actual terminal - since terminals don't clip oversized content from the
// bottom, that extra row scrolled the true TOP row (the panels' own top
// borders) off-screen instead.
func (m AppModel) bodyHeight() int {
	h := m.height - 1
	if h < 1 {
		h = 1
	}
	return h
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
			!m.connectionManager.IsDumping() &&
			!m.connectionManager.IsImporting() &&
			!(m.activePane == "editor" && m.connectionContainer != nil && m.connectionContainer.IsEditorCapturingInput()) &&
			!(m.activePane == "viewer" && m.connectionContainer != nil && m.connectionContainer.IsViewerEditingCell())

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
			case ":":
				// Pressing : while looking at Projects/Databases or the
				// Viewer jumps straight to the editor AND forwards the
				// ":" itself, so vimtea drops right into command mode -
				// you don't have to manually jump to pane 2 first just to
				// start typing a shorthand query. Only kicks in once
				// there's actually an editor to send it to, and does
				// nothing extra if you're already on it (vimtea handles
				// ":" itself there, same as always).
				if m.activePane != "editor" && m.connectionContainer != nil {
					m.activePane = "editor"
					viewCmd := m.applyActiveViewChangedCmd("editor")
					updatedCC, keyCmd := m.connectionContainer.UpdateEditor(msg)
					*m.connectionContainer = updatedCC
					return m, tea.Batch(viewCmd, keyCmd)
				}
			case "esc":
				// esc quits, but only where esc doesn't already mean
				// something more specific: leaving vim's Insert/Visual/
				// Command mode (editor pane), or backing out of the
				// Databases tab's tables drill-down (manager pane) - in
				// both of those cases we fall through to the normal
				// routing below instead of returning here, so vimtea/
				// conn_list still get to handle it themselves.
				switch m.activePane {
				case "manager":
					if !m.connectionManager.IsInTablesDrilldown() {
						return m, tea.Quit
					}
				case "editor":
					if m.connectionContainer == nil || m.connectionContainer.IsEditorInNormalMode() {
						return m, tea.Quit
					}
				case "viewer":
					return m, tea.Quit
				}
			case "q":
				// q quits, but never while actually typing something -
				// editingConnection/showHelp are already excluded by
				// canJumpPanes above, so the only remaining case to guard
				// is the editor: q is a normal character in Insert mode,
				// and vimtea uses it as a real command in Visual mode too
				// (recording macros) - only quit from Normal mode there.
				if m.activePane != "editor" || m.connectionContainer == nil || m.connectionContainer.IsEditorInNormalMode() {
					return m, tea.Quit
				}
			}
		}
	case conn_manager.ConnectedMsg:
		cc := InitConnectionContainer(msg.Database, msg.AutoRunQuery, msg.Table)
		m.connectionContainer = &cc
		// Never yank focus onto the editor just because a connection
		// succeeded (whether that's picking a project, opening a table,
		// or quick-connecting from the form) - the editor/viewer go live
		// in the background, but you stay wherever you already were and
		// jump over yourself with 2/tab whenever you're ready.
		nextPane := m.activePane
		var sizeCmd tea.Cmd
		if m.width > 0 {
			updated, cmd := m.connectionContainer.Update(tea.WindowSizeMsg{Width: m.rightWidth(), Height: m.bodyHeight()})
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
		cmMsg := msg
		if wsMsg, ok := msg.(tea.WindowSizeMsg); ok {
			// Reserve one row for AppModel's own global footer - both the
			// Connection Manager panel and the connected Editor/Viewer
			// used to get the full raw terminal height with nothing held
			// back for it, which pushed the whole composed view one row
			// taller than the actual terminal and scrolled the true top
			// row (the panels' own top borders) off-screen.
			cmMsg = tea.WindowSizeMsg{Width: wsMsg.Width, Height: m.bodyHeight()}
		}
		updatedCM, cmCmd = m.connectionManager.Update(cmMsg)
		m.connectionManager = updatedCM.(conn_manager.ConnectionManager)
		if m.connectionContainer != nil {
			// Split the shared width/height message into the narrower
			// right-hand size the connected screen actually gets.
			if _, ok := msg.(tea.WindowSizeMsg); ok {
				wsMsg := tea.WindowSizeMsg{Width: m.rightWidth(), Height: m.bodyHeight()}
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
	if m.connectionManager.IsEditingConnection() {
		// The New/Edit Connection modal needs more width (50-70 cols)
		// than the narrow Connection Manager panel has to give it (it
		// used to be a full third of the screen, now it's about half
		// that) - render it as a full-screen takeover, same as the help
		// overlay, instead of squeezing it into the panel body and
		// having it overflow past the panel's own border.
		return m.connectionManager.View()
	}
	if m.connectionManager.IsDumping() {
		// Same reasoning as IsEditingConnection above - the ctrl+d dump
		// modal is also ~70 cols, too wide for the narrow panel.
		return m.connectionManager.View()
	}
	if m.connectionManager.IsImporting() {
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

	universal := "1/2/3: jump pane, tab/shift+tab: cycle panes, ': ' jumps to editor + command mode, esc/q: quit"
	var specific string
	switch m.activePane {
	case "manager":
		specific = "enter/space: select, esc/h: back, shift+n: new project, e: edit, s: save, j/k: navigate, shift+h/l: projects/databases"
	case "editor":
		specific = "ctrl+r or ctrl+s: run query"
	case "viewer":
		specific = "j/k: rows, h/l: columns, a: mark row+col, r: row view, shift+a/shift+d: sort asc/desc, e: edit cell"
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
