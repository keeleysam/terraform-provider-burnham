# Provider config lives in dataformat.tf; both files load together when
# `terraform apply` runs against this directory.

# ─── CIDR set operations ─────────────────────────────────────────

output "cidr_merge" {
  description = "Aggregate adjacent and redundant CIDRs"
  value = provider::burnham::cidr_merge([
    "10.0.0.0/24", "10.0.1.0/24", # → "10.0.0.0/23"
    "10.0.0.0/25",                # redundant
    "192.168.1.0/24",
  ])
}

output "cidr_subtract" {
  description = "Carve a reserved range out of an allocation"
  value       = provider::burnham::cidr_subtract(["10.0.0.0/8"], ["10.1.0.0/16"])
}

output "cidr_intersect" {
  description = "Address space common to two lists"
  value = provider::burnham::cidr_intersect(
    ["10.0.0.0/8", "172.16.0.0/12"],
    ["10.100.0.0/16", "192.168.0.0/16"],
  )
}

output "range_to_cidrs" {
  description = "Convert an inclusive IP range (e.g. cloud feed) to CIDRs"
  value       = provider::burnham::range_to_cidrs("10.0.0.1", "10.0.0.6")
}

output "cidr_enumerate" {
  description = "All /26 subnets within a /24"
  value       = provider::burnham::cidr_enumerate("10.0.0.0/24", 2)
}

# ─── Queries / validation ────────────────────────────────────────

output "ip_in_cidr" {
  description = "Is the bastion address inside the management subnet?"
  value       = provider::burnham::ip_in_cidr("10.0.1.50", "10.0.1.0/24")
}

output "cidr_contains" {
  description = "Is one CIDR fully nested inside another?"
  value       = provider::burnham::cidr_contains("10.0.0.0/8", "10.1.2.0/24")
}

output "cidrs_are_disjoint" {
  description = "Validate a list of subnet CIDRs has no overlaps"
  value       = provider::burnham::cidrs_are_disjoint(["10.0.0.0/24", "10.0.1.0/24"])
}

# ─── Decomposition / arithmetic ──────────────────────────────────

locals {
  mgmt_subnet = "10.0.1.0/24"
  mgmt_first  = provider::burnham::cidr_first_ip(local.mgmt_subnet)
}

output "subnet_gateway" {
  description = "Conventional gateway address: subnet base + 1"
  value       = provider::burnham::ip_add(local.mgmt_first, 1)
}

output "subnet_dns" {
  description = "Conventional DNS address: subnet base + 2"
  value       = provider::burnham::ip_add(local.mgmt_first, 2)
}

output "subnet_usable_hosts" {
  description = "Usable hosts in a /24 (excludes network + broadcast)"
  value       = provider::burnham::cidr_usable_host_count("10.0.1.0/24")
}

output "ipv4_cisco_wildcard" {
  description = "Wildcard mask for Cisco-style ACL entries"
  value       = provider::burnham::cidr_wildcard("10.0.0.0/24")
}

# ─── Private-range checks ────────────────────────────────────────

output "is_private_192" {
  description = "RFC 1918"
  value       = provider::burnham::ip_is_private("192.168.1.1")
}

output "is_private_cgnat" {
  description = "RFC 6598 CGNAT"
  value       = provider::burnham::ip_is_private("100.64.0.1")
}

output "is_private_8888" {
  description = "Public — Google DNS"
  value       = provider::burnham::ip_is_private("8.8.8.8")
}

# ─── Dual-stack: NAT64 synthesis from an IPv4 allowlist ──────────

locals {
  ipv4_allowlist = ["203.0.113.0/24", "198.51.100.0/24"]
  nat64_prefix   = "64:ff9b::/96"
  ipv6_allowlist = provider::burnham::nat64_synthesize_cidrs(local.ipv4_allowlist, local.nat64_prefix)
  full_allowlist = concat(local.ipv4_allowlist, local.ipv6_allowlist)
}

output "nat64_full_allowlist" {
  description = "Original IPv4 ranges plus their NAT64 IPv6 equivalents"
  value       = local.full_allowlist
}

output "nat64_synthesize_single_host" {
  description = "Single-host NAT64 synthesis in mixed notation"
  value       = provider::burnham::nat64_synthesize("192.0.2.1", "64:ff9b::/96")
}

output "nat64_extract_roundtrip" {
  description = "Recover the IPv4 from a NAT64 IPv6 address"
  value       = provider::burnham::nat64_extract("64:ff9b::192.0.2.1")
}

# ─── NPTv6: ULA → public prefix translation ──────────────────────

output "nptv6_translated" {
  description = "Translate a ULA address to its checksum-neutral PA equivalent"
  value = provider::burnham::nptv6_translate(
    "fd00::10:0:1",
    "fd00::/48",
    "2001:db8::/48",
  )
}

# ─── IPAM: find the next free subnet ─────────────────────────────

output "next_free_subnet" {
  description = "Next available /24 in the VPC pool, skipping allocated blocks"
  value = provider::burnham::cidr_find_free(
    ["10.0.0.0/16"],
    ["10.0.0.0/24", "10.0.1.0/24"],
    24,
  )
}
