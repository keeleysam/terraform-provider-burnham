<!-- Edit here: this is the MarkdownDescription source for the burnham ip_version function. docs/functions/ip_version.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Returns `4` for IPv4 addresses and `6` for IPv6 addresses. IPv4-mapped IPv6 addresses (e.g. `::ffff:10.0.0.1`) are treated as IPv4.