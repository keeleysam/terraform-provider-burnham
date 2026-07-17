<!-- Edit here: this is the MarkdownDescription source for the burnham bit_xor function. docs/functions/bit_xor.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Returns the bitwise **XOR** of every integer in the list, folded left to right (`numbers[0] ^ numbers[1] ^ ...`). The list must be non-empty; a single-element list returns that element unchanged.

`bit_xor([5, 3]) = 6`. XOR is its own inverse, so folding a value in twice cancels it: `bit_xor([5, 3, 6]) = 0`.

Terraform has no bitwise operators at all, so this fills the gap.

-> Every element must be an integer. A non-integral (`1.5`) or infinite element is an error naming the offending value, and an empty list is an error.

~> Negative operands are treated as infinite two's-complement bit strings (matching `math/big`). Values are arbitrary-precision, so nothing overflows.