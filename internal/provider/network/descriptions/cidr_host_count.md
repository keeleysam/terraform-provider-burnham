<!-- Edit here: this is the MarkdownDescription source for the burnham cidr_host_count function. docs/functions/cidr_host_count.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Returns the total number of IP addresses in the CIDR, including the network and broadcast addresses for IPv4.

~> **Note:** For very large IPv6 prefixes the result is capped at `MaxInt64`.