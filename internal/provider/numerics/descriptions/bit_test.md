<!-- Edit here: this is the MarkdownDescription source for the burnham bit_test function. docs/functions/bit_test.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Returns `true` if bit `i` of `value` is set, `false` otherwise. Bit `i` has place value `2^i`, so `bit_test(8, 3) = true` and `bit_test(8, 0) = false`.

Handy for reading a single flag out of a bitmask: `bit_test(flags, 4)`.

-> `value` and `i` must be integers and `i` must be `>= 0`. A negative index, or a non-integral or infinite argument, is an error.