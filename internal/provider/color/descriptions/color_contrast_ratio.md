Returns the [WCAG 2.x contrast ratio](https://www.w3.org/WAI/WCAG22/Understanding/contrast-minimum.html) between two colors, a number from 1 (identical luminance) to 21 (black on white).

The ratio is computed from the exact WCAG relative-luminance formula (sRGB gamma expansion with the Rec. 709 weights), so it matches what accessibility checkers report. Alpha is ignored: contrast is defined for opaque colors, so composite over a known background first if you need to account for transparency.

Use it to gate a plan on legibility, for example asserting that a brand foreground on a brand background clears the AA threshold of 4.5 (normal text) or 3 (large text / UI), or AAA's 7.
