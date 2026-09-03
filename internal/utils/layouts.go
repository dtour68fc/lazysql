package utils

import (
	"slices"
)

var CONNECTION_MANAGER_MIN_WIDTH = 84
var CONNECTION_MANAGER_MIN_HEIGHT = 32

func CalculateConnectionManagerLayout(width int, height int) ConnectionManagerLayout {
	headerHeight := 3
	footerHeight := 3

	widths := []int{CONNECTION_MANAGER_MIN_WIDTH, width / 3}
	// Fill the full available height (previously height/3, which left the
	// panel much shorter than the terminal with empty space above/below it
	// once AppModel started composing it alongside the Editor/Viewer
	// placeholders - it should vertically fill the window like they do.
	heights := []int{CONNECTION_MANAGER_MIN_HEIGHT, height}
	winWidth := slices.Max(widths)
	winHeight := slices.Max(heights)
	// The connection form used to permanently occupy 2/3 of the panel
	// width next to the list. It's now a centered modal popup (like
	// LazyCurl's New Project modal) instead, so the list gets the full
	// width and the modal gets its own fixed, more compact size.
	listWidth := winWidth
	modalWidth := winWidth / 2
	if modalWidth > 70 {
		modalWidth = 70
	}
	if modalWidth < 50 {
		modalWidth = 50
	}

	return ConnectionManagerLayout{
		ScreenWidth:         width,
		ScreenHeight:        height,
		WinWidth:            winWidth,
		WinHeight:           winHeight,
		HeaderHeight:        headerHeight,
		BodyHeight:          winHeight - (headerHeight + footerHeight),
		ConnectionListWidth: listWidth,
		ModalWidth:          modalWidth,
		FooterHeight:        footerHeight,
		HelpWidth:           winWidth / 2,
		HelpHeight:          winHeight + 4,
	}
}

func CalculateConnectionContainerLayout(width int, height int) ConnectionContainerLayout {
	footerHeight := 2

	// No more Explorer pane - Editor/Viewer take the full width, same
	// 25%/75% height split as the pre-connect placeholder panels.
	editorWidth := width
	viewerWidth := width

	bodyHeight := height - footerHeight
	editorHeight := bodyHeight * 25 / 100
	viewerHeight := bodyHeight - editorHeight

	return ConnectionContainerLayout{
		ScreenWidth:  width,
		ScreenHeight: height,
		EditorWidth:  editorWidth,
		EditorHeight: editorHeight,
		ViewerWidth:  viewerWidth,
		ViewerHeight: viewerHeight,
		FooterHeight: footerHeight,
	}
}
