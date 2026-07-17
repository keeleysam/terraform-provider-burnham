<!-- Edit here: this is the MarkdownDescription source for the burnham cidr_merge function. docs/functions/cidr_merge.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Merges a list of CIDR strings into the smallest equivalent set by removing redundant prefixes and combining sibling pairs into supernets. Supports both IPv4 and IPv6.