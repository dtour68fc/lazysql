package main

import (
	"io/fs"

	"app.lazygit/internal/client"
	"app.lazygit/internal/session_manager"
	"app.lazygit/internal/utils"
)

func main() {
	manager := session_manager.InitSessionManager()
	defer manager.CurrentSession().Cleanup()

	// Bundled color scheme presets (default/adapta/tokyo-night), baked
	// into the binary via themes_embed.go - loaded before anything else
	// so the ctrl+t modal's preset selector has them ready from the
	// first keypress.
	if themesFS, err := fs.Sub(embeddedThemesFS, "bin/themes"); err == nil {
		utils.LoadBundledThemes(themesFS)
	}

	client.StartApp()
}
