<!-- Edit here: this is the MarkdownDescription source for the burnham lcm function. docs/functions/lcm.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Returns the **least common multiple** of a list of integers: the smallest non-negative integer that every element divides evenly. Uses arbitrary-precision arithmetic and combines pairwise as `l / gcd(l, x) * abs(x)`, dividing before multiplying so an intermediate product never overflows and the result stays exact for very large inputs.

Every element must be an integer. Negatives are reduced to their absolute value, so the result is always non-negative.

Conventions for the edge cases:

- `lcm([0, n])` is `0`: any list containing a zero has a least common multiple of `0`.
- `lcm([n])` on a single element returns `abs(n)`.

-> Pairs naturally with [`gcd`](gcd.md): `gcd(a, b) * lcm(a, b)` equals `abs(a * b)`.

~> Errors on an empty list (the lcm of no numbers is undefined) and on any non-integer element (for example `lcm([1.5, 2])`); the error names the offending value.