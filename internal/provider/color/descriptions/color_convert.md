Parses a color in any CSS Color 4 notation and re-serializes it in a target notation. Accepts hex (`#rgb`, `#rrggbb`, `#rrggbbaa`), `rgb()` / `rgba()`, `hsl()` / `hsla()`, `hwb()`, `lab()` / `lch()`, `oklab()` / `oklch()`, and the CSS named colors; the `target` is one of `hex`, `rgb`, `hsl`, or `oklch`.

The common use is normalizing input for a resource whose color field is picky: `github_issue_label.color` wants six hex digits with no leading `#`, while `gitlab_label.color` wants a leading `#` or a name. Pass `{ hash = false }` to drop the `#` and `{ uppercase = true }` to upper-case the hex digits.

Alpha is preserved when it is below 1: hex output becomes eight digits, and `rgb` / `hsl` / `oklch` gain their alpha component. The math runs through go-colorful, so a color that round-trips through `oklch` and back stays perceptually faithful.
