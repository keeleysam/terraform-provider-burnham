<!-- Edit here: this is the MarkdownDescription source for the burnham cowsay function. docs/functions/cowsay.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Returns `message` rendered as the original `cowsay(1)` would: a multi-line speech bubble (or thought bubble) attached to an ASCII cow figure. Useful for embedding in `/etc/motd` via cloud-init, login banners, or anywhere a generated config benefits from a recognizable greeting.

Options object:

- `action` (string): `"say"` (default) or `"think"`. Bubble geometry per action:
    - `"say"`: a single-line bubble uses `< >`; a multi-line bubble uses `/ \` top corners, `\ /` bottom corners, and `| |` sides. The cow is drawn with a `\` connector.
    - `"think"`: `( )` throughout, with `o` connectors.
- `eyes` (string): exactly two printable characters used for the cow's eyes. Default `"oo"`. Common alternatives: `"=="` (drowsy), `"@@"` (paranoid), `"--"` (dead), `"$$"` (greedy), `"OO"` (surprised). Control characters and ANSI escapes are rejected so a customised `eyes` value can't shift the cow's alignment or smuggle terminal codes into the rendered output.
- `tongue` (string): exactly two printable characters (or empty for no tongue). Default empty. Common: `"U "` (sticking out), `"V "` (vampire).
- `width` (number): wrap the input message to this many columns before rendering. Default `40`. Set to `0` to disable wrapping (lines stay as you wrote them).

Message lines are word-wrapped at `width` codepoints by default, matching upstream cowsay's `-W` option.

~> **Note:** Inputs longer than 64 KiB are rejected to bound plan-time memory.