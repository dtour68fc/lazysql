package client

import (
	utils "app.lazygit/internal/utils"
	textinput "github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	lipgloss "github.com/charmbracelet/lipgloss"
)

// themeModal is the ctrl+t color editor: one labeled text field per
// customizable color (see utils.Theme), values are whatever
// lipgloss.Color accepts (ANSI 256 numbers like "57", or hex like
// "#5f00ff"). Lives on AppModel directly since it's app-wide, not
// specific to any one pane.
type themeModal struct {
	active     bool
	inputs     []textinput.Model
	focusIndex int
}

func newThemeModal(current utils.Theme) themeModal {
	values := current.ThemeFieldValues()
	inputs := make([]textinput.Model, len(values))
	for i, v := range values {
		input := textinput.New()
		input.SetValue(v)
		input.CharLimit = 32
		inputs[i] = input
	}
	if len(inputs) > 0 {
		inputs[0].Focus()
	}
	return themeModal{active: true, inputs: inputs, focusIndex: 0}
}

func (m themeModal) values() []string {
	values := make([]string, len(m.inputs))
	for i, input := range m.inputs {
		values[i] = input.Value()
	}
	return values
}

func (m themeModal) theme() utils.Theme {
	return utils.ThemeFromFieldValues(m.values())
}

// updateThemeModal handles a keypress while the modal is open. Returns the
// updated modal, whether it should close+save, and whether it should
// close+cancel (esc) - AppModel.Update() applies whichever actually
// happened.
func (m themeModal) update(msg tea.KeyMsg) (themeModal, bool /*save*/, bool /*cancel*/) {
	switch msg.String() {
	case "esc":
		return m, false, true
	case "enter":
		return m, true, false
	case "tab", "down":
		m.inputs[m.focusIndex].Blur()
		m.focusIndex = (m.focusIndex + 1) % len(m.inputs)
		m.inputs[m.focusIndex].Focus()
		return m, false, false
	case "shift+tab", "up":
		m.inputs[m.focusIndex].Blur()
		m.focusIndex = (m.focusIndex - 1 + len(m.inputs)) % len(m.inputs)
		m.inputs[m.focusIndex].Focus()
		return m, false, false
	}
	var cmd tea.Cmd
	m.inputs[m.focusIndex], cmd = m.inputs[m.focusIndex].Update(msg)
	_ = cmd // textinput's own blink cmd isn't needed synchronously here
	return m, false, false
}

func (m themeModal) View() string {
	labels := utils.ThemeFieldNames()
	box := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1, 2).Width(60)
	labelStyle := lipgloss.NewStyle().Width(24)

	var rows []string
	rows = append(rows, "Color scheme (ANSI number like 57, or hex like #5f00ff)", "")
	for i, label := range labels {
		if i >= len(m.inputs) {
			break
		}
		row := lipgloss.JoinHorizontal(lipgloss.Top, labelStyle.Render(label+": "), m.inputs[i].View())
		rows = append(rows, row)
	}
	rows = append(rows, "", lipgloss.NewStyle().Faint(true).Render("Save (enter)   Cancel (esc)   tab/shift+tab: next/prev field"))
	return box.Render(lipgloss.JoinVertical(lipgloss.Left, rows...))
}
