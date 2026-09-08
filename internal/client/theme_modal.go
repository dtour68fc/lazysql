package client

import (
	utils "app.lazygit/internal/utils"
	textinput "github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	lipgloss "github.com/charmbracelet/lipgloss"
)

// themeModal is the ctrl+t color editor: a preset selector (cycled with
// h/l, same pattern as the connection form's Driver field) followed by
// one labeled text field per customizable color (see utils.Theme), values
// are whatever lipgloss.Color accepts (ANSI 256 numbers like "57", or hex
// like "#5f00ff"). Lives on AppModel directly since it's app-wide, not
// specific to any one pane.
//
// focusIndex 0 is always the preset selector; 1..len(inputs) map to
// inputs[0..len(inputs)-1].
type themeModal struct {
	active      bool
	inputs      []textinput.Model
	focusIndex  int
	presetIndex int // index into utils.BundledThemes, or -1 once you've hand-edited a field away from the selected preset's values
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
	m := themeModal{active: true, inputs: inputs, focusIndex: 0, presetIndex: matchingPresetIndex(current)}
	m.focusInput()
	return m
}

// matchingPresetIndex returns which bundled preset (if any) exactly
// matches the given theme, so reopening the modal shows the right preset
// already selected instead of always starting back at index 0.
func matchingPresetIndex(t utils.Theme) int {
	for i, nt := range utils.BundledThemes {
		if nt.Theme == t {
			return i
		}
	}
	return -1
}

// focusInput focuses whichever textinput focusIndex currently points at
// (if it's not the preset selector, focusIndex 0, which has no input of
// its own to focus).
func (m *themeModal) focusInput() {
	for i := range m.inputs {
		if i == m.focusIndex-1 {
			m.inputs[i].Focus()
		} else {
			m.inputs[i].Blur()
		}
	}
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

// applyPreset overwrites every field with the given bundled preset's
// values, same as switching the connection form's Driver field auto-
// updates the Port field.
func (m *themeModal) applyPreset(index int) {
	if index < 0 || index >= len(utils.BundledThemes) {
		return
	}
	m.presetIndex = index
	values := utils.BundledThemes[index].Theme.ThemeFieldValues()
	for i := range m.inputs {
		if i < len(values) {
			m.inputs[i].SetValue(values[i])
		}
	}
}

func (m themeModal) fieldCount() int { return len(m.inputs) + 1 }

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
		m.focusIndex = (m.focusIndex + 1) % m.fieldCount()
		m.focusInput()
		return m, false, false
	case "shift+tab", "up":
		m.focusIndex = (m.focusIndex - 1 + m.fieldCount()) % m.fieldCount()
		m.focusInput()
		return m, false, false
	case "h", "left":
		if m.focusIndex == 0 && len(utils.BundledThemes) > 0 {
			next := m.presetIndex - 1
			if next < 0 {
				next = len(utils.BundledThemes) - 1
			}
			m.applyPreset(next)
			return m, false, false
		}
	case "l", "right":
		if m.focusIndex == 0 && len(utils.BundledThemes) > 0 {
			next := (m.presetIndex + 1) % len(utils.BundledThemes)
			m.applyPreset(next)
			return m, false, false
		}
	}
	if m.focusIndex == 0 {
		// Preset selector isn't a text field - nothing else to do with
		// a keypress here.
		return m, false, false
	}
	i := m.focusIndex - 1
	var cmd tea.Cmd
	m.inputs[i], cmd = m.inputs[i].Update(msg)
	_ = cmd // textinput's own blink cmd isn't needed synchronously here
	// Any manual edit means the fields no longer necessarily match
	// whichever preset was last selected - stop claiming one is active.
	if m.presetIndex >= 0 && m.values()[i] != utils.BundledThemes[m.presetIndex].Theme.ThemeFieldValues()[i] {
		m.presetIndex = -1
	}
	return m, false, false
}

func (m themeModal) presetLabel() string {
	if len(utils.BundledThemes) == 0 {
		return "(no bundled presets found)"
	}
	name := "custom"
	if m.presetIndex >= 0 && m.presetIndex < len(utils.BundledThemes) {
		name = utils.BundledThemes[m.presetIndex].Name
	}
	return "< " + name + " >"
}

func (m themeModal) View() string {
	labels := utils.ThemeFieldNames()
	box := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(1, 2).Width(60)
	labelStyle := lipgloss.NewStyle().Width(24)

	presetStyle := labelStyle
	if m.focusIndex == 0 {
		presetStyle = presetStyle.Bold(true)
	}

	var rows []string
	rows = append(rows, "Color scheme (ANSI number like 57, or hex like #5f00ff)", "")
	rows = append(rows, lipgloss.JoinHorizontal(lipgloss.Top, presetStyle.Render("Preset (h/l): "), m.presetLabel()))
	rows = append(rows, "")
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
