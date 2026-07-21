package image

import _ "embed"

// The bundled Noto fonts are downloaded and checksum-verified by genfonts.go and
// are gitignored, so a fresh checkout must run `go generate ./...` before this
// package compiles. Web-safe named fonts (Arial/Times/Courier/...) are covered by
// the go-fonts Liberation modules plus an alias map, see svg_render.go.
//
//go:generate go run genfonts.go

// Noto text families, variable builds: one file covers the whole weight range via
// the wght axis, with italic as a separate file; resvg selects the weight/style.
// These back the CSS generic families sans-serif / serif / monospace.
//
//go:embed fonts/NotoSans.ttf
var notoSans []byte

//go:embed fonts/NotoSans-Italic.ttf
var notoSansItalic []byte

//go:embed fonts/NotoSerif.ttf
var notoSerif []byte

//go:embed fonts/NotoSerif-Italic.ttf
var notoSerifItalic []byte

//go:embed fonts/NotoSansMono.ttf
var notoSansMono []byte

// Noto Color Emoji COLRv1 (vector color emoji), rendered natively by resvg.
//
//go:embed fonts/Noto-COLRv1.ttf
var notoColorEmoji []byte
