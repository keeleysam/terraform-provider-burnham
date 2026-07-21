Rasterizes an SVG document to a PNG, returned as a base64 string (pair it with `base64decode` or a resource's `content_base64` to write the bytes).

Rendering is done by [resvg](https://github.com/linebender/resvg) compiled to WebAssembly and executed under a pure-Go runtime, so the provider stays CGO-free and the output is byte-identical across operating systems and CPU architectures: a plan on one machine and an apply on another produce the same PNG. It renders at near-browser fidelity, gradients, `clipPath`, masks, filters, patterns, text, and native color emoji, with no access to system fonts (fonts come only from the bundled set plus any you supply, which keeps the result deterministic and self-contained).

Options object keys:

- `width` / `height` (numbers, pixels): the output size. Supply one and the other is derived from the SVG's aspect ratio; supply both to force an exact box; supply neither to use the SVG's intrinsic size.
- `scale` (number): a multiplier over the intrinsic size, used when `width`/`height` are not set.
- `fonts` (list of strings): additional fonts as base64-encoded TTF/OTF, loaded alongside the bundled fonts. Use this to supply CJK, other scripts, or brand fonts. Each font's family name is read from the font itself, so an SVG `font-family` referencing it resolves.

Text with no matching font falls back to the bundled default, so labels always render; supply the right font when exact typography matters.
