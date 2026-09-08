package utils

import (
	"encoding/json"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Theme holds every customizable color in the app - values are whatever
// lipgloss.Color accepts (ANSI 256 numbers as strings like "57", or hex
// like "#5f00ff"). Field order here also drives the order fields show up
// in in the ctrl+t color editor modal (see client/theme_modal.go).
type Theme struct {
	BorderActive   string // Active pane's panel border + title
	BorderInactive string // Inactive panes' panel border + title
	HoverRow       string // Viewer: hovered row's background (also used for list row selection in Projects/Databases/Tables)
	HoverColumn    string // Viewer: hovered column's background
	HoverCell      string // Viewer: exact hovered cell's background (row+column intersection)
	HoverFg        string // Foreground text color for all of the above
	Marked         string // Viewer: marked (not hovered) rows/columns background - "a" to mark
	MarkedFg       string // Foreground text color for marked rows/columns
	TextFg         string // Normal (non-highlighted) text color - empty means "leave it at the terminal's default", same as before this existed
	ErrorFg        string // Error messages (failed connections/queries/dumps/imports, etc)
	SuccessFg      string // Success messages (e.g. "Query executed successfully")
}

// DefaultTheme is exactly what every color already was before this became
// customizable - existing installs with no saved theme.json see no visual
// change at all.
func DefaultTheme() Theme {
	return Theme{
		BorderActive:   "141",
		BorderInactive: "240",
		HoverRow:       "57",
		HoverColumn:    "60",
		HoverCell:      "63",
		HoverFg:        "229",
		Marked:         "97",
		MarkedFg:       "255",
		TextFg:         "",
		ErrorFg:        "161",
		SuccessFg:      "34",
	}
}

// MaybeForeground applies a Foreground color to style only if value is
// non-empty - used for TextFg, where empty deliberately means "leave the
// terminal's own default text color alone" rather than forcing a color.
func MaybeForeground(style lipgloss.Style, value string) lipgloss.Style {
	if value == "" {
		return style
	}
	return style.Foreground(Color(value))
}

// ThemeFieldNames returns the display label for each Theme field, in the
// same order as the struct - used by the color editor modal to build one
// labeled input per field without repeating this list by hand.
func ThemeFieldNames() []string {
	return []string{
		"Border (active pane)",
		"Border (inactive panes)",
		"Hover row",
		"Hover column",
		"Hover cell",
		"Hover text",
		"Marked",
		"Marked text",
		"Normal text (blank = terminal default)",
		"Error text",
		"Success text",
	}
}

// ThemeFieldValues/ThemeFromFieldValues let the modal read/write all
// fields generically (as a slice, matching ThemeFieldNames' order) instead
// of hand-wiring each one.
func (t Theme) ThemeFieldValues() []string {
	return []string{
		t.BorderActive, t.BorderInactive, t.HoverRow, t.HoverColumn, t.HoverCell, t.HoverFg,
		t.Marked, t.MarkedFg, t.TextFg, t.ErrorFg, t.SuccessFg,
	}
}

func ThemeFromFieldValues(values []string) Theme {
	get := func(i int) string {
		if i < len(values) {
			return values[i]
		}
		return ""
	}
	return Theme{
		BorderActive:   get(0),
		BorderInactive: get(1),
		HoverRow:       get(2),
		HoverColumn:    get(3),
		HoverCell:      get(4),
		HoverFg:        get(5),
		Marked:         get(6),
		MarkedFg:       get(7),
		TextFg:         get(8),
		ErrorFg:        get(9),
		SuccessFg:      get(10),
	}
}


// CurrentTheme is what every color-consuming render function (RenderPanel,
// Table's styles, ConnectionList's selection styles) actually reads at
// render/construction time - changing it (via ApplyTheme) takes effect
// immediately for panel borders, and on next data load for the Viewer
// table (whose styles are fixed at InitTable() construction, not
// re-computed per render).
var CurrentTheme = DefaultTheme()

func ApplyTheme(t Theme) { CurrentTheme = t }

func getThemeFilePath() (string, error) {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "lazysql", "theme.json"), nil
	}
	userConfigDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(userConfigDir, "lazysql", "theme.json"), nil
}

// LoadTheme returns DefaultTheme() (not an error) if there's no saved
// theme.json yet - a brand new install shouldn't fail to start just
// because it's never customized colors before.
func LoadTheme() (Theme, error) {
	path, err := getThemeFilePath()
	if err != nil {
		return DefaultTheme(), err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultTheme(), nil
		}
		return DefaultTheme(), err
	}
	var t Theme
	if err := json.Unmarshal(data, &t); err != nil {
		return DefaultTheme(), err
	}
	return t, nil
}

func SaveTheme(t Theme) error {
	path, err := getThemeFilePath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), os.ModePerm); err != nil {
		return err
	}
	data, err := json.MarshalIndent(t, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

// Color is a tiny convenience wrapper so callers don't need to import
// lipgloss just to turn a Theme string field into a usable color.
func Color(value string) lipgloss.Color { return lipgloss.Color(value) }

// NamedTheme pairs a bundled preset's display name (its filename minus
// ".json") with its parsed Theme.
type NamedTheme struct {
	Name  string
	Theme Theme
}

// BundledThemes is the app's built-in color scheme presets (default,
// adapta, tokyo-night, ...) - baked into the binary via go:embed (see
// themes_embed.go at the repo root) and populated once at startup via
// LoadBundledThemes, before anything reads it. The ctrl+t modal's preset
// selector cycles through this list with h/l.
var BundledThemes []NamedTheme

// themeNamePriority pins the order presets show up in when cycling,
// regardless of alphabetical filename order - "default" first since it's
// what you land on with nothing customized, then whatever else in
// whatever order they were added. Anything not listed here just falls in
// afterward, alphabetically.
var themeNamePriority = []string{"default", "adapta", "tokyo-night"}

// LoadBundledThemes parses every *.json file in themesFS (an embedded
// directory of preset Theme values) into BundledThemes. Safe to call with
// an fs that doesn't exist or is empty - BundledThemes just stays empty
// and the preset selector has nothing to cycle through, rather than
// crashing the whole app over a packaging mistake.
func LoadBundledThemes(themesFS fs.FS) error {
	entries, err := fs.ReadDir(themesFS, ".")
	if err != nil {
		return err
	}
	var loaded []NamedTheme
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		data, err := fs.ReadFile(themesFS, entry.Name())
		if err != nil {
			continue
		}
		var t Theme
		if err := json.Unmarshal(data, &t); err != nil {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".json")
		loaded = append(loaded, NamedTheme{Name: name, Theme: t})
	}
	sort.SliceStable(loaded, func(i, j int) bool {
		pi, pj := themeNamePriorityIndex(loaded[i].Name), themeNamePriorityIndex(loaded[j].Name)
		if pi != pj {
			return pi < pj
		}
		return loaded[i].Name < loaded[j].Name
	})
	BundledThemes = loaded
	return nil
}

func themeNamePriorityIndex(name string) int {
	for i, n := range themeNamePriority {
		if n == name {
			return i
		}
	}
	return len(themeNamePriority) // unlisted names sort after all pinned ones
}
