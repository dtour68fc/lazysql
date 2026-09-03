package client

import (
	"fmt"
	"golang.org/x/term"
	"os"

	adapters "app.lazygit/internal/adapters"
	editor "app.lazygit/internal/editor"
	explorer "app.lazygit/internal/explorer"
	utils "app.lazygit/internal/utils"
	viewer "app.lazygit/internal/viewer"
	tea "github.com/charmbracelet/bubbletea"
	lipgloss "github.com/charmbracelet/lipgloss"
)

var MIN_WIDTH = 600
var MIN_HEIGHT = 400

type ConnectionContainerModel struct {
	explorer    tea.Model
	editor      tea.Model
	viewer      tea.Model
	active_view string
	layout      utils.ConnectionContainerLayout
}

// InitConnectionContainer builds the connected 3-pane screen. preloadedDbName
// and preloadedTables are optional - when set (opened from an already-loaded
// Tables tab preview in the Connection Manager), Explorer opens with that
// database already expanded to its tables instead of starting fully
// collapsed and re-fetching everything the user just previewed a moment ago.
func InitConnectionContainer(database adapters.Database, preloadedDbName string, preloadedTables []string) ConnectionContainerModel {
	width, height, err := term.GetSize(int(os.Stdin.Fd()))
	if err != nil {
		width = MIN_WIDTH
		height = MIN_HEIGHT
	}

	layout := utils.CalculateConnectionContainerLayout(width, height)
	return ConnectionContainerModel{
		explorer:    explorer.InitExplorerPreloaded(database, layout, preloadedDbName, preloadedTables),
		editor:      editor.InitEditor(database, layout),
		viewer:      viewer.InitViewer(database, layout),
		active_view: "explorer",
		layout:      layout,
	}
}

func setLayout(width int, height int) tea.Cmd {
	return func() tea.Msg {
		return utils.LayoutUpdated(utils.CalculateConnectionContainerLayout(width, height))
	}
}

func (m ConnectionContainerModel) changeActiveView() tea.Cmd {
	return func() tea.Msg {
		var newActiveView string
		switch m.active_view {
		case "explorer":
			newActiveView = "editor"
		case "editor":
			newActiveView = "viewer"
		default:
			newActiveView = "explorer"
		}
		return utils.ActiveViewChanged(newActiveView)
	}
}

// changeActiveViewBackward is the reverse of changeActiveView, used by
// Shift+Tab (matching LazyCurl: Tab cycles forward, Shift+Tab cycles back).
func (m ConnectionContainerModel) changeActiveViewBackward() tea.Cmd {
	return func() tea.Msg {
		var newActiveView string
		switch m.active_view {
		case "explorer":
			newActiveView = "viewer"
		case "viewer":
			newActiveView = "editor"
		default:
			newActiveView = "explorer"
		}
		return utils.ActiveViewChanged(newActiveView)
	}
}

// jumpToView directly focuses a named view, same as LazyCurl's 1/2/3.
func jumpToView(view string) tea.Cmd {
	return func() tea.Msg {
		return utils.ActiveViewChanged(view)
	}
}

func (m ConnectionContainerModel) Init() tea.Cmd {
	return tea.Batch(
		m.explorer.Init(),
		m.editor.Init(),
		m.viewer.Init(),
	)
}

func (m ConnectionContainerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var explorerCmd, editorCmd, viewerCmd, command tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return m.handleKeyboardMsg(msg)
	case tea.WindowSizeMsg:
		command = setLayout(msg.Width, msg.Height)
	case utils.LayoutUpdated:
		m.layout = utils.ConnectionContainerLayout(msg)
	case utils.ActiveViewChanged:
		m.active_view = string(msg)
	}

	m.explorer, explorerCmd = m.explorer.Update(msg)
	m.editor, editorCmd = m.editor.Update(msg)
	m.viewer, viewerCmd = m.viewer.Update(msg)

	return m, tea.Batch(command, explorerCmd, editorCmd, viewerCmd)
}

func (m ConnectionContainerModel) handleKeyboardMsg(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	var cmd, activeViewCmd tea.Cmd

	// If the SQL editor is currently active and capturing raw keystrokes
	// (Insert/Command mode), global window-navigation shortcuts must be
	// disabled so the user can type digits/letters/tab into their query.
	editorCapturingInput := false
	if e, ok := m.editor.(editor.EditorModel); ok {
		editorCapturingInput = m.active_view == "editor" && e.IsCapturingInput()
	}

	if !editorCapturingInput {
		switch msg.String() {
		case "1":
			cmd = jumpToView("explorer")
		case "2":
			cmd = jumpToView("editor")
		case "3":
			cmd = jumpToView("viewer")
		case "tab":
			cmd = m.changeActiveView()
		case "shift+tab":
			cmd = m.changeActiveViewBackward()
		case "shift+j":
			if m.active_view == "editor" || m.active_view == "explorer" {
				cmd = jumpToView("viewer")
			}
		case "shift+k":
			if m.active_view == "viewer" || m.active_view == "explorer" {
				cmd = jumpToView("editor")
			}
		}
	}

	if m.active_view == "explorer" {
		m.explorer, activeViewCmd = m.explorer.Update(msg)
	} else if m.active_view == "editor" {
		m.editor, activeViewCmd = m.editor.Update(msg)
	} else {
		m.viewer, activeViewCmd = m.viewer.Update(msg)
	}
	return m, tea.Batch(cmd, activeViewCmd)
}

func (m ConnectionContainerModel) View() string {
	body := lipgloss.JoinHorizontal(lipgloss.Top,
		m.explorer.View(),
		lipgloss.JoinVertical(lipgloss.Left,
			m.editor.View(),
			m.viewer.View(),
		),
	)

	footer := m.buildFooter()

	return lipgloss.JoinVertical(lipgloss.Left, body, footer)
}

func (m ConnectionContainerModel) buildFooter() string {
	// Mode-style badge showing which numbered pane is active, matching
	// LazyCurl's colored mode badge on the left of the status bar.
	badgeLabel := map[string]string{
		"explorer": "1 EXPLORER",
		"editor":   "2 EDITOR",
		"viewer":   "3 VIEWER",
	}[m.active_view]
	badgeStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#1e1e2e")).
		Background(lipgloss.Color("141")).
		Bold(true).
		Padding(0, 1)
	badge := badgeStyle.Render(badgeLabel)

	universal := "1/2/3: jump pane, tab/shift+tab: cycle, shift+j/k: editor<->viewer, ctrl+p: switch projects"
	var specific string

	switch m.active_view {
	case "explorer":
		specific = "j/k: up/down, h/l: expand/collapse"
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

	spacerWidth := m.layout.ScreenWidth - lipgloss.Width(left) - lipgloss.Width(right)
	if spacerWidth < 0 {
		spacerWidth = 0
	}
	spacer := lipgloss.NewStyle().Width(spacerWidth).Render("")

	return lipgloss.JoinHorizontal(lipgloss.Top, left, spacer, right)
}
