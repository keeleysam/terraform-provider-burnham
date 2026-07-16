Returns the n-th decimal digit of 22/7 *following* the decimal point, formatted as the [RFC 3091 §2.1.2](https://www.rfc-editor.org/rfc/rfc3091#section-2.1.2) UDP reply payload `reply = nth_digit ":" DIGIT`. This is the "approximate service" of RFC 3091 §1.1/§2.2: long division of 22 by 7 gives `3.142857142857…`, a period-6 repeating cycle of `"142857"`.

- `pi_approximate_digit(1)` → `"1:1"`
- `pi_approximate_digit(7)` → `"7:1"` (cycle wraps to start of `"142857"`)
- `pi_approximate_digit(100)` → `"100:8"`

Because 22/7 cycles with period 6, the n-th digit is just `"142857"[(n-1) mod 6]`, a constant-time lookup. There is no upper bound on `n`: it can be arbitrarily large (up to ~10^150 in Terraform's 512-bit number type) and the function returns instantly.