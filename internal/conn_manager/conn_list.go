package conn_manager

import (
	"fmt"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"

	adapters "app.lazygit/internal/adapters"
	utils "app.lazygit/internal/utils"
	tea "github.com/charmbracelet/bubbletea"
)

type ConnectionList struct {
	connections        []adapters.DbConnection
	selectedConnectionIndex int
	viewport           viewport.Model
	layout             utils.ConnectionManagerLayout
}

func InitConnectionList(layout utils.ConnectionManagerLayout) ConnectionList {
	viewport := viewport.New(layout.ConnectionListWidth, layout.BodyHeight-2)
	model := ConnectionList{
		connections:        []adapters.DbConnection{},
		selectedConnectionIndex: 0,
		layout:             layout,
		viewport:           viewport,
	}
	model.viewport.SetContent(model.connectionsUI())
	return model
}

func (m ConnectionList) changeSelectedConnection(index int) tea.Cmd {
	return func() tea.Msg {
		return SelectedConnectionMsg(index)
	}
}

func (m ConnectionList) connectionsUI() string {
	var connectionsList []string
	normalStyle := lipgloss.NewStyle().
		Padding(0, 2)
	selectedStyle := lipgloss.NewStyle().
		Background(lipgloss.Color("57")).
		Foreground(lipgloss.Color("229")).
		Padding(0, 2)

	for i, conn := range m.connections {
		if i == m.selectedConnectionIndex {
			connectionsList = append(connectionsList, selectedStyle.Render(conn.Name))
		} else {
			connectionsList = append(connectionsList, normalStyle.Render(conn.Name))
		}
	}
	return lipgloss.JoinVertical(lipgloss.Left, connectionsList...)
}

func (m ConnectionList) Init() tea.Cmd {
	return nil
}

func (m ConnectionList) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var viewPortCmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up", "k":
			if m.selectedConnectionIndex > 0 {
				m.selectedConnectionIndex--
			}
			cmd = m.changeSelectedConnection(m.selectedConnectionIndex)
		case "down", "j":
			if m.selectedConnectionIndex < len(m.connections)-1 {
				m.selectedConnectionIndex++
			}
			cmd = m.changeSelectedConnection(m.selectedConnectionIndex)
		}
	case SelectedConnectionMsg:
		m.viewport.SetContent(m.connectionsUI())
	case LayoutUpdated:
		m.layout = utils.ConnectionManagerLayout(msg)
	case ConnectionsLoaded:
		m.connections = []adapters.DbConnection(msg)
		m.viewport.SetContent(m.connectionsUI())
	}
	m.viewport, viewPortCmd = m.viewport.Update(msg)
	return m, tea.Batch(cmd, viewPortCmd)
}

func (m ConnectionList) View() string {
	info := fmt.Sprintf("Current rows: %d\nTotal rows: %d", m.viewport.VisibleLineCount(), m.viewport.TotalLineCount())
	return fmt.Sprintf("%s\n%s", m.viewport.View(), info)
}
