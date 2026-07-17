<!-- Edit here: this is the MarkdownDescription source for the burnham pluscode_decode function. docs/functions/pluscode_decode.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Parses a full Open Location Code and returns:

- `latitude` / `longitude`: the centre of the cell, in degrees.
- `lat_min` / `lat_max` / `lon_min` / `lon_max`: the cell's bounding box.
- `length`: the code's nominal length.

Errors on short codes (those that need a reference location). Recover a short code to a full code against a reference location outside Terraform (any Open Location Code library exposes a recover-nearest operation) and pass the resulting full code in.