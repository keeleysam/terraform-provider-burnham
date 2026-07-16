Converts a list of IPv4 CIDRs into their NAT64 IPv6 CIDR equivalents in a single call.

Returns each IPv4 CIDR in `ipv4_cidrs` converted to its IPv6 equivalent under `nat64_prefix`. Only `/64` and `/96` NAT64 prefixes are supported.

By default the result uses mixed `x:x:x:x:x:x:d.d.d.d/N` notation. Pass `true` as the optional third argument to use standard hex notation instead.

Common uses:

- Translating an entire IPv4 address plan into NAT64 IPv6 space in one call.
- Generating the full set of IPv6 pool CIDRs for a NAT64 service.
- Building dual-stack firewall allowlists that include both the IPv4 and NAT64-IPv6 forms of the same ranges.