Recovers the IPv4 address from a NAT64 IPv6 address.

With no second argument, extracts the **last 32 bits** of the IPv6 address as a dotted-decimal IPv4 string. This is correct for the overwhelmingly common case: the Well-Known Prefix `64:ff9b::/96` and any other `/96` NAT64 prefix.

With an optional `nat64_prefix` argument (e.g. `"2001:db8::/48"`), uses the RFC 6052 byte layout for that prefix length instead, needed for `/32`–`/64` prefixes where the IPv4 bytes don't sit in the last 32 bits.

Common uses:

- Reverse-mapping NAT64 addresses in flow logs or firewall hits back to the original IPv4.
- ACL generation from IPv6 traffic records.