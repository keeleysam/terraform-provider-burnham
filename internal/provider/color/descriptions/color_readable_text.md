Returns the most legible text color to place on a given background, chosen by [WCAG 2.x contrast](https://www.w3.org/WAI/WCAG22/Understanding/contrast-minimum.html). By default it picks whichever of black (`#000000`) or white (`#ffffff`) contrasts more with the background, the standard "should this label have dark or light text" decision.

Pass `{ candidates = [...] }` to choose from your own palette instead; the candidate with the highest contrast wins, and it is returned exactly as you wrote it (so your formatting and casing are preserved). On a tie the earliest candidate wins.

This is the workhorse for generated labels, badges, and dashboard panels: compute the background from data, then let this pick text that stays readable.
