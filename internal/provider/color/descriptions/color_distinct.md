Generates `count` visually distinct colors by sweeping the hue wheel at a fixed lightness and chroma in the perceptually-uniform OKLCh space, so the colors read as a coherent family of similar perceived weight rather than a jarring mix.

It is fully deterministic: the same `count` always produces the same colors, with no randomness, so plan output never churns between runs. That makes it safe for assigning a stable color per element when you `for_each` over N services, series, labels, or teams, the classic fix for dashboards that run out of distinct series colors.

Tune the look with options: `lightness` (OKLCh L, 0-1, default 0.72), `chroma` (OKLCh C, default 0.14), and `hue_offset` (degrees to rotate the starting hue, default 0). Lower the lightness for darker sets, raise the chroma for more saturated ones.
