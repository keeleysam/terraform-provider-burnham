/*
Population variance (Σ (x − μ)² / N) — matches numpy's default. For sample variance (Bessel's correction), multiply by length(numbers) / (length(numbers) - 1).
*/
output "population_variance" {
  value = provider::burnham::variance([2, 4, 4, 4, 5, 5, 7, 9])
  // → 4   (mean = 5; sum of squared deviations = 32; 32/8 = 4)
}

output "sample_variance" {
  value = provider::burnham::variance([2, 4, 4, 4, 5, 5, 7, 9]) * 8 / 7
  // → 4.571428…  (Bessel-corrected, divide by n-1 instead of n)
}
