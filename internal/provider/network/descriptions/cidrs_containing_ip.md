<!-- Edit here: this is the MarkdownDescription source for the burnham cidrs_containing_ip function. docs/functions/cidrs_containing_ip.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Finds every CIDR in a list that contains a given IP address, useful when overlapping prefixes mean an address can belong to more than one range.

Returns every CIDR from `cidrs` that contains `ip`, as a list. Multiple CIDRs may match when the list contains overlapping prefixes (for example a summary `/8` and a more-specific `/24` both match).

Common uses:

- Routing decisions: given an observed IP, find every VRF, VPC, or security zone it belongs to.
- Determining which policy rules apply to a given address.

-> **Note:** Returns an empty list if no CIDR matches.