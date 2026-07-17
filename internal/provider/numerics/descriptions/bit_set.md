Returns `value` with bit `i` set to 1. Bit `i` has place value `2^i`, so `bit_set(0, 3) = 8`. Setting a bit that is already set is a no-op.

Handy for flag manipulation: `bit_set(flags, 4)` turns on the bit at index 4 without disturbing the others.

-> `value` and `i` must be integers and `i` must be `>= 0`. A negative index, or a non-integral or infinite argument, is an error.