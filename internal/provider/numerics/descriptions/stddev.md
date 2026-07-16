Returns σ = √(Σ (x − μ)² / N), the **population standard deviation**, where μ = `mean(numbers)`.

Population formula (divide by N), matching numpy's default. For sample standard deviation, take `sqrt(variance(numbers) × length(numbers) / (length(numbers) - 1))`. Errors when `numbers` is empty.