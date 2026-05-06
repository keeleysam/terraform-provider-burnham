# Find the first free /N within a pool, skipping already-allocated
# CIDRs. Returns null if no prefix of that size is available.
output "next_subnet" {
  value = provider::burnham::cidr_find_free(
    ["10.0.0.0/16"],
    ["10.0.0.0/24", "10.0.1.0/24"],
    24,
  )
  # → "10.0.2.0/24"
}
