<!-- Edit here: this is the MarkdownDescription source for the burnham ip_add function. docs/functions/ip_add.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Offsets an IP address by an integer, so you can derive nearby addresses like a gateway or DNS server without a full CIDR context.

Returns the IP address shifted by `n`: positive to advance, negative to go back. Supports both IPv4 and IPv6.

Common uses:

- Computing gateway addresses, for example `ip_add(cidr_first_ip(var.subnet), 1)`.
- Deriving DNS server IPs.
- Enumerating specific host addresses without needing a full CIDR context.

~> **Note:** Fails if the result would fall outside the address space, either past the end or, for a negative `n`, before the start.