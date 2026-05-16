package utils

import (
	"slices"
)

var CONNECTION_MANAGER_MIN_WIDTH = 84
var CONNECTION_MANAGER_MIN_HEIGHT = 32
var EXPLORER_MIN_WIDTH = 20

func CalculateConnectionManagerLayout(width int, height int) ConnectionManagerLayout {
	headerHeight := 3
	footerHeight := 3

	widths := []int{CONNECTION_MANAGER_MIN_WIDTH, width / 3}
	heights := []int{CONNECTION_MANAGER_MIN_HEIGHT, height / 3}
	winWidth := slices.Max(widths)
	winHeight := slices.Max(heights)
	listWidth := winWidth / 3
	formWidth := winWidth - listWidth

	return ConnectionManagerLayout{
		ScreenWidth:         width,
		ScreenHeight:        height,
		WinWidth:            winWidth,
		WinHeight:           winHeight,
		HeaderHeight:        headerHeight,
		BodyHeight:          winHeight - (headerHeight + footerHeight),
		ConnectionListWidth: listWidth,
		ConnectionFormWidth: formWidth,
		FooterHeight:        footerHeight,
		HelpWidth:           winWidth / 2,
		HelpHeight:          winHeight + 4,
	}
}

func CalculateConnectionContainerLayout(width int, height int) ConnectionContainerLayout {
	footerHeight := 2
	explorerWidths := []int{EXPLORER_MIN_WIDTH, width / 4}
	explorerWidth := slices.Max(explorerWidths)

	editorWidth := width - explorerWidth
	viewerWidth := editorWidth

	bodyHeight := height - footerHeight
	explorerHeight := bodyHeight
	editorHeight := bodyHeight / 3
	viewerHeight := bodyHeight - editorHeight

	return ConnectionContainerLayout{
		ScreenWidth:    width,
		ScreenHeight:   height,
		ExplorerWidth:  explorerWidth,
		ExplorerHeight: explorerHeight,
		EditorWidth:    editorWidth,
		EditorHeight:   editorHeight,
		ViewerWidth:    viewerWidth,
		ViewerHeight:   viewerHeight,
		FooterHeight:   footerHeight,
	}
}
