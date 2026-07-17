<!-- Edit here: this is the MarkdownDescription source for the burnham cidr_subtract function. docs/functions/cidr_subtract.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Returns the set of CIDRs that remain after removing all addresses covered by the `exclude` list from the `input` list. The result is merged into the smallest equivalent set.