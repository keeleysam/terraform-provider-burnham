Returns `value << n`: `value` shifted left by `n` bit positions, equivalent to multiplying by `2^n`. `bit_shift_left(1, 10) = 1024`.

The result is an arbitrary-precision integer, so shifts that overflow a 64-bit word are exact: `bit_shift_left(1, 100)` returns `2^100` in full.

Terraform has no shift operators, so this fills the gap.

-> `value` and `n` must be integers and `n` must be `>= 0`. A negative `n`, or a non-integral or infinite argument, is an error. For a right shift use `bit_shift_right`.