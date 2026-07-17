<!-- Edit here: this is the MarkdownDescription source for the burnham pluscode_encode function. docs/functions/pluscode_encode.md is generated from it by "go generate ./..."; do not edit the generated doc. -->

Returns the [Plus code](https://maps.google.com/pluscodes/) (Open Location Code) for `(latitude, longitude)` at the requested `length`. Code length controls cell size:

| length | cell size |
| ---: | --- |
| 2 | 20° × 20° (~2,200 km) |
| 4 | 1° × 1° (~110 km) |
| 6 | 0.05° × 0.05° (~5.5 km) |
| 8 | 0.0025° × 0.0025° (~275 m) |
| 10 | 0.000125° × 0.000125° (~14 m, OLC standard precision) |
| 11 | ~3.5 m |
| 12 | ~90 cm |

`length` must be even between 2 and 10, or any value 11–15. `latitude` must be in `[-90, 90]`, `longitude` in `[-180, 180]`.

```
pluscode_encode(37.7749, -122.4194, 10)
→ "849VQHFJ+X6"
```