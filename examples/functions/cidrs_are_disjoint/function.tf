# Validation helper: ensure a list of subnet CIDRs has no overlaps.
variable "subnet_cidrs" {
  type = list(string)
  validation {
    condition     = provider::burnham::cidrs_are_disjoint(var.subnet_cidrs)
    error_message = "subnet_cidrs must not contain overlapping entries."
  }
}
