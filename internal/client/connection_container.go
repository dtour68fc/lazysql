package client

import (
	"golang.org/x/term"
	"os"

	adapters "app.lazygit/internal/adapters"
	editor "app.lazygit/internal/editor"
	utils "app.lazygit/internal/utils"
	viewer "app.lazygit/internal/viewer"
	tea "github.com/charmbracelet/bubbletea"
)

var MIN_WIDTH = 600
var MIN_HEIGHT = 400

// ConnectionContainerModel just holds the Editor and Viewer models once a
// connection is live. AppModel owns pane focus, layout composition, and the
// shared footer now (see app.go) - Connection Manager is a permanent pane 1
// alongside these, always visible, not something you toggle away from.
type ConnectionContainerModel struct {
	editor tea.Model
	viewer tea.Model
	layout utils.ConnectionContainerLayout
}

func InitConnectionContainer(database adapters.Database, initialQuery string, currentTable string) ConnectionContainerModel {
	width, height, err := term.GetSize(int(os.Stdin.Fd()))
	if err != nil {
		width = MIN_WIDTH
		height = MIN_HEIGHT
	}

	layout := utils.CalculateConnectionContainerLayout(width, height)
	return ConnectionContainerModel{
		editor: editor.InitEditor(database, layout, initialQuery, currentTable),
		viewer: viewer.InitViewer(database, layout, currentTable),
		layout: layout,
	}
}

func setLayout(width int, height int) tea.Cmd {
	return func() tea.Msg {
		return utils.LayoutUpdated(utils.CalculateConnectionContainerLayout(width, height))
	}
}

func (m ConnectionContainerModel) Init() tea.Cmd {
	return tea.Batch(
		m.editor.Init(),
		m.viewer.Init(),
	)
}

func (m ConnectionContainerModel) Update(msg tea.Msg) (ConnectionContainerModel, tea.Cmd) {
	var editorCmd, viewerCmd, command tea.Cmd
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		command = setLayout(msg.Width, msg.Height)
	case utils.LayoutUpdated:
		m.layout = utils.ConnectionContainerLayout(msg)
	}

	m.editor, editorCmd = m.editor.Update(msg)
	m.viewer, viewerCmd = m.viewer.Update(msg)

	return m, tea.Batch(command, editorCmd, viewerCmd)
}

// UpdateEditor/UpdateViewer route a message to only one of the two models -
// used by AppModel to send keystrokes exclusively to whichever of the two
// is actually focused, without also leaking them into the other.
func (m ConnectionContainerModel) UpdateEditor(msg tea.Msg) (ConnectionContainerModel, tea.Cmd) {
	var cmd tea.Cmd
	m.editor, cmd = m.editor.Update(msg)
	return m, cmd
}

func (m ConnectionContainerModel) UpdateViewer(msg tea.Msg) (ConnectionContainerModel, tea.Cmd) {
	var cmd tea.Cmd
	m.viewer, cmd = m.viewer.Update(msg)
	return m, cmd
}

// IsEditorCapturingInput reports whether the SQL editor is in a vim mode
// that should swallow raw keystrokes (Insert/Command) - global pane-jump
// keys (1/2/3, tab) must not be intercepted while true.
func (m ConnectionContainerModel) IsEditorCapturingInput() bool {
	if e, ok := m.editor.(editor.EditorModel); ok {
		return e.IsCapturingInput()
	}
	return false
}

// IsEditorInNormalMode reports whether the editor is in vim's Normal mode
// specifically - used to gate the global esc/q "quit the app" shortcuts so
// they never fire while esc or q still mean something to vim itself
// (leaving Insert/Command mode, or canceling a Visual selection). Defaults
// to true (safe to quit) if the underlying model isn't the real editor for
// some reason.
func (m ConnectionContainerModel) IsEditorInNormalMode() bool {
	if e, ok := m.editor.(editor.EditorModel); ok {
		return e.IsInNormalMode()
	}
	return true
}

// IsViewerEditingCell reports whether the Viewer's "e" edit-cell modal is
// currently open - global shortcuts (esc/q to quit, 1/2/3 to jump panes)
// must not be intercepted while true.
func (m ConnectionContainerModel) IsViewerEditingCell() bool {
	if v, ok := m.viewer.(viewer.ViewerModel); ok {
		return v.IsEditingCell()
	}
	return false
}

func (m ConnectionContainerModel) EditorView() string { return m.editor.View() }
func (m ConnectionContainerModel) ViewerView() string { return m.viewer.View() }
