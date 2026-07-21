package image

import _ "embed"

// notoColorEmoji is the Noto Color Emoji COLRv1 (vector) build, pinned to a Noto
// release. resvg renders COLRv1 natively; the vector build is roughly half the
// size of the CBDT bitmap build and compresses far better. The .ttf is gitignored
// and fetched by `go generate` (see below), so it is not committed to this repo.
//
//go:generate sh -c "mkdir -p fonts && curl -fsSL -o fonts/Noto-COLRv1.ttf https://github.com/googlefonts/noto-emoji/raw/v2.051/fonts/Noto-COLRv1.ttf"
//go:embed fonts/Noto-COLRv1.ttf
var notoColorEmoji []byte
