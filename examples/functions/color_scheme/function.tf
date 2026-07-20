// Derive a coherent set of brand accent colors from one seed color. The palette
// is generated in OKLCh, so every color reads at the same perceived weight, and
// it is deterministic, so there is no perpetual diff between plans.
locals {
  brand = "#2563eb"
}

output "triadic" {
  value = provider::burnham::color_scheme(local.brand, "triadic")
  // → ["#2563eb", "#d10e2e", "#008c00"]
}

output "complementary" {
  value = provider::burnham::color_scheme(local.brand, "complementary")
  // → ["#2563eb", "#ab5b00"]
}

// Widen the analogous neighbors from the default 30 degrees to 45.
output "analogous_wide" {
  value = provider::burnham::color_scheme(local.brand, "analogous", { angle = 45 })
}
