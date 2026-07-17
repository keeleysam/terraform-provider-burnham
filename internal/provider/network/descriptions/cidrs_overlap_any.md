<!-- Edit here: this is the MarkdownDescription source for the burnham cidrs_overlap_any function. docs/functions/cidrs_overlap_any.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Reports whether two sets of CIDRs share any address space, so you can catch conflicts before creating resources.

Returns `true` if any CIDR in list `a` overlaps with any CIDR in list `b`.

Common uses:

- Pre-flight validation in `variable` validation blocks, to ensure a proposed VPC CIDR does not conflict with existing peered networks.
- Checking that new security group ranges don't collide with reserved address space.