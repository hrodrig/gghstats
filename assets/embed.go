package assets

import "embed"

// FaviconsFS holds the tab icon (SVG) and Apple touch icon (PNG) embedded at build time.
//
//go:embed favicons/*
var FaviconsFS embed.FS
