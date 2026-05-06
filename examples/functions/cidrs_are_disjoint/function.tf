// Validation helper: ensure a list of subnet CIDRs has no overlaps.
output "ok" {
  value = provider::burnham::cidrs_are_disjoint(["10.0.0.0/24", "10.0.1.0/24"])
  // → true
}

// Idiomatic use is in a variable validation block:
variable "subnet_cidrs" {
  type = list(string)
  validation {
    condition     = provider::burnham::cidrs_are_disjoint(var.subnet_cidrs)
    error_message = "subnet_cidrs must not contain overlapping entries."
  }
}
