<!-- Edit here: this is the MarkdownDescription source for the burnham nat64_prefix_valid function. docs/functions/nat64_prefix_valid.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Returns `true` if `prefix` is a valid NAT64 prefix: it must be an IPv6 prefix of length /32, /40, /48, /56, /64, or /96, and the reserved u-octet (bits 64–71) must be zero.

**Common uses:** `variable` validation blocks to reject operator-supplied NAT64 prefixes that would produce malformed translated addresses.