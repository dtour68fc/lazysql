package explorer

import (
	"fmt"
	"github.com/charmbracelet/bubbles/viewport"
	"strings"

	adapters "app.lazygit/internal/adapters"
	utils "app.lazygit/internal/utils"
	tea "github.com/charmbracelet/bubbletea"
	lipgloss "github.com/charmbracelet/lipgloss"
)

type DatabaseError string
type DatabasesLoaded []string
type TablesLoaded []string
type TableLoaded []string

type ExplorerModel struct {
	database      adapters.Database
	databaseError string
	explorerList  ExplorerList
	layout        utils.ConnectionContainerLayout
	isActive      bool
	viewport      viewport.Model

	// preloadedDatabaseName/preloadedTables let the connected screen open
	// already-expanded to a database whose tables were already fetched
	// during the Connection Manager's Tables tab preview, instead of
	// starting fully collapsed and making the user re-navigate to
	// re-fetch what they just looked at a moment ago.
	preloadedDatabaseName string
	preloadedTables       []string
}

func InitExplorer(database adapters.Database, layout utils.ConnectionContainerLayout) ExplorerModel {
	return InitExplorerPreloaded(database, layout, "", nil)
}

// InitExplorerPreloaded is InitExplorer, plus an optional already-known
// database name and its tables (see ExplorerModel's preloaded* fields).
// Pass "" / nil when there's nothing preloaded (e.g. connecting via the
// "quick connect while editing" shortcut, which never previewed anything).
func InitExplorerPreloaded(database adapters.Database, layout utils.ConnectionContainerLayout, preloadedDatabaseName string, preloadedTables []string) ExplorerModel {
	list := ExplorerList{}
	viewport := viewport.New(layout.ExplorerWidth-4, layout.ExplorerHeight-4)
	list.Initialize()
	return ExplorerModel{
		database:              database,
		databaseError:         "",
		explorerList:          list,
		layout:                layout,
		isActive:              true,
		viewport:              viewport,
		preloadedDatabaseName: preloadedDatabaseName,
		preloadedTables:       preloadedTables,
	}
}

func (m ExplorerModel) loadDatabases() tea.Cmd {
	return func() tea.Msg {
		databases, err := m.database.GetDatabases()
		if err != nil {
			return DatabaseError(fmt.Sprintf("Failed to load databases: %v", err))
		}
		return DatabasesLoaded(databases)
	}
}

func (m ExplorerModel) expandSelectedNode() tea.Cmd {
	return func() tea.Msg {
		if m.explorerList.Selected.Type == "database" {
			database := m.explorerList.Selected.Title
			tables, err := m.database.GetTables(database)
			if err != nil {
				return DatabaseError(fmt.Sprintf("Failed to load tables for database %s: %v", database, err))
			}
			return TablesLoaded(tables)
		} else if m.explorerList.Selected.Type == "table" {
			return TableLoaded([]string{"data", "schema", "indexes"})
		} else if m.explorerList.Selected.Type == "table_item" {
			tableItem := m.explorerList.Selected.Title
			tableName := m.explorerList.Selected.Parent.Title
			database := m.explorerList.Selected.Parent.Parent.Title
			itemData, err := m.database.GetTableItem(database, tableName, tableItem)
			if err != nil {
				return DatabaseError(fmt.Sprintf("Failed to load data for table item %s: %v", tableItem, err))
			}
			return utils.ViewerTableData(itemData)
		} else {
			return nil
		}
	}
}

func (m *ExplorerModel) preloadActiveDatabase() {
	if m.preloadedDatabaseName == "" {
		return
	}
	for i := range m.explorerList.Root.Children {
		if m.explorerList.Root.Children[i].Title != m.preloadedDatabaseName {
			continue
		}
		m.explorerList.Selected = &m.explorerList.Root.Children[i]
		var nodes []ExplorerNode
		for _, table := range m.preloadedTables {
			nodes = append(nodes, ExplorerNode{Title: table, Type: "table"})
		}
		m.explorerList.Expand(nodes)
		return
	}
}

func (m ExplorerModel) createDatabaseList(databases []string) ExplorerList {
	var nodes []ExplorerNode
	for _, db := range databases {
		nodes = append(nodes, ExplorerNode{Title: db, Type: "database"})
	}
	m.explorerList.Expand(nodes)
	return m.explorerList
}

func (m ExplorerModel) Init() tea.Cmd {
	return m.loadDatabases()
}

func (m ExplorerModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	var viewportCmd tea.Cmd
	m, cmd = m.handleKeyboardActions(msg)
	switch msg := msg.(type) {
	case DatabaseError:
		m.databaseError = string(msg)
	case DatabasesLoaded:
		m.databaseError = ""
		m.explorerList = m.createDatabaseList([]string(msg))
		m.preloadActiveDatabase()
	case TablesLoaded:
		m.databaseError = ""
		var nodes []ExplorerNode
		for _, table := range msg {
			nodes = append(nodes, ExplorerNode{Title: table, Type: "table"})
		}
		m.explorerList.Expand(nodes)
	case TableLoaded:
		m.databaseError = ""
		var nodes []ExplorerNode
		for _, item := range msg {
			nodes = append(nodes, ExplorerNode{Title: item, Type: "table_item"})
		}
		m.explorerList.Expand(nodes)
	case utils.ActiveViewChanged:
		m.isActive = string(msg) == "explorer"
	case utils.LayoutUpdated:
		m.layout = utils.ConnectionContainerLayout(msg)
		m.viewport.Width = (m.layout.ExplorerWidth - 4)
		m.viewport.Height = (m.layout.ExplorerHeight - 4)
	}
	m.viewport.SetContent(m.ListNode(m.explorerList.Root, 0))
	m.viewport, viewportCmd = m.viewport.Update(msg)
	return m, tea.Batch(cmd, viewportCmd)
}

func (m ExplorerModel) handleKeyboardActions(msg tea.Msg) (ExplorerModel, tea.Cmd) {
	var cmd tea.Cmd
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "l":
			cmd = m.expandSelectedNode()
		case "h":
			if m.explorerList.Selected.Expanded {
				m.explorerList.Contract()
			} else {
				m.explorerList.ContractParent()
			}
		case "j":
			if m.explorerList.IsLastNodeSeleced() {
				break
			}
			m.explorerList.MoveDown()
		case "k":
			if m.explorerList.IsFirstNodeSelected() {
				break
			}
			m.explorerList.MoveUp()
		}
	}
	return m, cmd
}

func (m ExplorerModel) View() string {
	content := fmt.Sprintf("%s\n%s", m.viewport.View(), m.databaseError)
	return utils.RenderPanel("1 Explorer", content, m.layout.ExplorerWidth, m.layout.ExplorerHeight, m.isActive)
}

func (m ExplorerModel) ListNode(node *ExplorerNode, indent int) string {
	prefix := strings.Repeat("  ", indent)
	var newIndent int
	var result string
	var icon string
	var style lipgloss.Style

	if node.Type == "root" {
		result = ""
		newIndent = indent
	} else {
		if node.Expanded {
			icon = "[-] "
		} else if node.Type == "table_item" {
			icon = ""
		} else {
			icon = "[+] "
		}

		if m.explorerList.Selected == node {
			style = lipgloss.NewStyle().Foreground(lipgloss.Color("205")).Bold(true)
		} else {
			style = lipgloss.NewStyle()
		}
		title := style.Render(fmt.Sprintf("%s%s", icon, node.Title))
		result = fmt.Sprintf("%s%s\n", prefix, title)
		newIndent = indent + 1
	}
	if node.Expanded {
		for i := range node.Children {
			result += m.ListNode(&node.Children[i], newIndent)
		}
	}
	return result
}
