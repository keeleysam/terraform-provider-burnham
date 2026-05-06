// Combine adjacent and redundant CIDRs into the smallest equivalent set.
output "merged" {
  value = provider::burnham::cidr_merge([
    "10.0.0.0/24", "10.0.1.0/24", // → folds to "10.0.0.0/23"
    "10.0.0.0/25",                // redundant — already inside /24
  ])
  // → ["10.0.0.0/23"]
}
