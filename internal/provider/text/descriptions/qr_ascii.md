<!-- Edit here: this is the MarkdownDescription source for the burnham qr_ascii function. docs/functions/qr_ascii.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Returns a multi-line string containing a QR code that encodes `payload`, rendered with Unicode half-block characters so two QR-module rows fit in one terminal row. Scannable directly from any monospaced display with adequate light/dark contrast (light-background terminals and light themes scan the default dark-on-light output directly; dark themes such as white-on-black need the inverted `light_on_dark` variant, see `style`).

Options object:

- `error_correction` (string): error-correction level, one of `"L"` (default, ~7%), `"M"` (~15%), `"Q"` (~25%), `"H"` (~30%). Higher levels survive more occlusion at the cost of a bigger code.
- `quiet_zone` (number): number of empty modules around the code, from `0` to `64`. Default `4` (the [QR spec](https://en.wikipedia.org/wiki/QR_code) minimum). Set to `0` for very tight layouts.
- `style` (string): `"dark_on_light"` (default; dark modules render as `▀ █ ▄`, light as space, for white terminals) or `"light_on_dark"` (inverted, for black terminals).

Layout is half-block: each terminal line covers two QR module rows.

-> **Note:** Payloads have a hard ceiling set by the largest QR version (40). At `error_correction = "L"` the encoder tops out near ~2,950 bytes; higher levels leave less room for data, so `"H"` tops out near ~1,270 bytes. A payload past the level's ceiling returns a "text too long to encode as QR" error, so pick `"H"` for its stronger recovery only when the payload is well within that smaller budget.