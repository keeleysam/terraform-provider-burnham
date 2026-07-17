Interpolates `count` colors evenly across a list of stop colors and returns them as a list of hex strings. Two stops give a smooth gradient; more stops let you route the ramp through intermediate colors (a green - yellow - red threshold scale, for example). A single stop yields `count` copies of it.

Interpolation runs in OKLCh by default for perceptually even steps; choose another space with `{ space = "..." }` (`oklch`, `oklab`, `rgb`, `hsv`, `lab`, `hcl`). Alpha is interpolated alongside the color.

Typical uses are dashboard threshold ramps (Grafana, Datadog), the 50-950 lightness scale a design system needs from a single brand color, and gradient legends generated into a theme via `templatefile`.
