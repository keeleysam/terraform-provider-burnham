/*
Geohash: interleaved-bits geocoding.

Geohash encodes a (lat, lon) pair into a base-32 string where each successive character refines the cell ten-fold (alternating between latitude and longitude). It's the de-facto format for spatial indexing in things like Elasticsearch, Redis, and Cassandra: short prefixes match nearby cells, so a `LIKE 'gbsuv%'` query gets you everything within ~5 km of a point.

`geohash_encode` builds a hash at a chosen precision (number of base-32 characters); `geohash_decode` returns the centre point of the encoded cell plus the cell's bounding box, so callers can decide between centre-only and bbox semantics.

Backed by [`github.com/mmcloughlin/geohash`](https://pkg.go.dev/github.com/mmcloughlin/geohash), the standard Go implementation, used by everything from Caddy to Tile38.
*/

package geographic

import (
	"context"
	_ "embed"
	"fmt"
	"math"
	"math/big"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/mmcloughlin/geohash"
)

const (
	geohashMinPrecision = 1
	geohashMaxPrecision = 12

	// geohashCornerLatMax / geohashCornerLonMax are the values the decoder reports for the upper edges of the "zzz…z" corner cell. The upstream library wraps `lat == 90` / `lon == 180` to the opposite corner, so we shrink the reported edges to the largest float strictly below that wraps cleanly. Determined empirically against `mmcloughlin/geohash`'s wrap threshold (~one ULP below 90 / 180); these values still encode to the same "z" prefix and are well below the float precision a real geographic measurement would carry.
	geohashCornerLatMax = 90.0 - 1e-13
	geohashCornerLonMax = 180.0 - 1e-13
)

// geohashAlphabet is the base-32 alphabet geohash uses ("Geohash-32"): the digits 0–9 plus the lowercase letters with `a`, `i`, `l`, `o` removed to avoid visual confusion. (This is a different exclusion set than Crockford-base32, which removes `i`, `l`, `o`, `u`.)
const geohashAlphabet = "0123456789bcdefghjkmnpqrstuvwxyz"

// ──────────────────────────────────────────────────────────────────────
// geohash_encode
// ──────────────────────────────────────────────────────────────────────

//go:embed descriptions/geohash_encode.md
var geohashEncodeDescription string

var _ function.Function = (*GeohashEncodeFunction)(nil)

type GeohashEncodeFunction struct{}

func NewGeohashEncodeFunction() function.Function { return &GeohashEncodeFunction{} }

func (f *GeohashEncodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "geohash_encode"
}

func (f *GeohashEncodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Encode (latitude, longitude) into a geohash string",
		MarkdownDescription: geohashEncodeDescription,
		Parameters: []function.Parameter{
			function.NumberParameter{Name: "latitude", Description: "Latitude in degrees, [-90, 90]."},
			function.NumberParameter{Name: "longitude", Description: "Longitude in degrees, [-180, 180]."},
			function.Int64Parameter{Name: "precision", Description: "Number of base-32 characters in the resulting hash; in [1, 12]."},
		},
		Return: function.StringReturn{},
	}
}

func (f *GeohashEncodeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var precision int64
	lat, lon, ferr := fetchLatLon(ctx, req, &precision)
	if ferr != nil {
		resp.Error = ferr
		return
	}
	if precision < geohashMinPrecision || precision > geohashMaxPrecision {
		resp.Error = function.NewArgumentFuncError(2, fmt.Sprintf("precision must be in [%d, %d]; received %d", geohashMinPrecision, geohashMaxPrecision, precision))
		return
	}
	/*
		The upstream encoder treats latitude as [-90, 90) and longitude as [-180, 180): the top of each range wraps to the opposite corner. The wrap is not confined to the exact endpoint. The single float one ULP below the edge (math.Nextafter(90, 0) / math.Nextafter(180, 0)) also wraps, encoding to the south pole / antimeridian rather than to a "z" cell, so we must reject everything at or above that penultimate float, not just the endpoint itself. Two ULPs below the edge is the largest value that encodes cleanly.

		The negative corners (-90 and -180) DO encode and round-trip correctly (they are the geometric origin of cell "0" / "h" / "8"), so this asymmetric rejection is intentional. The decoder shrinks its corner-cell bbox edges well below 90 / 180 so re-encoding `geohash_decode("zzz…z").lat_max` round-trips into the same cell rather than tripping this check.
	*/
	if lat >= math.Nextafter(90, 0) {
		resp.Error = function.NewArgumentFuncError(0, "latitude at or within one ULP of 90 wraps to the south pole under the standard geohash encoder; pass a value at least two ULPs below 90 (latitude == -90 is fine, only the upper bound wraps)")
		return
	}
	if lon >= math.Nextafter(180, 0) {
		resp.Error = function.NewArgumentFuncError(1, "longitude at or within one ULP of 180 wraps to the antimeridian under the standard geohash encoder; pass a value at least two ULPs below 180, or -180 (the same meridian), only the upper bound wraps")
		return
	}
	out := geohash.EncodeWithPrecision(lat, lon, uint(precision))
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &out))
}

// ──────────────────────────────────────────────────────────────────────
// geohash_decode
// ──────────────────────────────────────────────────────────────────────

//go:embed descriptions/geohash_decode.md
var geohashDecodeDescription string

var _ function.Function = (*GeohashDecodeFunction)(nil)

type GeohashDecodeFunction struct{}

func NewGeohashDecodeFunction() function.Function { return &GeohashDecodeFunction{} }

// geohashDecodeAttrs is the schema returned by geohash_decode. Centre + bbox in one object.
var geohashDecodeAttrs = map[string]attr.Type{
	"latitude":  types.NumberType,
	"longitude": types.NumberType,
	"lat_min":   types.NumberType,
	"lat_max":   types.NumberType,
	"lon_min":   types.NumberType,
	"lon_max":   types.NumberType,
}

func (f *GeohashDecodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "geohash_decode"
}

func (f *GeohashDecodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Decode a geohash into the centre point and bounding box of its cell",
		MarkdownDescription: geohashDecodeDescription,
		Parameters: []function.Parameter{
			function.StringParameter{Name: "code", Description: "Geohash string to decode."},
		},
		Return: function.ObjectReturn{AttributeTypes: geohashDecodeAttrs},
	}
}

func (f *GeohashDecodeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var code string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &code))
	if resp.Error != nil {
		return
	}
	lower := strings.ToLower(code)
	if lower == "" {
		resp.Error = function.NewArgumentFuncError(0, "code must not be empty")
		return
	}
	for i, r := range lower {
		if !strings.ContainsRune(geohashAlphabet, r) {
			resp.Error = function.NewArgumentFuncError(0, fmt.Sprintf("code contains invalid character %q at position %d (allowed: %s)", r, i, geohashAlphabet))
			return
		}
	}

	// Use DecodeCenter (box.Center()), not Decode (box.Round()): the latter returns a shortened representative point inside the cell, but the documentation and the returned bbox promise the cell centre.
	lat, lon := geohash.DecodeCenter(lower)
	box := geohash.BoundingBox(lower)
	// The corner cell ("zzz…z") reports its upper edges at exactly 90 / 180. Those values wrap if fed back into `geohash_encode`, so shrink them to the nearest representable float that the encoder still accepts. The shift is below any meaningful geographic resolution (~10 nm at 90°N) and keeps the bbox round-trippable.
	latMax := box.MaxLat
	if latMax >= 90 {
		latMax = geohashCornerLatMax
	}
	lonMax := box.MaxLng
	if lonMax >= 180 {
		lonMax = geohashCornerLonMax
	}
	out, diags := types.ObjectValue(geohashDecodeAttrs, map[string]attr.Value{
		"latitude":  types.NumberValue(big.NewFloat(lat)),
		"longitude": types.NumberValue(big.NewFloat(lon)),
		"lat_min":   types.NumberValue(big.NewFloat(box.MinLat)),
		"lat_max":   types.NumberValue(big.NewFloat(latMax)),
		"lon_min":   types.NumberValue(big.NewFloat(box.MinLng)),
		"lon_max":   types.NumberValue(big.NewFloat(lonMax)),
	})
	if diags.HasError() {
		resp.Error = function.NewFuncError("building geohash_decode result")
		return
	}
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &out))
}
