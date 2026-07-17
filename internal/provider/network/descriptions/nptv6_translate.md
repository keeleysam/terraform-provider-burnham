<!-- Edit here: this is the MarkdownDescription source for the burnham nptv6_translate function. docs/functions/nptv6_translate.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Translates `ipv6` from `from_prefix` to `to_prefix` using the checksum-neutral algorithm defined in RFC 6296. Both prefixes must be `/48`.

The first 48 bits are replaced with the new prefix. An adjustment is then applied to the 16-bit subnet field (bytes 6 and 7, bits 48-63), leaving the interface identifier unchanged, so that the one's complement sum of all 128 bits is preserved, which keeps transport-layer (TCP/UDP) checksums valid without packet rewriting.

Common uses:

- Computing the external address an internal host will appear as through an NPTv6 gateway (e.g. for DNS, ACL, or route configuration).
- Reverse-translating an external address back to its internal form by swapping `from_prefix` and `to_prefix`.