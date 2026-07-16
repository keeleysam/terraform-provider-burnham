Returns the RFC 4291 IPv4-mapped IPv6 representation of an IPv4 address in mixed notation, e.g. `192.0.2.1` → `::ffff:192.0.2.1`.

Common uses:

- Configuring dual-stack listeners and sockets that accept both IPv4 and IPv6 connections.
- Expressing IPv4 addresses in systems that require IPv6 format (e.g. some BGP implementations, certain firewall APIs).
- Building allowlists that cover IPv4-mapped IPv6 representations of IPv4 addresses.