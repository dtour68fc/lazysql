package conn_manager

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
