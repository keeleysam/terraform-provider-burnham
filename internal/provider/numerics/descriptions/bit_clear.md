<!-- Edit here: this is the MarkdownDescription source for the burnham bit_clear function. docs/functions/bit_clear.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Returns `value` with bit `i` cleared to 0. Bit `i` has place value `2^i`, so `bit_clear(15, 1) = 13`. Clearing a bit that is already clear is a no-op.

Handy for flag manipulation: `bit_clear(flags, 4)` turns off the bit at index 4 without disturbing the others.

-> `value` and `i` must be integers and `i` must be `>= 0`. A negative index, or a non-integral or infinite argument, is an error.