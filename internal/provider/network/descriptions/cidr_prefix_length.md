<!-- Edit here: this is the MarkdownDescription source for the burnham cidr_prefix_length function. docs/functions/cidr_prefix_length.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Extracts and returns just the prefix length from a CIDR string.

**Common uses:** passing prefix lengths to BGP route-map configurations, conditional logic based on subnet size, feeding into `cidrsubnet` calls.