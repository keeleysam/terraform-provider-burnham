<!-- Edit here: this is the MarkdownDescription source for the burnham nat64_synthesize function. docs/functions/nat64_synthesize.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Embeds `ipv4` into the given NAT64 `prefix` following the RFC 6052 byte layout to produce the corresponding IPv6 address. The prefix must be `/32`, `/40`, `/48`, `/56`, `/64`, or `/96`.

By default the address is returned in mixed `x:x:x:x:x:x:d.d.d.d` notation (e.g. `64:ff9b::192.0.2.1`). Pass `true` as the optional third argument to get standard hex notation instead.

Common uses:

- Pre-computing NAT64 pool members for DNS64 AAAA records.
- Configuring NAT64 gateway address pools.
- Generating IPv6 addresses for IPv4-only services.