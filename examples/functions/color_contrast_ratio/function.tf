// Check whether white text on the brand color passes WCAG AA (>= 4.5) / AAA (>= 7).
output "ratio" {
  value = provider::burnham::color_contrast_ratio("#1e3a8a", "#ffffff")
  // → 10.36
}
