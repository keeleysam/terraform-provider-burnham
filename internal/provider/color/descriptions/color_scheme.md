Generates a color-harmony palette from a base color by rotating its hue in the perceptually-uniform OKLCh space while holding lightness and chroma fixed, so every color in the scheme reads at the same perceived weight rather than as a jarring mix. The base color always leads the returned list (canonicalized to hex), and any alpha on the base is preserved on every entry.

Pick a `scheme` from the classic color-wheel relationships:

- `complementary` -> the base plus its opposite (2 colors).
- `analogous` -> the base plus its two neighbors (3 colors).
- `triadic` -> three colors evenly spaced 120 degrees apart (3 colors).
- `split-complementary` -> the base plus the two colors adjacent to its complement (3 colors).
- `tetradic` -> a rectangle on the wheel (4 colors).
- `square` -> four colors evenly spaced 90 degrees apart (4 colors).

Tune the spread with the `angle` option (degrees, default 30): it widens or narrows the neighbors used by `analogous` and `split-complementary`, and is ignored by the fixed-geometry schemes.

It is fully deterministic: the same inputs always produce the same palette, with no randomness, so plan output never churns between runs. That makes it a stable source of brand-consistent accent colors, chart series, or theme variants derived from a single seed color.
