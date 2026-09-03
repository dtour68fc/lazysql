package viewer

import (
	"fmt"

	adapters "app.lazygit/internal/adapters"
	utils "app.lazygit/internal/utils"
	tea "github.com/charmbracelet/bubbletea"
)

type ViewerModel struct {
	database    adapters.Database
	selectedRow int
	table       utils.Table
	layout      utils.ConnectionContainerLayout
	isActive    bool
	content     string
}

func InitViewer(database adapters.Database, layout utils.ConnectionContainerLayout) ViewerModel {
	return ViewerModel{
		database: database,
		layout:   layout,
		isActive: false,
		table:    utils.InitTable([][]string{}, layout.ViewerWidth, layout.ViewerHeight),
		content:  "",
	}
}

func createTableFromData(data [][]string, layout utils.ConnectionContainerLayout) utils.Table {
	return utils.InitTable(data, layout.ViewerWidth-2, layout.ViewerHeight-2)
}

func (m ViewerModel) Init() tea.Cmd { return nil }

func (m ViewerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var viewPortCmd tea.Cmd
	switch msg := msg.(type) {
	case utils.ViewerStringData:
		m.content = string(msg)
		m.table = utils.InitTable([][]string{}, m.layout.ViewerWidth-2, m.layout.ViewerHeight-2)
	case utils.ViewerTableData:
		m.table = createTableFromData(msg, m.layout)
		m.content = ""
	case utils.ActiveViewChanged:
		m.isActive = string(msg) == "viewer"
	case utils.LayoutUpdated:
		m.layout = utils.ConnectionContainerLayout(msg)
	}
	m.table, viewPortCmd = m.table.Update(msg)
	return m, viewPortCmd
}

func (m ViewerModel) View() string {
	var content string
	if m.table.HasData() {
		content = fmt.Sprintf("%s\n", m.table.View())
	} else {
		content = m.content
	}
	return utils.RenderPanel("3 Viewer", content, m.layout.ViewerWidth, m.layout.ViewerHeight, m.isActive)
}
