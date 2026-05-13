package editor

import (
	adapters "app.lazygit/internal/adapters"
	utils "app.lazygit/internal/utils"
	tea "github.com/charmbracelet/bubbletea"
	lipgloss "github.com/charmbracelet/lipgloss"

	"github.com/kujtimiihoxha/vimtea"
)

type EditorModel struct {
	database adapters.Database
	layout   utils.ConnectionContainerLayout
	isActive bool
	editor   vimtea.Editor
}

func InitEditor(database adapters.Database, layout utils.ConnectionContainerLayout) EditorModel {
	editor := vimtea.NewEditor(
		vimtea.WithEnableStatusBar(true),
	)
	editor.AddBinding(vimtea.KeyBinding{
		Key:         "ctrl+r",
		Mode:        vimtea.ModeVisual,
		Description: "Run the selected query",
		Handler: func(buf vimtea.Buffer) tea.Cmd {
			return func() tea.Msg {
				return nil
			}
		},
		VisualHandler: func(text string) tea.Cmd {
			return func() tea.Msg {
				return utils.ViewerStringData(text)
			}
		},
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
	style := lipgloss.
		NewStyle().
		Border(lipgloss.RoundedBorder()).
		Width(m.layout.EditorWidth - 2).
		Height(m.layout.EditorHeight - 2)

	if m.isActive {
		style = style.BorderForeground(lipgloss.Color("205"))
	}
	return style.Render(m.editor.View())
}
