Returns the value(s) appearing most frequently in `numbers`, as a sorted ascending list. The result is always a list because the data may be **multimodal**: e.g. `mode([1, 1, 2, 2, 3])` is `[1, 2]`, not just one of them. For unimodal data the list has length 1.

**No mode for all-unique data.** When every value occurs exactly once and the list has more than one element (`mode([1, 2, 3])`), the function returns an empty list rather than echoing the input, since having a "mode" requires at least one value to repeat. Single-element input `mode([5])` returns `[5]` (degenerate but unambiguous).

Two numeric values are considered equal here when they compare equal as `*big.Float` (`Cmp == 0`), so `mode([1, 1.0])` collapses to `[1]`. Errors when `numbers` is empty.