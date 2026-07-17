<!-- Edit here: this is the MarkdownDescription source for the burnham mod_floor function. docs/functions/mod_floor.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Returns the **floor modulo** of `a` by `b`: `a − b·⌊a/b⌋`. The result follows the sign of the divisor `b` (it is always 0 or has `b`'s sign), not the dividend's sign the way truncated modulo does, so for `b > 0` the result is always in `[0, b)`, exactly the "wrap a possibly-negative index into the array length" behaviour Python's `%` operator gives you.

This is *not* the same as Terraform's built-in `%` operator. The built-in follows Go's truncated-modulo convention, which keeps the sign of the dividend: `-7 % 3 = -1` (Terraform/Go) vs `mod_floor(-7, 3) = 2` (this function). Both are valid "modulo" definitions; this one is the one that makes `mod_floor(i, n)` a safe array-wrapping idiom for any integer `i`.

Errors when `b == 0` (division by zero is undefined regardless of which modulo flavour you choose), and when either `a` or `b` is non-finite (an infinite input has no meaningful floor-modulo).