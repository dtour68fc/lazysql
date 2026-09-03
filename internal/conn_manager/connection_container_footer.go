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
	return fmt.Sprintf("%s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s, %s",
		"Select (enter/space)",
		"New project (shift+n)",
		"Edit (e)",
		"Delete (d)",
		"Reload (r)",
		"Dump to .sql (ctrl+d)",
		"Import .sql (ctrl+u)",
		"Save (s)",
		"Navigate (j,k)",
		"Back (esc/h)",
		"Switch Projects/Databases (shift+h/l)",
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
- Database (optional) only matters if you connect via Enter while
  editing (skipping the Projects->Databases flow entirely) - it sets
  which database your queries target from the start. Going through
  Projects->Databases always shows the full list and lets you pick.
- Picking a database (enter/space) lists its tables; picking a table
  opens the editor with "SELECT * FROM <table>" already run. esc/h
  backs out of the table list to the database list.
- Quit this dialog by hitting "?" or "esc"
	`
	return lipgloss.NewStyle().
		Width(width).
		Height(height).
		Padding(1).
		Border(lipgloss.NormalBorder()).
		Render(helpText)
}
