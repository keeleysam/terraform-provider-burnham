Blends two colors at a given ratio and returns the result as hex. An `amount` of 0 returns the first color, 1 returns the second, and 0.5 is the midpoint; values outside [0, 1] are clamped.

Mixing happens in OKLCh by default, which interpolates hue and lightness perceptually and avoids the muddy grey midpoints of naive sRGB blending. Choose another space with `{ space = "..." }`: one of `oklch`, `oklab`, `rgb`, `hsv`, `lab`, `hcl`. This mirrors the CSS `color-mix(in <space>, a, b)` function.

Alpha is interpolated alongside the color, so mixing a translucent color yields a proportionally translucent result.
