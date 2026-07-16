Returns the number of usable host addresses in the CIDR. For IPv4, the network and broadcast addresses are subtracted, with special cases: `/31` returns 2 (point-to-point, RFC 3021), `/32` returns 1 (host route). For IPv6, all addresses are considered usable.

**Common uses:** asserting a subnet is large enough for a given number of workloads without manually subtracting 2 everywhere; sizing auto-scaling groups or node pools based on the actual available IP space in the target subnet.

~> **Note:** For very large IPv6 prefixes the result is capped at `MaxInt64`.