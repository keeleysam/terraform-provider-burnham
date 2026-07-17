<!-- Edit here: this is the MarkdownDescription source for the burnham variance function. docs/functions/variance.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Returns the **population variance** σ² = Σ (x − μ)² / N, where μ = `mean(numbers)`.

This is the population formula (divide by N), matching numpy's default. For sample variance (Bessel's correction, divide by N-1), multiply by `length(numbers) / (length(numbers) - 1)`. Errors when `numbers` is empty.