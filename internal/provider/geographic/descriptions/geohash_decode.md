Parses `code` and returns:

- `latitude` / `longitude`: the centre of the cell, in degrees.
- `lat_min` / `lat_max` / `lon_min` / `lon_max`: the cell's bounding box (the points the code *might* have been encoded from).

`code` is case-insensitive but must use the standard geohash alphabet: digits `0-9` plus lowercase `b-z` with `i`, `l`, `o` removed. Errors on any other character.

Cells on the northern edge report a `lat_max` of exactly `90`, and cells on the eastern edge report a `lon_max` of exactly `180` (the corner cell `zzz…z` hits both). The upstream encoder wraps those values, so the decoder shrinks any such edge a few ULPs below the geometric edge (`90` / `180`) to a value the encoder still accepts, keeping `lat_max` / `lon_max` round-trippable back through `geohash_encode`.