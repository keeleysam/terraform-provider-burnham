Returns the IPv6 CIDR that corresponds to `ipv4_cidr` under the given NAT64 `nat64_prefix`. Only /64 and /96 NAT64 prefixes are supported.

By default returns the result in mixed `x:x:x:x:x:x:d.d.d.d/N` notation. Pass `true` as the optional third argument to use standard hex notation.

**Common uses:** pre-computing the IPv6 pool CIDR for a NAT64 gateway configuration; expressing IPv4 address allocations in IPv6 space for dual-stack firewall rules.