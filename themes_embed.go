package main

import "embed"

// Bundled color scheme presets, baked directly into the binary (not read
// from disk at runtime) so a single `cp` of the compiled binary is still
// everything needed to run - no separate "also copy the themes folder"
// install step. See bin/themes/*.json for the actual preset values, and
// internal/utils/theme.go for how these get parsed/cycled.
//
//go:embed bin/themes/*.json
var embeddedThemesFS embed.FS
