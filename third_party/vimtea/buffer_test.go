package vimtea

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBufferBasics(t *testing.T) {
	initialContent := "Line 1\nLine 2\nLine 3"
	buf := newBuffer(initialContent)

	assert.Equal(t, initialContent, buf.text(), "Buffer content should match initial content")
	assert.Equal(t, 3, buf.lineCount(), "Buffer should have 3 lines")

	assert.Equal(t, "Line 1", buf.Line(0), "Line 0 should be 'Line 1'")
	assert.Equal(t, "Line 2", buf.Line(1), "Line 1 should be 'Line 2'")
	assert.Equal(t, "Line 3", buf.Line(2), "Line 2 should be 'Line 3'")

	assert.Equal(t, 6, buf.lineLength(0), "Line 0 length should be 6")
}

func TestBufferInsertAt(t *testing.T) {
	buf := newBuffer("")

	buf.insertAt(0, 0, "Hello")
	assert.Equal(t, "Hello", buf.text(), "Buffer content should be 'Hello'")

	buf.insertAt(0, 5, " World")
	assert.Equal(t, "Hello World", buf.text(), "Buffer content should be 'Hello World'")

	buf.insertAt(0, 11, "\n")
	assert.Equal(t, "Hello World\n", buf.text(), "Buffer content should be 'Hello World\\n'")

	buf.insertAt(1, 0, "Line 2")
	assert.Equal(t, "Hello World\nLine 2", buf.text(), "Buffer content should be 'Hello World\\nLine 2'")

	assert.Equal(t, 2, buf.lineCount(), "Buffer should have 2 lines")
}

func TestBufferDeleteAt(t *testing.T) {
	buf := newBuffer("Hello World\nLine 2")

	buf.deleteAt(0, 5, 0, 5)
	assert.Equal(t, "HelloWorld\nLine 2", buf.text(), "Buffer content should be 'HelloWorld\\nLine 2'")

	buf.deleteRange(newCursor(0, 5), newCursor(0, 9))
	assert.Equal(t, "Hello\nLine 2", buf.text(), "Buffer content should be 'Hello\\nLine 2'")

	buf.deleteAt(0, 5, 1, 0)
	assert.Equal(t, "HelloLine 2", buf.text(), "Buffer content should be 'HelloLine 2'")

	assert.Equal(t, 1, buf.lineCount(), "Buffer should have 1 line")
}

func TestBufferUndoRedo(t *testing.T) {
	buf := newBuffer("Initial")
	cursor := newCursor(0, 0)

	initialState := buf.text()

	buf.saveUndoState(cursor)
	buf.insertAt(0, 7, " Content")
	modifiedState := buf.text()

	assert.Equal(t, "Initial Content", buf.text(), "Buffer content should be 'Initial Content'")

	undoMsg := buf.undo(cursor)().(UndoRedoMsg)
	assert.True(t, undoMsg.Success, "Undo should have succeeded")
	assert.Equal(t, initialState, buf.text(), "Buffer content after undo should match initial state")

	redoMsg := buf.redo(cursor)().(UndoRedoMsg)
	assert.True(t, redoMsg.Success, "Redo should have succeeded")
	assert.Equal(t, modifiedState, buf.text(), "Buffer content after redo should match modified state")
}

func TestBufferLineOperations(t *testing.T) {
	buf := newBuffer("Line 1\nLine 2\nLine 3")

	buf.insertLine(1, "New Line")
	expectedContent := "Line 1\nNew Line\nLine 2\nLine 3"
	assert.Equal(t, expectedContent, buf.text(), "Buffer content should match expected after insertLine")

	buf.deleteLine(2)
	expectedContent = "Line 1\nNew Line\nLine 3"
	assert.Equal(t, expectedContent, buf.text(), "Buffer content should match expected after deleteLine")

	assert.Equal(t, 3, buf.lineCount(), "Buffer should have 3 lines")
}

func TestBufferGetRange(t *testing.T) {
	buf := newBuffer("Hello world! This is a test.")

	result := buf.getRange(newCursor(0, 0), newCursor(0, 4))
	assert.Equal(t, "Hello", result, "getRange should return 'Hello'")

	buf = newBuffer("Line 1\nLine 2\nLine 3")
	result = buf.getRange(newCursor(0, 0), newCursor(1, 2))
	assert.Equal(t, "Line 1\nLin", result, "getRange should return 'Line 1\\nLin'")

	result = buf.getRange(newCursor(0, 0), newCursor(2, 3))
	assert.Equal(t, "Line 1\nLine 2\nLine", result, "getRange should return 'Line 1\\nLine 2\\nLine'")
}

func TestBufferDeleteRange(t *testing.T) {
	buf := newBuffer("Line 1\nLine 2\nLine 3")

	buf.deleteRange(newCursor(0, 5), newCursor(1, 2))
	expectedContent := "Line e 2\nLine 3"
	assert.Equal(t, expectedContent, buf.text(), "Buffer content should match expected after deleteRange")

	buf = newBuffer("Line 1\nLine 2\nLine 3\nLine 4")
	buf.deleteRange(newCursor(0, 3), newCursor(2, 3))
	expectedContent = "Lin 3\nLine 4"
	assert.Equal(t, expectedContent, buf.text(), "Buffer content should match expected after multi-line deleteRange")
}

func TestBufferTabRendering(t *testing.T) {
	buf := newBuffer("Line\twith\ttabs")

	// We now preserve tabs in the buffer
	assert.True(t, strings.Contains(buf.text(), "\t"), "Buffer content should contain literal tabs")

	// Check visual length calculation
	line := buf.Line(0)
	assert.Contains(t, line, "\t", "Line should contain tab characters")
	
	// Visual length with 4-space tabs should be greater than buffer length
	assert.Greater(t, buf.visualLineLength(0), len(line), "Visual line length should be greater than buffer line length")
}

func TestBufferReplaceContent(t *testing.T) {
	buf := newBuffer("Initial text")

	// Manually replace lines
	buf.lines = []string{"New content"}

	assert.Equal(t, "New content", buf.text(), "Buffer content should match replaced content")
	assert.Equal(t, 1, buf.lineCount(), "Buffer should have 1 line")

	// Test with multi-line content
	buf.lines = []string{"Line 1", "Line 2", "Line 3"}
	assert.Equal(t, 3, buf.lineCount(), "Buffer should have 3 lines")
	assert.Equal(t, "Line 2", buf.Line(1), "Line 1 should be 'Line 2'")
}

func TestBufferMultipleOperations(t *testing.T) {
	buf := newBuffer("Line 1\nLine 2\nLine 3")
	cursor := newCursor(0, 0)

	// Test multiple operations with undo
	buf.saveUndoState(cursor)
	buf.insertAt(0, 6, " modified")
	buf.insertAt(1, 6, " modified")

	expected := "Line 1 modified\nLine 2 modified\nLine 3"
	assert.Equal(t, expected, buf.text(), "Buffer content should match expected after multiple insertions")

	// Test undo for multiple operations
	undoMsg := buf.undo(cursor)().(UndoRedoMsg)
	assert.True(t, undoMsg.Success, "Undo should have succeeded")

	expected = "Line 1\nLine 2\nLine 3"
	assert.Equal(t, expected, buf.text(), "Buffer content after undo should match original")

	// Test replacing a line with setLine
	buf.setLine(1, "New Line 2")

	expected = "Line 1\nNew Line 2\nLine 3"
	assert.Equal(t, expected, buf.text(), "Buffer content should match expected after setLine")
}

func TestBufferClear(t *testing.T) {
	// Create a buffer with some content
	buf := newBuffer("Line 1\nLine 2\nLine 3")
	assert.Equal(t, 3, buf.lineCount(), "Buffer should have 3 lines initially")
	
	// Clear the buffer
	buf.clear()
	
	// After clearing, the buffer should have a single empty line
	assert.Equal(t, 1, buf.lineCount(), "Buffer should have 1 line after clear")
	assert.Equal(t, "", buf.Line(0), "The single line should be empty")
	assert.Equal(t, "", buf.text(), "Buffer text should be empty")
}
