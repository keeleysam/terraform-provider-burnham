<!-- Edit here: this is the MarkdownDescription source for the burnham percentile function. docs/functions/percentile.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Returns the `p`-th percentile of `numbers` using linear interpolation between adjacent ordered values. This is **Hyndman & Fan Type 7**, the default method in [NumPy](https://numpy.org/doc/stable/reference/generated/numpy.percentile.html), R, and Excel's `PERCENTILE.INC`.

Definition: let the sorted observations be `x[0] ≤ x[1] ≤ … ≤ x[N-1]`. Compute `h = (p / 100) × (N - 1)`. If `h` is an integer, return `x[h]`. Otherwise return `x[⌊h⌋] + (h - ⌊h⌋) × (x[⌈h⌉] - x[⌊h⌋])`.

Valid `p` is a finite number in `[0, 100]`. `p = 0` returns the minimum; `p = 100` returns the maximum; `p = 50` matches `median(numbers)`.

Errors when `numbers` is empty, or when `p` is outside `[0, 100]` or non-finite.