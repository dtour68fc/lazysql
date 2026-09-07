package vimtea

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCursorBasics(t *testing.T) {
	cursor := newCursor(5, 10)

	assert.Equal(t, 5, cursor.Row, "Cursor row should be 5")
	assert.Equal(t, 10, cursor.Col, "Cursor column should be 10")

	cursorCopy := cursor.Clone()
	assert.Equal(t, cursor.Row, cursorCopy.Row, "Cloned cursor's row should match original")
	assert.Equal(t, cursor.Col, cursorCopy.Col, "Cloned cursor's column should match original")
}

func TestCursorPosition(t *testing.T) {
	testContent := "Line 1\nLine 2\nLine 3"
	editor := NewEditor(WithContent(testContent))
	model := editor.(*editorModel)

	model.cursor = newCursor(1, 2)
	assert.Equal(t, 1, model.cursor.Row, "Cursor row should be 1")
	assert.Equal(t, 2, model.cursor.Col, "Cursor column should be 2")

	model.adjustCursorPosition()
	assert.GreaterOrEqual(t, model.cursor.Row, 0, "Cursor row should be within lower buffer bounds")
	assert.Less(t, model.cursor.Row, model.buffer.lineCount(), "Cursor row should be within upper buffer bounds")

	assert.GreaterOrEqual(t, model.cursor.Col, 0, "Cursor column should be within lower line bounds")
	assert.Less(t, model.cursor.Col, model.buffer.lineLength(model.cursor.Row), "Cursor column should be within upper line bounds")
}

func TestViewportCursorVisibility(t *testing.T) {
	testContent := ""
	for i := range 50 {
		testContent += "Line " + string(rune('0'+i%10)) + "\n"
	}

	editor := NewEditor(WithContent(testContent))
	model := editor.(*editorModel)

	model.width = 80
	model.height = 20
	model.viewport.Width = 80
	model.viewport.Height = 20

	model.cursor = newCursor(5, 0)
	model.viewport.YOffset = 0

	model.ensureCursorVisible()
	assert.Equal(t, 0, model.viewport.YOffset, "Viewport should not scroll when cursor is already visible")

	model.cursor = newCursor(30, 0)

	model.ensureCursorVisible()
	assert.GreaterOrEqual(t, model.cursor.Row, model.viewport.YOffset, "Cursor row should be within or after viewport start")
	assert.Less(t, model.cursor.Row, model.viewport.YOffset+model.height, "Cursor row should be within viewport end")
}

func TestCursorBoundaryConditions(t *testing.T) {
	testContent := "Line 1\nLine 2\nLine 3"
	editor := NewEditor(WithContent(testContent))
	model := editor.(*editorModel)

	// Test cursor past end of line
	model.cursor = newCursor(0, 20)
	model.adjustCursorPosition()

	assert.Equal(t, 5, model.cursor.Col, "Cursor column should be adjusted to line length-1 (5)")

	// Test cursor past last line
	model.cursor = newCursor(10, 0)
	model.adjustCursorPosition()

	assert.Equal(t, 2, model.cursor.Row, "Cursor row should be adjusted to last line (2)")

	// Test cursor at negative positions
	model.cursor = newCursor(-1, -5)
	model.adjustCursorPosition()

	assert.Equal(t, 0, model.cursor.Row, "Cursor row should be adjusted to 0 when negative")
	assert.Equal(t, 0, model.cursor.Col, "Cursor column should be adjusted to 0 when negative")

	// Test cursor on empty line
	model.buffer.insertLine(3, "")
	model.cursor = newCursor(3, 0)
	model.adjustCursorPosition()

	assert.Equal(t, 3, model.cursor.Row, "Cursor row should remain at 3 for empty line")
	assert.Equal(t, 0, model.cursor.Col, "Cursor column should be 0 for empty line")
}

func TestCursorBasicOperations(t *testing.T) {
	c1 := newCursor(5, 10)
	c2 := newCursor(5, 10)

	// Test equality
	assert.Equal(t, c1.Row, c2.Row, "Cursors with same position should have equal rows")
	assert.Equal(t, c1.Col, c2.Col, "Cursors with same position should have equal columns")

	c2 = newCursor(5, 11)
	assert.Equal(t, c1.Row, c2.Row, "Cursors should have equal rows")
	assert.NotEqual(t, c1.Col, c2.Col, "Cursors with different columns should not be equal")

	c2 = newCursor(6, 10)
	assert.NotEqual(t, c1.Row, c2.Row, "Cursors with different rows should not be equal")
	assert.Equal(t, c1.Col, c2.Col, "Cursors should have equal columns")

	// Test Clone method
	c1 = newCursor(5, 10)
	c2 = c1.Clone()

	assert.Equal(t, c1.Row, c2.Row, "Cloned cursor should have same row")
	assert.Equal(t, c1.Col, c2.Col, "Cloned cursor should have same column")

	// Test cursor position comparison
	c1 = newCursor(5, 10)
	c2 = newCursor(8, 3)

	assert.Less(t, c1.Row, c2.Row, "Cursor (5,10) should have lower row than (8,3)")

	c1 = newCursor(5, 10)
	c2 = newCursor(5, 15)

	assert.Less(t, c1.Col, c2.Col, "Cursor (5,10) should have lower column than (5,15)")

	// Test manual cursor ordering
	c1 = newCursor(5, 10)
	c2 = newCursor(8, 15)

	var start, end Cursor
	if c1.Row < c2.Row || (c1.Row == c2.Row && c1.Col < c2.Col) {
		start, end = c1, c2
	} else {
		start, end = c2, c1
	}

	assert.Equal(t, c1.Row, start.Row, "Start cursor should have row 5")
	assert.Equal(t, c1.Col, start.Col, "Start cursor should have column 10")
	assert.Equal(t, c2.Row, end.Row, "End cursor should have row 8")
	assert.Equal(t, c2.Col, end.Col, "End cursor should have column 15")
}
