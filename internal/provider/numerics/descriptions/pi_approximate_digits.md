Returns the first `count` decimal digits of 22/7 *following* the decimal point as a single ASCII string. Models the [RFC 3091 §1.1](https://www.rfc-editor.org/rfc/rfc3091#section-1.1) TCP approximate service, which streams `"starting with the most significant digit following the decimal point"`: there is no seek operation, so this function takes only `count`.

Because 22/7 is a period-6 repeating decimal, the output for any count is just `"142857"` repeated and truncated.

- `pi_approximate_digits(12)` → `"142857142857"` (the 6-digit cycle, twice)

~> **Note:** `count` greater than 3,141,592 (= ⌊π × 10⁶⌋) errors, matching `pi_digits` so neither function can be coaxed into materialising a multi-GB string at plan time.