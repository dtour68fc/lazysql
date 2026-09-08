package utils

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Border colors used to be fixed constants here (Lavender/Surface0,
// mirroring LazyCurl's look) - now customizable via CurrentTheme (see
// theme.go), defaulting to those exact same values (DefaultTheme).

func Border() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder())
}

func BottomBorder() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, false, true, false)
}

func TopBorder() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), true, false, false, false)
}

func RightBorder() lipgloss.Style {
	return lipgloss.NewStyle().
		Border(lipgloss.NormalBorder(), false, true, false, false)
}

func FocusedTextInputStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Width(30).
		Border(lipgloss.RoundedBorder()).
		Padding(0, 1)
}

func TextInputStyle() lipgloss.Style {
	return lipgloss.NewStyle().
		Width(30).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(0, 1)
}

// RenderPanel draws a rounded-corner box with the title embedded directly in
// the top border, e.g.:
//
//	╭─ 1 Explorer ──────────────╮
//	│ ...content...             │
//	╰────────────────────────────╯
//
// matching LazyCurl's panel style exactly (see internal/ui/model.go's
// renderPanel there), so the two apps look the same at a glance and the
// numbered titles line up with the 1/2/3 window-jump keybindings. Border
// colors come from CurrentTheme (see theme.go) - customizable via ctrl+t.
func RenderPanel(title, content string, width, height int, active bool) string {
	if width < 4 {
		width = 4
	}
	if height < 2 {
		height = 2
	}

	var borderColor lipgloss.Color
	var titleFg lipgloss.Color
	if active {
		borderColor = Color(CurrentTheme.BorderActive)
		titleFg = Color(CurrentTheme.BorderActive)
	} else {
		borderColor = Color(CurrentTheme.BorderInactive)
		titleFg = lipgloss.Color("245")
	}

	titleText := " " + title + " "
	titleStyled := lipgloss.NewStyle().Foreground(titleFg).Bold(true).Render(titleText)

	innerWidth := width - 2
	titleWidth := lipgloss.Width(titleStyled)
	leftPadding := 1
	rightDashes := innerWidth - leftPadding - titleWidth
	if rightDashes < 0 {
		rightDashes = 0
	}

	borderChar := lipgloss.NewStyle().Foreground(borderColor)

	topBorder := borderChar.Render("╭") +
		borderChar.Render(strings.Repeat("─", leftPadding)) +
		titleStyled +
		borderChar.Render(strings.Repeat("─", rightDashes)) +
		borderChar.Render("╮")

	contentStyle := lipgloss.NewStyle().
		Width(width - 4).
		Height(height - 2)
	styledContent := contentStyle.Render(content)

	contentLines := strings.Split(styledContent, "\n")
	var borderedContent strings.Builder
	for i := 0; i < height-2; i++ {
		line := ""
		if i < len(contentLines) {
			line = contentLines[i]
		}
		lineWidth := lipgloss.Width(line)
		padding := width - 4 - lineWidth
		if padding < 0 {
			padding = 0
		}
		borderedContent.WriteString(borderChar.Render("│") + " " + line + strings.Repeat(" ", padding) + " " + borderChar.Render("│") + "\n")
	}

	bottomBorder := borderChar.Render("╰") +
		borderChar.Render(strings.Repeat("─", width-2)) +
		borderChar.Render("╯")

	return topBorder + "\n" + borderedContent.String() + bottomBorder
}
