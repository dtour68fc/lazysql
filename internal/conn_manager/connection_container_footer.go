package conn_manager

import (
	"fmt"
	"github.com/charmbracelet/lipgloss"
)

func editFooter() string {
	return fmt.Sprintf("%s, %s, %s, %s",
		"Cancel (esc)",
		"Navigate (tab, shift+tab)",
		"Change Driver (h/l, ←/→ on Driver field)",
		"Change Mode (m, on Driver field)",
	)
}

func normalFooter() string {
	return fmt.Sprintf("%s, %s, %s, %s, %s, %s, %s, %s",
		"Load tables (enter/space)",
		"New project (shift+n)",
		"Edit (e)",
		"Save (s)",
		"Navigate (j,k)",
		"Switch Projects/Tables (shift+h/l)",
		"Change Mode (m)",
		"Help (?)",
	)
}

func errorFooter(errorMessage string) string {
	error_message := lipgloss.NewStyle().Foreground(lipgloss.Color("161")).Render(errorMessage)
	return fmt.Sprintf("%s\n%s", error_message, "Press 'e' to edit connection details.")
}

func connectingFooter() string {
	return "Connecting..."
}

func renderHelp(width int, height int) string {
	helpText := `Connection Manager Help
- Name is for connection name that will appear in the list
- Driver is used to establish and find the database server
 * Cycle between the supported drivers with h/l or left/right arrows
 * PostgreSQL (pgx) and MySQL (mysql) are currently supported
- Database (optional) picks which specific database the Tables tab and
  editor queries target - leave blank and it'll guess (the first one
  GetDatabases() returns, often just the empty admin db)
- Quit this dialog by hitting "?" or "esc"
	`
	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Padding(1).
		Border(lipgloss.NormalBorder()).
		Render(helpText)
}
