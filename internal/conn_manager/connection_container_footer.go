package conn_manager

import (
	"fmt"
	"github.com/charmbracelet/lipgloss"
)

func editFooter() string {
	return fmt.Sprintf("%s, %s",
		"Cancel (esc)",
		"Navigate (tab, shift+tab)",
	)
}

func normalFooter() string {
	return fmt.Sprintf("%s, %s, %s, %s, %s, %s",
		"Connect (enter)",
		"Edit (e)",
		"Save (s)",
		"Navigate (j,k)",
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
- Driver is used to establish and find the database server, user
 * pgx for PostgreSQL
 * mysql for MySQL
- Quit this dialog by hitting "?" or "esc"
	`
	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Padding(1).
		Border(lipgloss.NormalBorder()).
		Render(helpText)
}
