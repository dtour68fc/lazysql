package vimtea

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEditorBasics(t *testing.T) {
	testContent := "Line 1\nLine 2\nLine 3"
	editor := NewEditor(WithContent(testContent))

	assert.Equal(t, ModeNormal, editor.GetMode(), "Initial mode should be Normal")

	buffer := editor.GetBuffer()
	assert.Equal(t, testContent, buffer.Text(), "Buffer content should match initial content")
	assert.Equal(t, 3, buffer.LineCount(), "Buffer should have 3 lines")
}

func TestEditorModes(t *testing.T) {
	editor := NewEditor()

	editor.SetMode(ModeInsert)
	assert.Equal(t, ModeInsert, editor.GetMode(), "Mode should be Insert")

	editor.SetMode(ModeVisual)
	assert.Equal(t, ModeVisual, editor.GetMode(), "Mode should be Visual")

	editor.SetMode(ModeCommand)
	assert.Equal(t, ModeCommand, editor.GetMode(), "Mode should be Command")

	editor.SetMode(ModeNormal)
	assert.Equal(t, ModeNormal, editor.GetMode(), "Mode should be Normal")
}

func TestEditorKeypressHandling(t *testing.T) {
	editor := NewEditor()
	model := editor.(*editorModel)

	keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}}
	updated, _ := model.handleKeypress(keyMsg)
	model = updated.(*editorModel)

	assert.Equal(t, ModeInsert, model.mode, "After pressing 'i' in normal mode, should be in insert mode")

	model.mode = ModeNormal

	model.mode = ModeInsert
	keyMsg = tea.KeyMsg{Type: tea.KeyEsc}
	updated2, _ := model.handleKeypress(keyMsg)
	model = updated2.(*editorModel)

	assert.Equal(t, ModeNormal, model.mode, "After pressing Escape in insert mode, should be in normal mode")
}

func TestEditorCursorMovement(t *testing.T) {
	testContent := "Line 1\nLine 2\nLine 3"
	editor := NewEditor(WithContent(testContent))
	model := editor.(*editorModel)

	assert.Equal(t, 0, model.cursor.Row, "Initial cursor row should be 0")
	assert.Equal(t, 0, model.cursor.Col, "Initial cursor column should be 0")

	downBinding := model.registry.FindExact("j", ModeNormal)
	require.NotNil(t, downBinding, "Binding for 'j' should exist")

	downBinding.Command(model)

	assert.Equal(t, 1, model.cursor.Row, "After 'j', cursor row should be 1")
	assert.Equal(t, 0, model.cursor.Col, "After 'j', cursor column should remain 0")

	upBinding := model.registry.FindExact("k", ModeNormal)
	require.NotNil(t, upBinding, "Binding for 'k' should exist")

	upBinding.Command(model)

	assert.Equal(t, 0, model.cursor.Row, "After 'k', cursor row should be back to 0")
	assert.Equal(t, 0, model.cursor.Col, "After 'k', cursor column should remain 0")

	rightBinding := model.registry.FindExact("l", ModeNormal)
	require.NotNil(t, rightBinding, "Binding for 'l' should exist")

	rightBinding.Command(model)

	assert.Equal(t, 0, model.cursor.Row, "After 'l', cursor row should remain 0")
	assert.Equal(t, 1, model.cursor.Col, "After 'l', cursor column should be 1")

	leftBinding := model.registry.FindExact("h", ModeNormal)
	require.NotNil(t, leftBinding, "Binding for 'h' should exist")

	leftBinding.Command(model)

	assert.Equal(t, 0, model.cursor.Row, "After 'h', cursor row should remain 0")
	assert.Equal(t, 0, model.cursor.Col, "After 'h', cursor column should be back to 0")
}

func TestEditorInsertDelete(t *testing.T) {
	editor := NewEditor()
	model := editor.(*editorModel)
	buffer := editor.GetBuffer()

	editor.SetMode(ModeInsert)

	for _, ch := range "Hello" {
		keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}}
		updated, _ := model.handleKeypress(keyMsg)
		model = updated.(*editorModel)
	}

	assert.Equal(t, "Hello", buffer.Text(), "Buffer content should be 'Hello'")

	keyMsg := tea.KeyMsg{Type: tea.KeyBackspace}
	model.handleKeypress(keyMsg)

	assert.Equal(t, "Hell", buffer.Text(), "After deletion, buffer content should be 'Hell'")
}

func TestEditorUndoRedo(t *testing.T) {
	editor := NewEditor()
	model := editor.(*editorModel)
	buffer := editor.GetBuffer()

	editor.SetMode(ModeInsert)

	for _, ch := range "test undo" {
		keyMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}}
		updated, _ := model.handleKeypress(keyMsg)
		model = updated.(*editorModel)
	}

	editor.SetMode(ModeNormal)

	originalContent := buffer.Text()

	undoBinding := model.registry.FindExact("u", ModeNormal)
	require.NotNil(t, undoBinding, "Binding for 'u' (undo) should exist")

	model.buffer.undo(model.cursor)()

	assert.NotEqual(t, originalContent, buffer.Text(), "After undo, content should have changed")

	redoBinding := model.registry.FindExact("ctrl+r", ModeNormal)
	require.NotNil(t, redoBinding, "Binding for 'ctrl+r' (redo) should exist")

	model.buffer.redo(model.cursor)()

	assert.Equal(t, originalContent, buffer.Text(), "After redo, content should be back to original")
}

func TestEditorVisualMode(t *testing.T) {
	testContent := "Line 1\nLine 2\nLine 3"
	editor := NewEditor(WithContent(testContent))
	model := editor.(*editorModel)

	vBinding := model.registry.FindExact("v", ModeNormal)
	require.NotNil(t, vBinding, "Binding for 'v' should exist")

	vBinding.Command(model)

	assert.Equal(t, ModeVisual, model.mode, "After 'v', should be in visual mode")

	visualStart := model.visualStart
	assert.Equal(t, 0, visualStart.Row, "Visual selection start row should be 0")
	assert.Equal(t, 0, visualStart.Col, "Visual selection start column should be 0")

	jBinding := model.registry.FindExact("j", ModeVisual)
	require.NotNil(t, jBinding, "Binding for 'j' in visual mode should exist")

	jBinding.Command(model)

	start, end := model.GetSelectionBoundary()

	assert.Equal(t, 0, start.Row, "Selection start row should be 0")
	assert.Equal(t, 1, end.Row, "Selection end row should be 1")
}

func TestEditorStatusMessage(t *testing.T) {
	editor := NewEditor()
	model := editor.(*editorModel)

	testMsg := "Test status message"
	cmd := editor.SetStatusMessage(testMsg)

	cmd()

	assert.Equal(t, testMsg, model.statusMessage, "Status message should match set message")
}

func TestEditorCommandMode(t *testing.T) {
	editor := NewEditor()
	model := editor.(*editorModel)

	assert.True(t, model.enableCommandMode, "Command mode should be enabled by default")

	testCmdCalled := false
	editor.AddCommand("test", func(b Buffer, args []string) tea.Cmd {
		testCmdCalled = true
		return nil
	})

	colonBinding := model.registry.FindExact(":", ModeNormal)
	require.NotNil(t, colonBinding, "Binding for ':' should exist")

	colonBinding.Command(model)

	assert.Equal(t, ModeCommand, model.mode, "After ':', should be in command mode")

	model.commandBuffer = "test"

	updated, _ := model.Update(CommandMsg{Command: "test"})
	model = updated.(*editorModel)

	assert.True(t, testCmdCalled, "Command 'test' should have been called")
	assert.Equal(t, ModeNormal, model.mode, "After command execution, should return to normal mode")
}

func TestEditorMultipleBindings(t *testing.T) {
	editor := NewEditor()

	bindingCalled := false

	editor.AddBinding(KeyBinding{
		Key:         "ctrl+t",
		Mode:        ModeNormal,
		Description: "Test binding",
		Handler: func(b Buffer) tea.Cmd {
			bindingCalled = true
			return nil
		},
	})

	model := editor.(*editorModel)
	model.Init()

	keyMsg := tea.KeyMsg{
		Type: tea.KeyCtrlT,
	}

	model.handleKeypress(keyMsg)

	assert.True(t, bindingCalled, "Custom key binding ctrl+t should have been called")
}

func TestEditorCountPrefixCommands(t *testing.T) {
	testContent := "Line 1\nLine 2\nLine 3\nLine 4\nLine 5"
	editor := NewEditor(WithContent(testContent))
	model := editor.(*editorModel)

	model.countPrefix = 3

	jBinding := model.registry.FindExact("j", ModeNormal)
	require.NotNil(t, jBinding, "Binding for 'j' should exist")

	jBinding.Command(model)

	assert.Equal(t, 3, model.cursor.Row, "After '3j', cursor should be at row 3")
}

func TestEditorWindowResize(t *testing.T) {
	editor := NewEditor()
	model := editor.(*editorModel)

	assert.Equal(t, 0, model.width, "Initial window width should be 0")
	assert.Equal(t, 0, model.height, "Initial window height should be 0")

	newWidth, newHeight := 80, 24
	updated2, _ := model.SetSize(newWidth, newHeight)
	model = updated2.(*editorModel)

	assert.Equal(t, newWidth, model.width, "Window width should be updated to new width")

	expectedHeight := newHeight - 2 // Adjusted for status bar
	assert.Equal(t, expectedHeight, model.height, "Window height should be adjusted for status bar")
}

func TestEditorYankPaste(t *testing.T) {
	testContent := "Line 1\nLine 2\nLine 3"
	editor := NewEditor(WithContent(testContent))
	model := editor.(*editorModel)
	buffer := editor.GetBuffer()

	vBinding := model.registry.FindExact("v", ModeNormal)
	require.NotNil(t, vBinding, "Binding for 'v' should exist")
	vBinding.Command(model)

	for range 5 {
		rightBinding := model.registry.FindExact("l", ModeVisual)
		require.NotNil(t, rightBinding, "Binding for 'l' in visual mode should exist")
		rightBinding.Command(model)
	}

	yBinding := model.registry.FindExact("y", ModeVisual)
	require.NotNil(t, yBinding, "Binding for 'y' should exist")
	yBinding.Command(model)

	assert.Equal(t, ModeNormal, model.mode, "After yanking, should return to normal mode")
	assert.NotEmpty(t, model.yankBuffer, "Yank buffer should not be empty")

	model.cursor.Col = 0

	pBinding := model.registry.FindExact("p", ModeNormal)
	require.NotNil(t, pBinding, "Binding for 'p' should exist")
	pBinding.Command(model)

	assert.Contains(t, buffer.Text(), model.yankBuffer, "Buffer should contain yanked text after paste")
}

func TestEditorVisualHandler(t *testing.T) {
	editor := NewEditor(WithContent("Line 1\nLine 2\nLine 3"))
	model := editor.(*editorModel)

	var receivedText string
	editor.AddBinding(KeyBinding{
		Key:  "ctrl+t",
		Mode: ModeVisual,
		VisualHandler: func(text string) tea.Cmd {
			receivedText = text
			return nil
		},
	})

	model.registry.FindExact("v", ModeNormal).Command(model)
	for range 5 {
		model.registry.FindExact("l", ModeVisual).Command(model)
	}
	model.registry.FindExact("ctrl+t", ModeVisual).Command(model)

	assert.Equal(t, "Line 1", receivedText, "VisualHandler should receive the selected text")
}

func TestEditorVisualBindingHandlerOnly(t *testing.T) {
	editor := NewEditor(WithContent("Line 1\nLine 2"))
	model := editor.(*editorModel)

	handlerCalled := false
	editor.AddBinding(KeyBinding{
		Key:  "ctrl+t",
		Mode: ModeVisual,
		Handler: func(b Buffer) tea.Cmd {
			handlerCalled = true
			return nil
		},
	})

	model.registry.FindExact("v", ModeNormal).Command(model)
	model.registry.FindExact("ctrl+t", ModeVisual).Command(model)

	assert.True(t, handlerCalled, "Handler-only visual binding should still fire (backward compat)")
}

func MockCursorBlinkMsg() tea.Msg {
	return cursorBlinkMsg(time.Now())
}

func TestEditorCursorBlink(t *testing.T) {
	editor := NewEditor()
	model := editor.(*editorModel)

	initialBlink := model.cursorBlink

	model.lastBlinkTime = time.Now().Add(-2 * model.blinkInterval)

	updated2, _ := model.Update(MockCursorBlinkMsg())
	model = updated2.(*editorModel)

	assert.NotEqual(t, initialBlink, model.cursorBlink, "Cursor blink state should have toggled")
}
