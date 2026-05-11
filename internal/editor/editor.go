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
	lineNumberStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#888888")).
		Background(lipgloss.Color("#222222")).
		PaddingRight(1)

	currentLineStyle := lipgloss.NewStyle().
		Foreground(lipgloss.Color("white")).
		Background(lipgloss.Color("#444444")).
		Bold(true).
		PaddingRight(1)

	cursorStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("#CC8800")).
		Foreground(lipgloss.Color("black"))
	return EditorModel{
		database: database,
		layout:   layout,
		isActive: false,
		editor: vimtea.NewEditor(
			vimtea.WithLineNumberStyle(lineNumberStyle),
			vimtea.WithCurrentLineNumberStyle(currentLineStyle),
			vimtea.WithCursorStyle(cursorStyle),
			vimtea.WithRelativeNumbers(true),
			vimtea.WithContent("Initial content"),
			vimtea.WithEnableStatusBar(true),
			vimtea.WithDefaultSyntaxTheme("catppuccin-macchiato"),
		),
	}
}

func (m EditorModel) Init() tea.Cmd {
	return m.editor.Init()
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
