<!-- Edit here: this is the MarkdownDescription source for the burnham pi_approximate_digits function. docs/functions/pi_approximate_digits.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Returns the first `count` decimal digits of 22/7 *following* the decimal point as a single ASCII string. Models the [RFC 3091 §1.1](https://www.rfc-editor.org/rfc/rfc3091#section-1.1) TCP approximate service, which streams `"starting with the most significant digit following the decimal point"`: there is no seek operation, so this function takes only `count`.

Because 22/7 is a period-6 repeating decimal, the output for any count is just `"142857"` repeated and truncated.

- `pi_approximate_digits(12)` → `"142857142857"` (the 6-digit cycle, twice)

~> **Note:** `count` greater than 3,141,592 (= ⌊π × 10⁶⌋) errors, matching `pi_digits`. At one byte per digit the output can never exceed a few megabytes, keeping both `pi_approximate_digits` and `pi_digits` bounded at plan time.