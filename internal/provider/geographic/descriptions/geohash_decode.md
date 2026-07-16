Parses `code` and returns:

- `latitude` / `longitude`: the centre of the cell, in degrees.
- `lat_min` / `lat_max` / `lon_min` / `lon_max`: the cell's bounding box (the points the code *might* have been encoded from).

`code` is case-insensitive but must use the standard geohash alphabet: digits `0-9` plus lowercase `b-z` with `i`, `l`, `o` removed. Errors on any other character.

The upstream bounding box reaches exactly `90` on the northern edge and exactly `180` on the eastern edge (the corner cell `zzz…z` hits both). Those values wrap if fed back into the encoder, so `geohash_decode` instead reports `lat_max` / `lon_max` a few ULPs below the geometric edge (`90` / `180`), a value the encoder still accepts, keeping the cell round-trippable back through `geohash_encode`.