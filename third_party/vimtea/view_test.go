package vimtea

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
)

func TestViewRenderBasics(t *testing.T) {
	editor := NewEditor(WithContent("Line 1\nLine 2\nLine 3\n"))
	model := editor.(*editorModel)

	// Set up viewport size
	model.width = 40
	model.height = 10
	model.viewport.Width = 40
	model.viewport.Height = 10
	model.cursor = newCursor(3, 0)

	// Render view
	view := model.View()

	// Basic content checks
	assert.Contains(t, view, "Line 1", "View should contain 'Line 1'")
	assert.Contains(t, view, "Line 2", "View should contain 'Line 2'")
	assert.Contains(t, view, "Line 3", "View should contain 'Line 3'")

	// Status line should be present
	assert.Contains(t, strings.ToLower(view), "normal", "View should contain mode indicator 'NORMAL'")
}

func TestViewLineNumbers(t *testing.T) {
	// Create editor with content
	editor := NewEditor(
		WithContent("Line A\nLine B\nLine C\nLine D\nLine E"),
	)
	model := editor.(*editorModel)

	// Set up viewport size
	model.width = 40
	model.height = 10
	model.viewport.Width = 40
	model.viewport.Height = 10

	// Render view
	view := model.View()

	// Check for line numbers
	assert.True(t,
		strings.Contains(view, "1") &&
			strings.Contains(view, "2") &&
			strings.Contains(view, "3"),
		"View should contain line numbers when enabled")

	// Test relative line numbers
	model.relativeNumbers = true
	model.cursor.Row = 2 // Set cursor to line 3

	view = model.View()
	lines := strings.Split(view, "\n")

	lineNumber := string(strings.TrimSpace(ansi.Strip(lines[2]))[0])
	// Check for relative line numbers (current line should be absolute)
	assert.Equal(t, lineNumber, "3", "Current line should show absolute line number 3")
}

func TestViewCommandBuffer(t *testing.T) {
	editor := NewEditor()
	model := editor.(*editorModel)

	// Set up command mode
	model.mode = ModeCommand
	model.commandBuffer = "test"

	// Set up viewport
	model.width = 40
	model.height = 10
	model.viewport.Width = 40
	model.viewport.Height = 10

	view := model.View()

	// Command should be shown in status area
	assert.Contains(t, view, ":test", "View should show command buffer in command mode")
}

func TestViewStatusMessages(t *testing.T) {
	editor := NewEditor()
	model := editor.(*editorModel)

	// Set status message
	model.statusMessage = "Test status message"

	// Set up viewport
	model.width = 40
	model.height = 10
	model.viewport.Width = 40
	model.viewport.Height = 10

	view := model.View()

	// Status message should be displayed
	assert.Contains(t, view, "Test status message", "View should show status message")
}

func TestViewSyntaxHighlighting(t *testing.T) {
	// Create Go code
	goCode := "package main\n\nfunc main() {\n\t// Comment\n\tfmt.Println(\"Hello\")\n}"

	editor := NewEditor(
		WithContent(goCode),
		WithFileName("test.go"),
	)
	model := editor.(*editorModel)

	// Set up viewport
	model.width = 40
	model.height = 10
	model.viewport.Width = 40
	model.viewport.Height = 10

	view := model.View()

	// Syntax highlighting should add ANSI codes
	assert.Contains(t, view, "\033[", "View should contain ANSI codes for syntax highlighting")
}

func TestViewLongContent(t *testing.T) {
	// Create content with many lines
	var content strings.Builder
	for i := 1; i <= 100; i++ {
		content.WriteString("Line ")
		content.WriteString(string(rune('0' + i%10)))
		content.WriteString("\n")
	}

	editor := NewEditor(WithContent(content.String()))
	model := editor.(*editorModel)

	// Set up viewport with limited height
	model.width = 40
	model.height = 10
	model.viewport.Width = 40
	model.viewport.Height = 10

	// Position cursor far down
	model.cursor = newCursor(50, 0)
	model.ensureCursorVisible()

	view := model.View()

	// View should contain content near cursor position
	assert.Contains(t, view, "Line 0", "View should contain visible content near cursor")

	// First lines should not be visible
	assert.NotContains(t, view, "Line 1\nLine 2", "View should not contain content from beginning when scrolled down")
}
