<!-- Edit here: this is the MarkdownDescription source for the burnham cidr_find_free function. docs/functions/cidr_find_free.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Finds the first free subnet of a given size inside an address pool, so you can allocate the next subnet without hardcoding offsets.

Returns the first prefix of length `prefix_len` that is available within `pool` after removing all `used` CIDRs.

This is handy for IPAM-style subnet allocation: give your VPC CIDR(s) as the `pool` list (a single CIDR still goes in as a one-element list, e.g. `["10.0.0.0/16"]`) and the list of already-allocated subnets as `used`, and it returns the next free subnet to assign to a new workload. It works well in `locals` blocks to compute the next available AZ subnet.

-> **Note:** Returns `null` if no prefix of that size is available in the pool.