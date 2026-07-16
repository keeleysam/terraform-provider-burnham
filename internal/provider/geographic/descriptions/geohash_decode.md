Parses `code` and returns:

- `latitude` / `longitude`: the centre of the cell, in degrees.
- `lat_min` / `lat_max` / `lon_min` / `lon_max`: the cell's bounding box (the points the code *might* have been encoded from).

`code` is case-insensitive but must use the standard geohash alphabet: digits `0-9` plus lowercase `b-z` with `i`, `l`, `o` removed. Errors on any other character.

For the corner cell `zzz…z` the geometric upper edges are exactly `(90, 180)`, but the upstream encoder wraps those values; the decoder shrinks `lat_max` / `lon_max` for that cell to a value a few ULPs below the geometric edge (`90` / `180`) that the encoder still accepts, so round-tripping `lat_max` / `lon_max` back through `geohash_encode` lands on the same cell.