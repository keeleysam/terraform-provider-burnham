Returns `value >> n`: `value` shifted right by `n` bit positions. `bit_shift_right(1024, 3) = 128`.

This is an **arithmetic** shift: for a negative `value` it floors toward negative infinity (matching `math/big`'s `Rsh`), so `bit_shift_right(-8, 1) = -4` and `bit_shift_right(-1, 1) = -1`, not truncation toward zero. For non-negative values it is the same as integer division by `2^n`.

Terraform has no shift operators, so this fills the gap.

-> `value` and `n` must be integers and `n` must be `>= 0`. A negative `n`, or a non-integral or infinite argument, is an error.