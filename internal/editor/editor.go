package editor

import (
	adapters "app.lazygit/internal/adapters"
	utils "app.lazygit/internal/utils"
	tea "github.com/charmbracelet/bubbletea"
	lipgloss "github.com/charmbracelet/lipgloss"
	"github.com/kujtimiihoxha/vimtea"
)

// IsCapturingInput reports whether the underlying vim editor is in a mode
// that should swallow raw keystrokes (Insert/Command), meaning global
// window-navigation shortcuts (1/2/3, tab, shift+j/k) must NOT be
// intercepted so the user can still type digits/letters into their query.
func (m EditorModel) IsCapturingInput() bool {
	mode := m.editor.GetMode()
	return mode == vimtea.ModeInsert || mode == vimtea.ModeCommand
}

type EditorModel struct {
	database adapters.Database
	layout   utils.ConnectionContainerLayout
	isActive bool
	editor   vimtea.Editor
}

func InitEditor(database adapters.Database, layout utils.ConnectionContainerLayout) EditorModel {
	editor := vimtea.NewEditor(
		vimtea.WithEnableStatusBar(true),
		vimtea.WithSelectedStyle(lipgloss.NewStyle().Background(lipgloss.Color("240")).Foreground(lipgloss.Color("255"))),
	)
	runQuery := func(text string) tea.Cmd {
		return func() tea.Msg {
			rows, err := database.RunQuery(text)
			if err != nil {
				return utils.ViewerStringData(lipgloss.NewStyle().Foreground(lipgloss.Color("160")).Render(err.Error()))
			} else if len(rows) > 0 && len(rows[0]) > 0 {
				return utils.ViewerTableData(rows)
			} else {
				return utils.ViewerStringData(lipgloss.NewStyle().Foreground(lipgloss.Color("34")).Render("Query executed successfully"))
			}
		}
	}
	editor.AddBinding(vimtea.KeyBinding{
		Key:           "ctrl+r",
		Mode:          vimtea.ModeVisual,
		Description:   "Run the selected query",
		VisualHandler: func(text string) tea.Cmd { return runQuery(text) },
	})
	// ctrl+s runs the WHOLE query buffer, matching LazyCurl's "ctrl+s sends
	// the request" keybind - no need to visually select the text first like
	// ctrl+r requires.
	editor.AddBinding(vimtea.KeyBinding{
		Key:         "ctrl+s",
		Mode:        vimtea.ModeNormal,
		Description: "Run the whole query (like LazyCurl's send-request key)",
		Handler:     func(buf vimtea.Buffer) tea.Cmd { return runQuery(buf.Text()) },
	})
	return EditorModel{
		database: database,
		layout:   layout,
		isActive: false,
		editor:   editor,
	}
}

func (m EditorModel) Init() tea.Cmd {
	newEditor, cmd := m.editor.SetSize(m.layout.EditorWidth-2, m.layout.EditorHeight-2)
	m.editor = newEditor.(vimtea.Editor)
	return tea.Batch(m.editor.Init(), cmd)
}

func (m EditorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case utils.ActiveViewChanged:
		m.isActive = string(msg) == "editor"
	case utils.LayoutUpdated:
		m.layout = utils.ConnectionContainerLayout(msg)
		newEditor, sizeCmd := m.editor.SetSize(m.layout.EditorWidth-2, m.layout.EditorHeight-2)
		m.editor = newEditor.(vimtea.Editor)
		cmd = sizeCmd
	}
	newEditor, newCmd := m.editor.Update(msg)
	m.editor = newEditor.(vimtea.Editor)
	return m, tea.Batch(cmd, newCmd)
}

func (m EditorModel) View() string {
	return utils.RenderPanel("2 Editor", m.editor.View(), m.layout.EditorWidth, m.layout.EditorHeight, m.isActive)
}
