<!-- Edit here: this is the MarkdownDescription source for the burnham cidr_enumerate function. docs/functions/cidr_enumerate.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Returns every possible subnet of size `(parent prefix length + newbits)` within `cidr`. For example, `cidr_enumerate("10.0.0.0/24", 2)` returns all four /26 subnets within the /24. Returns an error if the result would exceed 65536 subnets.

**Common uses:** pre-computing all AZ-sized subnets within a region block for later `element()` selection; generating candidate subnets to feed into an IP address management workflow.