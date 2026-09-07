package vimtea

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSyntaxHighlighting(t *testing.T) {
	testLine := "func testFunction() {}"

	highlighter := newSyntaxHighlighter("monokai", "test.go")

	highlighted := highlighter.HighlightLine(testLine)

	assert.Contains(t, highlighted, "\033[", "Highlighting should apply ANSI color codes")

	assert.Contains(t, highlighted, "func", "Highlighting should preserve 'func' keyword")

	highlighter = newSyntaxHighlighter("monokai", "test.py")
	pythonLine := "def test_function():"
	highlightedPython := highlighter.HighlightLine(pythonLine)

	assert.Contains(t, highlightedPython, "def", "Python highlighting should preserve 'def' keyword")

	highlighter = newSyntaxHighlighter("monokai", "README")
	plainText := "This is plain text"
	highlightedPlain := highlighter.HighlightLine(plainText)

	assert.Equal(t, plainText, highlightedPlain, "Text with no recognized extension should be returned unchanged")
}

func TestHighlightCache(t *testing.T) {
	testLine := "var x = 10;"

	highlighter := newSyntaxHighlighter("monokai", "test.js")

	highlighted1 := highlighter.HighlightLine(testLine)

	highlighted2 := highlighter.HighlightLine(testLine)

	assert.Equal(t, highlighted1, highlighted2, "Second highlight call should return same result")

	testLine2 := "var x = 20;"
	highlighted3 := highlighter.HighlightLine(testLine2)

	assert.NotEqual(t, highlighted1, highlighted3, "Different lines should have different highlighting results")
}

func TestHighlightWithNoSyntax(t *testing.T) {
	testCode := "Plain text without syntax highlighting"

	highlighter := newSyntaxHighlighter("monokai", "")

	highlighted := highlighter.HighlightLine(testCode)

	assert.Equal(t, testCode, highlighted, "Text with empty filename should be returned unchanged")

	highlighter = newSyntaxHighlighter("monokai", "test.go")
	highlighter.enabled = false

	highlighted = highlighter.HighlightLine(testCode)

	assert.Equal(t, testCode, highlighted, "Text with disabled highlighting should be returned unchanged")
}

func TestYankHighlight(t *testing.T) {
	highlight := newYankHighlight()

	assert.False(t, highlight.Active, "New yank highlight should not be active")

	expectedDuration := 100 * time.Millisecond
	assert.Equal(t, expectedDuration, highlight.Duration, "Default yank highlight duration should be correct")

	editor := NewEditor(WithContent("Line 1\nLine 2\nLine 3"))
	model := editor.(*editorModel)

	model.mode = ModeVisual
	model.visualStart = newCursor(0, 0)
	model.cursor = newCursor(0, 6)

	binding := model.registry.FindExact("y", ModeVisual)
	require.NotNil(t, binding, "Visual mode yank binding should exist")

	binding.Command(model)

	assert.Contains(t, model.yankBuffer, "Line", "yankBuffer should contain the yanked text")

	assert.Equal(t, ModeNormal, model.mode, "Mode should be ModeNormal after yanking")
}
