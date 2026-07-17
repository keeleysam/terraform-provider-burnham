<!-- Edit here: this is the MarkdownDescription source for the burnham mean function. docs/functions/mean.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Returns Σ x / N, the arithmetic mean of `numbers`. Errors when `numbers` is empty.

Use this when you want a plain average. For weighted means, geometric means, or trimmed means, do the weighting explicitly in HCL. The goal of this function is the unambiguous canonical definition.