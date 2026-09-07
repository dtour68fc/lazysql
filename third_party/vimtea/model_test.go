package vimtea

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestModelInitialization(t *testing.T) {
	// Test with default options
	editor := NewEditor()
	model := editor.(*editorModel)

	assert.Equal(t, ModeNormal, model.mode, "Initial mode should be Normal")
	assert.NotNil(t, model.buffer, "Buffer should be initialized")
	assert.NotNil(t, model.registry, "Binding registry should be initialized")
	assert.NotNil(t, model.commands, "Command registry should be initialized")

	// Test with content option
	testContent := "Test content"
	editor = NewEditor(WithContent(testContent))
	model = editor.(*editorModel)

	assert.Equal(t, testContent, model.buffer.text(), "Buffer content should be initialized with provided content")

	// Test with filename option
	editor = NewEditor(WithContent(""), WithFileName("test.go"))
	model = editor.(*editorModel)

	assert.NotNil(t, model.highlighter, "Syntax highlighter should be initialized")
}

func TestModelUpdate(t *testing.T) {
	editor := NewEditor(WithFullScreen())
	model := editor.(*editorModel)

	// Test key message handling
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}}
	updated, _ := model.Update(keyMsg)
	updatedModel := updated.(*editorModel)

	assert.Equal(t, ModeInsert, updatedModel.mode, "After pressing 'i', mode should be Insert")

	// Test window resize message
	sizeMsg := tea.WindowSizeMsg{Width: 100, Height: 50}
	updated, _ = model.Update(sizeMsg)
	updatedModel = updated.(*editorModel)

	assert.Equal(t, 100, updatedModel.width, "Window width should be updated correctly")
	assert.Equal(t, 48, updatedModel.height, "Window height should be adjusted for status bar") // height is adjusted for status bar

	// Test setting status message
	statusCmd := model.SetStatusMessage("Test status")
	statusCmd()

	assert.Equal(t, "Test status", model.statusMessage, "Status message should be updated correctly")
}

func TestModelKeySequences(t *testing.T) {
	editor := NewEditor(WithContent("Line 1\nLine 2\nLine 3"))
	model := editor.(*editorModel)

	// Get the dd (delete line) binding
	binding := model.registry.FindExact("dd", ModeNormal)
	require.NotNil(t, binding, "Built-in binding for 'dd' should exist")

	// First 'd' key press
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}}
	updated, _ := model.Update(keyMsg)
	updatedModel := updated.(*editorModel)

	// Should be collecting key sequence
	assert.Len(t, updatedModel.keySequence, 1, "Key sequence should have 1 key after first 'd'")
	assert.Equal(t, "d", updatedModel.keySequence[0], "First key in sequence should be 'd'")

	// Second 'd' key press
	updated, _ = updatedModel.Update(keyMsg)
	updatedModel = updated.(*editorModel)

	// Sequence should be executed and cleared
	assert.Empty(t, updatedModel.keySequence, "Key sequence should be cleared after command execution")

	// Buffer should be updated (line deleted)
	assert.Equal(t, 2, updatedModel.buffer.lineCount(), "Buffer should have 2 lines after deletion")

	// First line should now be what was previously the second line
	assert.Equal(t, "Line 2", updatedModel.buffer.Line(0), "After deleting first line, new first line should be 'Line 2'")
}

func TestModelCountPrefix(t *testing.T) {
	editor := NewEditor(WithContent("Line 1\nLine 2\nLine 3\nLine 4\nLine 5"))
	model := editor.(*editorModel)

	// Press '3'
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}}
	updated, _ := model.Update(keyMsg)
	updatedModel := updated.(*editorModel)

	assert.Equal(t, 3, updatedModel.countPrefix, "Count prefix should be 3")

	// Press 'j' to move down 3 lines
	keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	updated, _ = updatedModel.Update(keyMsg)
	updatedModel = updated.(*editorModel)

	assert.Equal(t, 3, updatedModel.cursor.Row, "Cursor should move down 3 lines to row 3")

	// Count prefix should be reset
	assert.Equal(t, 1, updatedModel.countPrefix, "Count prefix should be reset after use")

	// Test multi-digit count
	keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}}
	updated, _ = updatedModel.Update(keyMsg)
	updatedModel = updated.(*editorModel)

	keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}}
	updated, _ = updatedModel.Update(keyMsg)
	updatedModel = updated.(*editorModel)

	assert.Equal(t, 12, updatedModel.countPrefix, "Count prefix should be 12")
}

func TestModelCommandMode(t *testing.T) {
	editor := NewEditor()
	model := editor.(*editorModel)

	// Enter command mode
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{':'}}
	updated, _ := model.Update(keyMsg)
	updatedModel := updated.(*editorModel)

	assert.Equal(t, ModeCommand, updatedModel.mode, "Mode should be Command after pressing ':'")

	// Type command
	for _, ch := range "test" {
		keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}}
		updated, _ = updatedModel.Update(keyMsg)
		updatedModel = updated.(*editorModel)
	}

	assert.Equal(t, "test", updatedModel.commandBuffer, "Command buffer should contain 'test'")

	// Register test command
	commandCalled := false
	editor.AddCommand("test", func(b Buffer, args []string) tea.Cmd {
		commandCalled = true
		return nil
	})

	// Execute command with Enter
	keyMsg = tea.KeyMsg{Type: tea.KeyEnter}
	updated, cmd := updatedModel.Update(keyMsg)
	for cmd != nil {
		updated, cmd = updatedModel.Update(cmd())
	}
	updatedModel = updated.(*editorModel)

	assert.True(t, commandCalled, "Command should have been called")
	assert.Equal(t, ModeNormal, updatedModel.mode, "Mode should return to Normal after command execution")

	// Test command backspace
	updatedModel.mode = ModeCommand
	updatedModel.commandBuffer = "test"

	keyMsg = tea.KeyMsg{Type: tea.KeyBackspace}
	updated, _ = updatedModel.Update(keyMsg)
	updatedModel = updated.(*editorModel)

	assert.Equal(t, "tes", updatedModel.commandBuffer, "Command buffer should be 'tes' after backspace")
}

func TestModelVisualMode(t *testing.T) {
	editor := NewEditor(WithContent("Line 1\nLine 2\nLine 3"))
	model := editor.(*editorModel)

	// Enter visual mode
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'v'}}
	updated, _ := model.Update(keyMsg)
	updatedModel := updated.(*editorModel)

	assert.Equal(t, ModeVisual, updatedModel.mode, "Mode should be Visual after pressing 'v'")
	assert.Equal(t, 0, updatedModel.visualStart.Row, "Visual start row should be 0")
	assert.Equal(t, 0, updatedModel.visualStart.Col, "Visual start column should be 0")

	// Move cursor to create selection
	keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	updated, _ = updatedModel.Update(keyMsg)
	updatedModel = updated.(*editorModel)

	// Check selection boundaries
	start, end := updatedModel.GetSelectionBoundary()

	assert.Equal(t, 0, start.Row, "Selection start row should be 0")
	assert.Equal(t, 1, end.Row, "Selection end row should be 1")

	// Test yank in visual mode
	keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}}
	updated, _ = updatedModel.Update(keyMsg)
	updatedModel = updated.(*editorModel)

	assert.Equal(t, ModeNormal, updatedModel.mode, "Mode should return to Normal after yanking")
	assert.Contains(t, updatedModel.yankBuffer, "Line 1", "Yank buffer should contain 'Line 1'")
}

func TestModelInsertMode(t *testing.T) {
	editor := NewEditor(WithContent("Line 1"))
	model := editor.(*editorModel)

	// Enter insert mode
	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}}
	updated, _ := model.Update(keyMsg)
	updatedModel := updated.(*editorModel)

	assert.Equal(t, ModeInsert, updatedModel.mode, "Mode should be Insert after pressing 'i'")
	model.cursor = newCursor(0, 6)

	// Type some text
	for _, ch := range " inserted" {
		keyMsg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}}
		updated, _ = updatedModel.Update(keyMsg)
		updatedModel = updated.(*editorModel)
	}

	expectedText := "Line 1 inserted"
	assert.Equal(t, expectedText, updatedModel.buffer.text(), "Buffer content should match expected after insertion")

	// Exit insert mode
	keyMsg = tea.KeyMsg{Type: tea.KeyEsc}
	updated, _ = updatedModel.Update(keyMsg)
	updatedModel = updated.(*editorModel)

	assert.Equal(t, ModeNormal, updatedModel.mode, "Mode should be Normal after pressing Escape")
}

func TestEditorOptions(t *testing.T) {
	// Test multiple options
	editor := NewEditor(
		WithContent("Test content"),
		WithFileName("test.go"),
		WithEnableStatusBar(false),
		WithBlinkInterval(200*time.Millisecond),
	)

	model := editor.(*editorModel)

	assert.Equal(t, "Test content", model.buffer.text(), "WithContent option should be applied correctly")
	assert.False(t, model.enableStatusBar, "WithEnableStatusBar(false) option should be applied correctly")
	assert.Equal(t, 200*time.Millisecond, model.blinkInterval, "WithBlinkInterval option should be applied correctly")

	// Test disabling command mode
	editor = NewEditor(WithEnableModeCommand(false))
	model = editor.(*editorModel)

	assert.False(t, model.enableCommandMode, "WithEnableModeCommand(false) option should be applied correctly")

	// Test enabling relative line numbers
	editor = NewEditor(WithRelativeNumbers(true))
	model = editor.(*editorModel)

	assert.True(t, model.relativeNumbers, "WithRelativeNumbers option should be applied correctly")
}
