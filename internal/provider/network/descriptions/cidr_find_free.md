Finds the first free subnet of a given size inside an address pool, so you can allocate the next subnet without hardcoding offsets.

Returns the first prefix of length `prefix_len` that is available within `pool` after removing all `used` CIDRs.

This is handy for IPAM-style subnet allocation: give a VPC CIDR as the `pool` and the list of already-allocated subnets as `used`, and it returns the next free subnet to assign to a new workload. It works well in `locals` blocks to compute the next available AZ subnet.

-> **Note:** Returns `null` if no prefix of that size is available in the pool.