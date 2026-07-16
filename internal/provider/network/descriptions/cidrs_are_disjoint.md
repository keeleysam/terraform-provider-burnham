Checks that a single list of CIDRs contains no overlaps, so you can validate a set of subnets before creating them.

Returns `true` if every CIDR in the list is non-overlapping with every other. Unlike `cidrs_overlap_any`, which compares two separate lists, this checks a single list against itself.

A typical use is validating a `list(string)` variable of subnet CIDRs to ensure no two subnets overlap. This catches mistakes like including both a summary prefix and a more-specific one in the same list.