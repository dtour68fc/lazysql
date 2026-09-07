package vimtea

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestWrappedBuffer(t *testing.T) {
	editor := NewEditor(WithContent("Line 1\nLine 2\nLine 3"))
	model := editor.(*editorModel)
	wrapped := model.GetBuffer()

	assert.Equal(t, "Line 1\nLine 2\nLine 3", wrapped.Text(), "WrappedBuffer Text() should return underlying buffer content")
	assert.Equal(t, 3, wrapped.LineCount(), "WrappedBuffer LineCount() should be 3")

	lines := wrapped.Lines()
	assert.Equal(t, "Line 2", lines[1], "WrappedBuffer Lines()[1] should return 'Line 2'")
	assert.Equal(t, 6, wrapped.LineLength(2), "WrappedBuffer LineLength(2) should be 6")
}

func TestWrappedBufferModifications(t *testing.T) {
	editor := NewEditor(WithContent("Initial content"))
	model := editor.(*editorModel)
	wrapped := model.GetBuffer()

	wrapped.InsertAt(0, 7, " modified")
	assert.Equal(t, "Initial modified content", wrapped.Text(), "WrappedBuffer InsertAt should modify content correctly")

	wrapped.DeleteAt(0, 7, 0, 15)
	assert.Equal(t, "Initial content", wrapped.Text(), "WrappedBuffer DeleteAt should remove text correctly")

	// Test delete range through the Buffer interface
	model.buffer.insertLine(1, "New line")
	assert.Equal(t, 2, wrapped.LineCount(), "Buffer LineCount should be 2 after insertLine")

	lines := wrapped.Lines()
	assert.Equal(t, "New line", lines[1], "Lines()[1] should be 'New line'")

	model.buffer.deleteLine(1)
	assert.Equal(t, 1, wrapped.LineCount(), "Buffer LineCount should be 1 after deleteLine")
}

func TestWrappedBufferUndoRedo(t *testing.T) {
	editor := NewEditor(WithContent("First line\nSecond line\nThird line"))
	model := editor.(*editorModel)
	wrapped := model.GetBuffer()

	// Get range from underlying buffer
	selection := model.buffer.getRange(newCursor(0, 0), newCursor(1, 5))
	assert.Equal(t, "First line\nSecond", selection, "buffer.getRange should return correct text selection")

	// Test undo/redo through the wrapped buffer
	model.cursor = newCursor(0, 0)
	wrapped.InsertAt(0, 0, "Test ")

	assert.True(t, strings.HasPrefix(wrapped.Text(), "Test First"), "InsertAt should add text at beginning")

	// Undo the insertion
	cmd := wrapped.Undo()
	msg := cmd().(UndoRedoMsg)

	assert.True(t, msg.Success, "Undo should succeed")
	assert.True(t, strings.HasPrefix(wrapped.Text(), "First"), "Undo should restore original text")

	// Redo the insertion
	cmd = wrapped.Redo()
	msg = cmd().(UndoRedoMsg)

	assert.True(t, msg.Success, "Redo should succeed")
	assert.True(t, strings.HasPrefix(wrapped.Text(), "Test First"), "Redo should reapply changes")
}

func TestWrappedBufferCanUndoRedo(t *testing.T) {
	editor := NewEditor(WithContent("Initial state"))
	model := editor.(*editorModel)
	wrapped := model.GetBuffer()

	// Test CanUndo, CanRedo
	assert.False(t, wrapped.CanUndo(), "CanUndo should be false initially")

	// Make a change
	model.cursor = newCursor(0, 0)
	model.buffer.saveUndoState(model.cursor)
	wrapped.InsertAt(0, 0, "Test ")

	assert.True(t, wrapped.CanUndo(), "CanUndo should be true after making changes")
	assert.False(t, wrapped.CanRedo(), "CanRedo should be false before undoing")

	// Undo
	wrapped.Undo()()

	assert.True(t, wrapped.CanRedo(), "CanRedo should be true after undoing")

	// Redo
	wrapped.Redo()()

	assert.False(t, wrapped.CanRedo(), "CanRedo should be false after redoing")
	assert.True(t, wrapped.CanUndo(), "CanUndo should be true after redo")
}
