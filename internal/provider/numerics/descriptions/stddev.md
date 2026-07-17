<!-- Edit here: this is the MarkdownDescription source for the burnham stddev function. docs/functions/stddev.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Returns σ = √(Σ (x − μ)² / N), the **population standard deviation**, where μ = `mean(numbers)`.

Population formula (divide by N), matching numpy's default. For sample standard deviation, take `sqrt(variance(numbers) × length(numbers) / (length(numbers) - 1))`. Errors when `numbers` is empty.