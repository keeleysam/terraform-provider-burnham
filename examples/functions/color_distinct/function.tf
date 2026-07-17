// One stable, distinct color per service for a generated dashboard. Deterministic:
// the palette never churns between plans, so there is no perpetual diff.
locals {
  services = ["api", "web", "worker", "cache", "db"]
  palette  = provider::burnham::color_distinct(length(local.services))
}

output "service_colors" {
  value = zipmap(local.services, local.palette)
}
