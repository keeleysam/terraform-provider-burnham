<!-- Edit here: this is the MarkdownDescription source for the burnham popcount function. docs/functions/popcount.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Returns the number of set bits (the **Hamming weight** or population count) in `value`. `popcount(255) = 8`, `popcount(0) = 0`.

`value` is arbitrary-precision, so the count is exact past 64 bits: `popcount(2^64) = 1`.

Useful for counting how many flags are set in a bitmask.

-> `value` must be a non-negative integer. A negative value is an error: in two's-complement it has infinitely many set bits, so the count is undefined. A non-integral or infinite argument is also an error.