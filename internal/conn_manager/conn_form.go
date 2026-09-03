package conn_manager

import (
	adapters "app.lazygit/internal/adapters"
	utils "app.lazygit/internal/utils"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/lipgloss"
)

type ConnectionForm struct {
	inputs      []textinput.Model
	mode        string
	focusIndex  int
	layout      utils.ConnectionManagerLayout
	connections map[string]adapters.DbConnection
	driverIndex int // Index into DriverOptions for the Driver field (input 0)
}

func InitConnForm(layout utils.ConnectionManagerLayout) ConnectionForm {
	driverInput := createDriverInput(DriverOptions[0].Label)
	return ConnectionForm{
		inputs: []textinput.Model{
			driverInput,
			createNameInput(""),
			createHostInput(""),
			createPortInput(""),
			createUserInput(""),
			createPasswordInput(""),
			createUrlInput(""),
			createCommandInput(""),
			createDatabaseInput(""), // index 8 - appended, not inserted,
			// so none of the hardcoded 0-7 indices used throughout this
			// file shift around. Always visible regardless of mode (see
			// getVisibleIndices) - which database to list tables from
			// matters no matter how you're connecting.
		},
		focusIndex:  -1,
		layout:      layout,
		mode:        "credentials",
		driverIndex: 0,
	}
}

func (m ConnectionForm) changeFocusedInput() []tea.Cmd {
	var cmds []tea.Cmd
	for i := range m.inputs {
		if i == m.focusIndex {
			cmds = append(cmds, m.inputs[i].Focus())
		} else {
			m.inputs[i].Blur()
		}
	}
	return cmds
}

func (m ConnectionForm) updateInputs(msg tea.Msg) tea.Cmd {
	cmds := make([]tea.Cmd, len(m.inputs))

	for i := range m.inputs {
		m.inputs[i], cmds[i] = m.inputs[i].Update(msg)
	}

	return tea.Batch(cmds...)
}

func (m ConnectionForm) setSelectedConnection(conn adapters.DbConnection) ConnectionForm {
	m.driverIndex = driverIndexForValue(conn.Driver)
	m.inputs[0].SetValue(DriverOptions[m.driverIndex].Label)
	m.inputs[1].SetValue(conn.Name)
	m.inputs[2].SetValue(conn.Host)
	port := conn.Port
	if port == "" {
		// New/blank connection - prefill the driver's conventional
		// default port instead of leaving it empty, so you don't have to
		// go look up "what port does postgres use again" every time.
		port = DriverOptions[m.driverIndex].DefaultPort
	}
	m.inputs[3].SetValue(port)
	m.inputs[4].SetValue(conn.Username)
	m.inputs[5].SetValue(conn.Password)
	m.inputs[6].SetValue(conn.Url)
	m.inputs[7].SetValue(conn.Command)
	m.inputs[8].SetValue(conn.Database)

	if conn.Command != "" {
		m.mode = "command"
	} else if conn.Url != "" {
		m.mode = "url"
	} else {
		m.mode = "credentials"
	}
	return m
}

func (m ConnectionForm) Init() tea.Cmd {
	return nil
}

func (m ConnectionForm) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "tab", "shift+tab":
			if m.focusIndex != -1 {
				s := msg.String()
				m.focusIndex = m.changeFocusIndex(s)
				cmds := m.changeFocusedInput()
				return m, tea.Batch(cmds...)
			}
		case "m":
			// Also allowed while editing as long as the Driver field is
			// focused (index 0) - that field never accepts typed text (see
			// the h/l cycling below), so there's no ambiguity with typing a
			// literal "m". This used to require focusIndex == -1 (not
			// editing at all), which meant once you pressed 'e' to edit a
			// connection that was saved in url/command mode, you could
			// NEVER switch back to credentials mode to reveal Host/User/
			// Password - they just don't show up in url/command mode at
			// all (see getVisibleIndices), and there was no way out.
			if m.focusIndex == -1 || m.focusIndex == 0 {
				if m.mode == "credentials" {
					m.mode = "command"
				} else if m.mode == "command" {
					m.mode = "url"
				} else {
					m.mode = "credentials"
				}
				return m, nil
			}
		case "left", "h":
			if m.focusIndex == 0 {
				m.driverIndex = (m.driverIndex - 1 + len(DriverOptions)) % len(DriverOptions)
				m.inputs[0].SetValue(DriverOptions[m.driverIndex].Label)
				if isDefaultPort(m.inputs[3].Value()) {
					m.inputs[3].SetValue(DriverOptions[m.driverIndex].DefaultPort)
				}
				return m, nil
			}
		case "right", "l":
			if m.focusIndex == 0 {
				m.driverIndex = (m.driverIndex + 1) % len(DriverOptions)
				m.inputs[0].SetValue(DriverOptions[m.driverIndex].Label)
				if isDefaultPort(m.inputs[3].Value()) {
					m.inputs[3].SetValue(DriverOptions[m.driverIndex].DefaultPort)
				}
				return m, nil
			}
		}

		// The Driver field is a closed set of options (see DriverOptions) -
		// cycle with left/right/h/l above, never accept free-typed text here
		// (that's what let "PostgreSQL" get typed in instead of the actual
		// accepted value "pgx", silently saving an unusable connection).
		if m.focusIndex == 0 {
			return m, nil
		}
	case SelectedConnectionMsg:
		conn := adapters.DbConnection(msg)
		m = m.setSelectedConnection(conn)
	case EditConnectionMsg:
		canEdit := bool(msg)
		if canEdit {
			m.focusIndex = 0
		} else {
			m.focusIndex = -1
		}
		cmds := m.changeFocusedInput()
		return m, tea.Batch(cmds...)
	case LayoutUpdated:
		m.layout = utils.ConnectionManagerLayout(msg)
	case SavedConnectionsLoaded:
		connections := map[string]adapters.DbConnection(msg)
		m.connections = connections
	}
	cmd := m.updateInputs(msg)
	return m, cmd
}

func (m ConnectionForm) View() string {
	return m.RenderModal()
}

// RenderModal renders the connection form as a centered popup modal (like
// LazyCurl's "New Project" dialog) instead of a permanently-docked side
// panel - title, fields, and a Save/Cancel hint row, all in one bordered
// box. Only shown while actively editing (ConnectionManager.editingConnection).
func (m ConnectionForm) RenderModal() string {
	title := "New Connection"
	if name := m.inputs[1].Value(); name != "" && name != "New Connection" {
		title = "Edit Connection: " + name
	}
	titleStyle := lipgloss.NewStyle().Bold(true).Align(lipgloss.Center).Width(m.layout.ModalWidth - 4)

	indices := m.getVisibleIndices()
	fields := m.renderFieldsForIndexes(indices)

	buttonRow := lipgloss.NewStyle().
		Width(m.layout.ModalWidth - 4).
		Align(lipgloss.Center).
		MarginTop(1).
		Render("Save (s)   Cancel (esc)")

	content := lipgloss.JoinVertical(lipgloss.Left,
		append([]string{titleStyle.Render(title), ""}, append(fields, buttonRow)...)...,
	)

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("141")).
		Width(m.layout.ModalWidth).
		Padding(1, 2).
		Render(content)
}

func (m ConnectionForm) getVisibleIndices() []int {
	indices := []int{0, 1}
	if m.mode == "credentials" {
		indices = append(indices, 2, 3, 4, 5)
	} else if m.mode == "url" {
		indices = append(indices, 6)
	} else if m.mode == "command" {
		indices = append(indices, 7)
	}
	// Database (index 8) is always visible, same as Driver/Name, regardless
	// of credentials/url/command mode - it determines which database the
	// Tables tab lists, independent of how you connect.
	indices = append(indices, 8)
	return indices
}

func (m ConnectionForm) renderFieldsForIndexes(indexes []int) []string {
	var result []string
	for _, index := range indexes {
		var inputView string
		if index == m.focusIndex {
			inputView = utils.FocusedTextInputStyle().Render(m.inputs[index].View())
		} else {
			inputView = utils.TextInputStyle().Render(m.inputs[index].View())
		}
		result = append(result, inputView)
	}
	return result
}

func (m ConnectionForm) changeFocusIndex(key string) int {
	visibleIndices := m.getVisibleIndices()
	if len(visibleIndices) == 0 {
		return -1
	}

	currentIndexInVisible := -1
	for i, idx := range visibleIndices {
		if idx == m.focusIndex {
			currentIndexInVisible = i
			break
		}
	}

	if key == "tab" {
		if currentIndexInVisible == -1 {
			return visibleIndices[0]
		}
		return visibleIndices[(currentIndexInVisible+1)%len(visibleIndices)]
	} else if key == "shift+tab" {
		if currentIndexInVisible == -1 {
			return visibleIndices[len(visibleIndices)-1]
		}
		return visibleIndices[(currentIndexInVisible-1+len(visibleIndices))%len(visibleIndices)]
	}
	return m.focusIndex
}

func (m ConnectionForm) toDbConnection() adapters.DbConnection {
	return adapters.DbConnection{
		Driver:   DriverOptions[m.driverIndex].Value,
		Name:     m.inputs[1].Value(),
		Host:     m.inputs[2].Value(),
		Port:     m.inputs[3].Value(),
		Username: m.inputs[4].Value(),
		Password: m.inputs[5].Value(),
		Url:      m.inputs[6].Value(),
		Command:  m.inputs[7].Value(),
		Database: m.inputs[8].Value(),
	}
}
