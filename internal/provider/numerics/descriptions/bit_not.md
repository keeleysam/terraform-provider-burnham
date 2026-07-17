<!-- Edit here: this is the MarkdownDescription source for the burnham bit_not function. docs/functions/bit_not.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Returns the bitwise complement of `value` within an unsigned field of `bits` bits: `value XOR (2^bits - 1)`. Every bit of the field is flipped, so `bit_not(0, 8) = 255`, `bit_not(255, 8) = 0`, and `bit_not(1, 4) = 14`.

The width is required on purpose. A width-less NOT of an integer is infinite in two's-complement (`~0` is `-1`, an endless string of ones), which is almost never what a configuration wants. Pinning the field width makes the result a plain unsigned integer.

-> `bits` must be `>= 1` and `value` must satisfy `0 <= value < 2^bits`. A value outside that range (for example `bit_not(256, 8)`) is an error, as is a non-integral or infinite argument.