<!-- Edit here: this is the MarkdownDescription source for the burnham bit_and function. docs/functions/bit_and.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Returns the bitwise **AND** of every integer in the list, folded left to right (`numbers[0] & numbers[1] & ...`). The list must be non-empty; a single-element list returns that element unchanged. This is the n-ary form, so the binary case is just a two-element list: `bit_and([12, 10]) = 8`.

Terraform has no bitwise operators at all, so this fills the gap. A common use is masking: `bit_and(flags, 0xFF)` keeps the low 8 bits.

-> Every element must be an integer. A non-integral (`1.5`) or infinite element is an error naming the offending value, and an empty list is an error.

~> Negative operands are treated as infinite two's-complement bit strings (matching `math/big`), which is well defined but rarely what you want. The flag / mask use case uses non-negative integers. Values are arbitrary-precision, so nothing overflows.