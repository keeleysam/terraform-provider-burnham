<!-- Edit here: this is the MarkdownDescription source for the burnham cidr_expand function. docs/functions/cidr_expand.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Returns every individual IP address within the given CIDR as a list of strings. Returns an error if the CIDR contains more than 65536 addresses to prevent accidental large expansions.