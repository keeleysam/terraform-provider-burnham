<!-- Edit here: this is the MarkdownDescription source for the burnham geohash_encode function. docs/functions/geohash_encode.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Returns the [geohash](https://en.wikipedia.org/wiki/Geohash) of the given `(latitude, longitude)` at the requested `precision` (number of base-32 characters). Higher precision = smaller cell. Approximate cell side at the equator (worst case):

| precision | cell side |
| ---: | --- |
| 1 | ~5,000 km |
| 3 | ~156 km |
| 5 | ~4.9 km |
| 7 | ~152 m |
| 9 | ~4.8 m |
| 12 | ~3.7 cm |

`precision` must be in `[1, 12]`. `latitude` must be in `[-90, 90]` and `longitude` in `[-180, 180]`.

~> **Note:** the upper bounds wrap under the standard encoder. A `latitude` at or within one ULP of `90`, and a `longitude` at or within one ULP of `180`, are rejected (they would wrap to the south pole / antimeridian). Pass a value at least two ULPs below the upper bound. For `longitude` you can instead use `-180`, which is the same meridian as `+180`; for `latitude` there is no equivalent edge, since `+90` and `-90` are opposite poles. The lower bounds `-90` and `-180` encode fine.

```
geohash_encode(37.7749, -122.4194, 7)
→ "9q8yyk8"   (≈ Civic Center, San Francisco)
```