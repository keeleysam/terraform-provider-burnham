Snaps an arbitrary color onto a fixed palette by returning the palette entry that is perceptually closest to it. The chosen entry is returned exactly as it was written in the palette, not canonicalized, so you get back the palette's own spelling (a name, a shorthand hex, whatever you supplied). On a tie the earlier entry wins.

This is the tool for constraining a color to a known set: map a computed or user-supplied color onto your brand palette, quantize to the terminal 256-color or ANSI-16 space, or bucket colors into a small legend. Because it returns the palette string verbatim, the result composes directly with the rest of your configuration.

Distance is measured with CIEDE2000 by default, the most accurate model of perceived color difference. Override it with the `metric` option:

- `ciede2000` (default) -> CIEDE2000 perceptual distance.
- `oklab` -> Euclidean distance in the OKLab space.
- `lab` -> Euclidean distance in CIE L*a*b*.
- `rgb` -> Euclidean distance in sRGB (fast, but least perceptually accurate).

It is fully deterministic: the same color, palette, and metric always yield the same result, so plan output never churns between runs.
