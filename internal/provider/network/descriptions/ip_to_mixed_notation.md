Returns the IPv6 address formatted with the last 32 bits expressed as a dotted-decimal IPv4 address, e.g. `64:ff9b::192.0.2.1` instead of `64:ff9b::c000:201`. Zero-compression (::) is applied to the hex portion. IPv4 addresses are returned unchanged.

Common uses:

- Making NAT64 addresses human-readable in outputs and documentation.
- Formatting IPv4-mapped addresses for display.
- Expressing IPv4-compatible addresses in a form that makes the embedded IPv4 immediately visible.