Nudges one or more OKLCh channels of a color and returns the result as hex. Because it works in OKLCh, a single function covers what would otherwise be a pile of separate operations: lighten and darken (the `lightness` channel), saturate and desaturate (`chroma`), rotate hue (`hue`), grayscale (`chroma` set to 0), and fade (`alpha`).

The `adjustments` object maps channel names (`lightness` 0-1, `chroma` >= 0, `hue` degrees, `alpha` 0-1) to values. A bare number sets the channel absolutely; a string applies an operation relative to the current value: `"+0.1"` adds, `"-0.1"` subtracts, `"*0.9"` scales down (a 10% darken when applied to lightness), `"/2"` halves. This is the chroma.js adjustment grammar.

Lightness and alpha are clamped to [0, 1], chroma to non-negative, and hue wraps modulo 360, so any adjustment yields a valid color. The headline use is deriving a full set of UI state colors (hover, active, disabled, focus ring) from one brand color in a generated theme.
