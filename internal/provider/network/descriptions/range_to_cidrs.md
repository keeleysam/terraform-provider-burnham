<!-- Edit here: this is the MarkdownDescription source for the burnham range_to_cidrs function. docs/functions/range_to_cidrs.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Converts an inclusive IP range `[first_ip, last_ip]` into the minimal list of CIDRs that exactly covers the range. Both IPs must be the same address family (IPv4 or IPv6).