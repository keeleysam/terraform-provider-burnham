<!-- Edit here: this is the MarkdownDescription source for the burnham pi_digits function. docs/functions/pi_digits.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Returns the first `count` decimal digits of π *following* the decimal point as a single ASCII string. Models the [RFC 3091 §1](https://www.rfc-editor.org/rfc/rfc3091#section-1) TCP service, which always streams "starting with the most significant digit following the decimal point". There is no seek operation in the protocol, so this function takes only `count`, not a starting position.

- `pi_digits(10)` → `"1415926535"` (the leading 3 is implied per the RFC's "Note" section and never emitted)

~> **Note:** `count` > 3,141,592 (= ⌊π × 10⁶⌋) errors. See `pi_digit` for the rationale.