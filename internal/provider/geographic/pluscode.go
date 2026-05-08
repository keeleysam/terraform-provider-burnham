/*
Plus codes — Open Location Code (OLC).

Google's geohash competitor. Same idea — encode (lat, lon) as a short alphanumeric string — but with a different alphabet, decoding visually-distinct cell sizes (every 2 characters narrows the cell by ~20×), and explicit support for short codes anchored at a reference location. Increasingly used in places without street addresses; the United Nations and several national postal services have adopted it.

`pluscode_encode` builds a full code (or a code of the requested length); `pluscode_decode` returns the centre point and bounding box.

Backed by Google's reference implementation [`github.com/google/open-location-code/go`](https://github.com/google/open-location-code/tree/main/go).
*/

package geographic

import (
	"context"
	"fmt"
	"math/big"

	olc "github.com/google/open-location-code/go"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	pluscodeDefaultLength = 10 // OLC default — ≈ 14 m × 14 m at the equator
	pluscodeMinLength     = 2
	pluscodeMaxLength     = 15
)

// ──────────────────────────────────────────────────────────────────────
// pluscode_encode
// ──────────────────────────────────────────────────────────────────────

var _ function.Function = (*PluscodeEncodeFunction)(nil)

type PluscodeEncodeFunction struct{}

func NewPluscodeEncodeFunction() function.Function { return &PluscodeEncodeFunction{} }

func (f *PluscodeEncodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "pluscode_encode"
}

func (f *PluscodeEncodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Encode (latitude, longitude) into an Open Location Code (Plus code)",
		MarkdownDescription: "Returns the [Plus code](https://maps.google.com/pluscodes/) (Open Location Code) for `(latitude, longitude)` at the requested `length`. Code length controls cell size:\n\n| length | cell size |\n| ---: | --- |\n| 2 | 1° × 1° (~110 km) |\n| 4 | 0.05° × 0.05° (~5.5 km) |\n| 6 | ~275 m |\n| 8 | ~14 m |\n| 10 | ~3.5 m (default) |\n| 11 | ~70 cm |\n\n`length` must be even between 2 and 10, or any value 11–15. `latitude` must be in `[-90, 90]`, `longitude` in `[-180, 180]`.\n\n```\npluscode_encode(37.7749, -122.4194, 10)\n→ \"849VQHFJ+X6\"\n```",
		Parameters: []function.Parameter{
			function.NumberParameter{Name: "latitude", Description: "Latitude in degrees, [-90, 90]."},
			function.NumberParameter{Name: "longitude", Description: "Longitude in degrees, [-180, 180]."},
			function.Int64Parameter{Name: "length", Description: "Code length: even in [2, 10], or any value in [11, 15]. Out-of-range values are an error."},
		},
		Return: function.StringReturn{},
	}
}

// validatePluscodeLength enforces the OLC length rules: even values 2-10 (every 2 characters narrows cell ~20×), or 11-15 for sub-metre precision.
func validatePluscodeLength(n int64) *function.FuncError {
	if n < pluscodeMinLength || n > pluscodeMaxLength {
		return function.NewArgumentFuncError(2, fmt.Sprintf("length must be in [%d, %d]; received %d", pluscodeMinLength, pluscodeMaxLength, n))
	}
	if n <= 10 && n%2 != 0 {
		return function.NewArgumentFuncError(2, fmt.Sprintf("length in [2, 10] must be even; received %d", n))
	}
	return nil
}

func (f *PluscodeEncodeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var length int64
	lat, lon, ferr := fetchLatLon(ctx, req, &length)
	if ferr != nil {
		resp.Error = ferr
		return
	}
	if ferr := validatePluscodeLength(length); ferr != nil {
		resp.Error = ferr
		return
	}
	out := olc.Encode(lat, lon, int(length))
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &out))
}

// ──────────────────────────────────────────────────────────────────────
// pluscode_decode
// ──────────────────────────────────────────────────────────────────────

var _ function.Function = (*PluscodeDecodeFunction)(nil)

type PluscodeDecodeFunction struct{}

func NewPluscodeDecodeFunction() function.Function { return &PluscodeDecodeFunction{} }

func (f *PluscodeDecodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "pluscode_decode"
}

// pluscodeDecodeAttrs is the schema returned by pluscode_decode. Same shape as geohash_decode plus a `length` field reflecting the code's nominal length.
var pluscodeDecodeAttrs = map[string]attr.Type{
	"latitude":  types.NumberType,
	"longitude": types.NumberType,
	"lat_min":   types.NumberType,
	"lat_max":   types.NumberType,
	"lon_min":   types.NumberType,
	"lon_max":   types.NumberType,
	"length":    types.Int64Type,
}

func (f *PluscodeDecodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Decode a Plus code into centre point, bounding box, and length",
		MarkdownDescription: "Parses a full Open Location Code and returns its centre, bounding box, and original length. Errors on short codes (those that need a reference location); pre-extend short codes with `olc.RecoverNearest` upstream of Terraform if you need to ingest them.",
		Parameters: []function.Parameter{
			function.StringParameter{Name: "code", Description: "The Plus code to decode (full code, including the `+`)."},
		},
		Return: function.ObjectReturn{AttributeTypes: pluscodeDecodeAttrs},
	}
}

func (f *PluscodeDecodeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var code string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &code))
	if resp.Error != nil {
		return
	}
	if err := olc.CheckFull(code); err != nil {
		resp.Error = function.NewArgumentFuncError(0, fmt.Sprintf("code must be a full Open Location Code: %s", err.Error()))
		return
	}
	area, err := olc.Decode(code)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, fmt.Sprintf("decoding Plus code: %s", err.Error()))
		return
	}
	centerLat, centerLon := area.Center()
	out, diags := types.ObjectValue(pluscodeDecodeAttrs, map[string]attr.Value{
		"latitude":  types.NumberValue(big.NewFloat(centerLat)),
		"longitude": types.NumberValue(big.NewFloat(centerLon)),
		"lat_min":   types.NumberValue(big.NewFloat(area.LatLo)),
		"lat_max":   types.NumberValue(big.NewFloat(area.LatHi)),
		"lon_min":   types.NumberValue(big.NewFloat(area.LngLo)),
		"lon_max":   types.NumberValue(big.NewFloat(area.LngHi)),
		"length":    types.Int64Value(int64(area.Len)),
	})
	if diags.HasError() {
		resp.Error = function.NewFuncError("building pluscode_decode result")
		return
	}
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &out))
}
