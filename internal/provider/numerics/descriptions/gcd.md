Returns the **greatest common divisor** of a list of integers: the largest integer that divides every element with no remainder. Uses arbitrary-precision arithmetic, so a list of very large integers returns an exact result.

Every element must be an integer. Negatives are reduced to their absolute value, so the result is always non-negative.

Conventions for the edge cases:

- `gcd([0, 0])` is `0` (every integer divides zero, so there is no greatest divisor; 0 is the conventional answer).
- `gcd([0, n])` is `abs(n)` (zero contributes no constraint).
- `gcd([n])` on a single element returns `abs(n)`.

-> Pairs naturally with [`lcm`](lcm.md): `gcd(a, b) * lcm(a, b)` equals `abs(a * b)`.

~> Errors on an empty list (the gcd of no numbers is undefined) and on any non-integer element (for example `gcd([1.5, 2])`); the error names the offending value.