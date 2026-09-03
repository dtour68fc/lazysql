package editor

import (
	adapters "app.lazygit/internal/adapters"
	utils "app.lazygit/internal/utils"
	tea "github.com/charmbracelet/bubbletea"
	lipgloss "github.com/charmbracelet/lipgloss"
	"github.com/kujtimiihoxha/vimtea"
)

// IsCapturingInput reports whether the underlying vim editor is in a mode
// that should swallow raw keystrokes (Insert/Command), meaning global
// window-navigation shortcuts (1/2/3, tab, shift+j/k) must NOT be
// intercepted so the user can still type digits/letters into their query.
func (m EditorModel) IsCapturingInput() bool {
	mode := m.editor.GetMode()
	return mode == vimtea.ModeInsert || mode == vimtea.ModeCommand
}

// IsInNormalMode reports whether the editor is in vim's Normal mode
// specifically (not Insert/Visual/Command) - used to gate the global
// esc/q "quit the app" shortcuts, which must never fire while esc or q
// still mean something to vim itself (leaving Insert/Command mode, or
// canceling a Visual selection).
func (m EditorModel) IsInNormalMode() bool {
	return m.editor.GetMode() == vimtea.ModeNormal
}

type EditorModel struct {
	database     adapters.Database
	layout       utils.ConnectionContainerLayout
	isActive     bool
	editor       vimtea.Editor
	runQuery     func(text string) tea.Cmd
	initialQuery string
	// currentTable is whatever table the query shorthand DSL should
	// default to when you don't type one explicitly (e.g. plain ":sa"
	// instead of ":sa users") - seeded from picking a table in the
	// Databases tab's drill-down, and updated any time a shorthand query
	// resolves a table (whether you typed it or it came from this same
	// default), so it tracks whatever you're actually looking at.
	currentTable string
}

// InitEditor builds the editor. initialQuery, when non-empty, seeds the
// buffer with that text (e.g. "SELECT * FROM users;" from picking a table
// in the Databases tab's drill-down) AND runs it immediately on Init() -
// same affordance as netrw/oil.nvim opening a file the moment you select
// it, instead of making you retype/re-trigger the query yourself.
// currentTable seeds the shorthand DSL's default table (see EditorModel).
func InitEditor(database adapters.Database, layout utils.ConnectionContainerLayout, initialQuery string, currentTable string) EditorModel {
	editorOpts := []vimtea.EditorOption{
		vimtea.WithEnableStatusBar(true),
		vimtea.WithSelectedStyle(lipgloss.NewStyle().Background(lipgloss.Color("240")).Foreground(lipgloss.Color("255"))),
	}
	if initialQuery != "" {
		editorOpts = append(editorOpts, vimtea.WithContent(initialQuery))
	}
	editor := vimtea.NewEditor(editorOpts...)
	runQuery := func(text string) tea.Cmd {
		return func() tea.Msg {
			rows, err := database.RunQuery(text)
			if err != nil {
				return utils.ViewerStringData(lipgloss.NewStyle().Foreground(lipgloss.Color("160")).Render(err.Error()))
			} else if len(rows) > 0 && len(rows[0]) > 0 {
				return utils.ViewerTableData(rows)
			} else {
				return utils.ViewerStringData(lipgloss.NewStyle().Foreground(lipgloss.Color("34")).Render("Query executed successfully"))
			}
		}
	}
	editor.AddBinding(vimtea.KeyBinding{
		Key:           "ctrl+r",
		Mode:          vimtea.ModeVisual,
		Description:   "Run the selected query",
		VisualHandler: func(text string) tea.Cmd { return runQuery(text) },
	})
	// ctrl+s runs the WHOLE query buffer, matching LazyCurl's "ctrl+s sends
	// the request" keybind - no need to visually select the text first like
	// ctrl+r requires.
	editor.AddBinding(vimtea.KeyBinding{
		Key:         "ctrl+s",
		Mode:        vimtea.ModeNormal,
		Description: "Run the whole query (like LazyCurl's send-request key)",
		Handler:     func(buf vimtea.Buffer) tea.Cmd { return runQuery(buf.Text()) },
	})
	return EditorModel{
		database:     database,
		layout:       layout,
		isActive:     false,
		editor:       editor,
		runQuery:     runQuery,
		initialQuery: initialQuery,
		currentTable: currentTable,
	}
}

func (m EditorModel) Init() tea.Cmd {
	newEditor, cmd := m.editor.SetSize(m.layout.EditorWidth-2, m.layout.EditorHeight-2)
	m.editor = newEditor.(vimtea.Editor)
	if m.initialQuery != "" {
		return tea.Batch(m.editor.Init(), cmd, m.runQuery(m.initialQuery))
	}
	return tea.Batch(m.editor.Init(), cmd)
}

func (m EditorModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd
	handledShorthand := false
	switch msg := msg.(type) {
	case utils.ActiveViewChanged:
		m.isActive = string(msg) == "editor"
	case utils.LayoutUpdated:
		m.layout = utils.ConnectionContainerLayout(msg)
		newEditor, sizeCmd := m.editor.SetSize(m.layout.EditorWidth-2, m.layout.EditorHeight-2)
		m.editor = newEditor.(vimtea.Editor)
		cmd = sizeCmd
	case vimtea.CommandMsg:
		// vimtea's own AddCommand registry only actually works for
		// zero-arg commands in real use - by the time its CommandMsg
		// makes it back through Update(), the buffer it re-reads to
		// parse args has already been cleared. So instead of fighting
		// that, we parse the raw ":..." text ourselves here, straight
		// off the message, for the query shorthand DSL (":sa users",
		// ":s(col1,col2) users", chained ":j other on a.x=b.y" joins).
		// Anything that doesn't match just falls through to vimtea's own
		// handling below (which will show "Unknown command" for genuine
		// typos, same as always).
		if query, autoRun, resolvedTable, ok := parseShorthandQuery(msg.Command, m.currentTable); ok {
			buf := m.editor.GetBuffer()
			buf.Clear()
			buf.InsertAt(0, 0, query)
			if resolvedTable != "" {
				m.currentTable = resolvedTable
			}
			if autoRun {
				cmd = m.runQuery(query)
			}
			handledShorthand = true
		}
	}
	newEditor, newCmd := m.editor.Update(msg)
	m.editor = newEditor.(vimtea.Editor)
	// vimtea's own CommandMsg handling above still needs to run regardless
	// (it's what resets the command buffer and drops back to Normal mode)
	// - but since it never finds our shorthand in ITS OWN command
	// registry, it always stamps "Unknown command" over the status bar
	// right after, even when we just successfully handled it ourselves.
	// Override that back once we know better. SetStatusMessage returns a
	// tea.Cmd that mutates synchronously the moment it's *called* - call
	// it directly here instead of batching it, so the override lands
	// immediately with no visible flicker of "Unknown command" first.
	if handledShorthand {
		m.editor.SetStatusMessage("")()
	}
	return m, tea.Batch(cmd, newCmd)
}

func (m EditorModel) View() string {
	return utils.RenderPanel("2 Editor", m.editor.View(), m.layout.EditorWidth, m.layout.EditorHeight, m.isActive)
}
