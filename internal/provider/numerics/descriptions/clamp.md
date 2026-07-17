<!-- Edit here: this is the MarkdownDescription source for the burnham clamp function. docs/functions/clamp.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Returns `value` if it falls within `[min_val, max_val]`, `min_val` if `value < min_val`, and `max_val` if `value > max_val`. Equivalent to `max(min_val, min(max_val, value))` but easier to read and harder to get backwards.

Errors when `min_val > max_val`: the interval is empty in that case and any return value would be a guess. Both bounds are inclusive.