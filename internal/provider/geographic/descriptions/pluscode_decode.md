Parses a full Open Location Code and returns:

- `latitude` / `longitude`: the centre of the cell, in degrees.
- `lat_min` / `lat_max` / `lon_min` / `lon_max`: the cell's bounding box.
- `length`: the code's nominal length.

Errors on short codes (those that need a reference location); pre-extend short codes with `olc.RecoverNearest` upstream of Terraform if you need to ingest them.