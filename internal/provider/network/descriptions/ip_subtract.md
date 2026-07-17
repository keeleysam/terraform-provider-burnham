<!-- Edit here: this is the MarkdownDescription source for the burnham ip_subtract function. docs/functions/ip_subtract.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Returns the signed integer distance between two IP addresses: how many address positions separate them.

The result is `a - b`:

- Positive means `a` is higher than `b`.
- Negative means `b` is higher than `a`.
- Zero means they are equal.

Both addresses must be in the same family. For IPv4 the result always fits; for IPv6 an error is returned if the difference exceeds the int64 range.

Common uses:

- Asserting that two IPs are within N addresses of each other.
- Computing the length of an arbitrary IP range.
- Confirming an IP falls exactly at the expected offset from a base address.
- Generating loop indices over a sparse range.