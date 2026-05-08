/*
Geohash — interleaved-bits geocoding.

Geohash encodes a (lat, lon) pair into a base-32 string where each successive character refines the cell ten-fold (alternating between latitude and longitude). It's the de-facto format for spatial indexing in things like Elasticsearch, Redis, and Cassandra: short prefixes match nearby cells, so a `LIKE 'gbsuv%'` query gets you everything within ~5 km of a point.

`geohash_encode` builds a hash at a chosen precision (number of base-32 characters); `geohash_decode` returns the centre point of the encoded cell plus the cell's bounding box, so callers can decide between centre-only and bbox semantics.

Backed by [`github.com/mmcloughlin/geohash`](https://pkg.go.dev/github.com/mmcloughlin/geohash) — the standard Go implementation, used by everything from Caddy to Tile38.
*/

package geographic

import (
	"context"
	"fmt"
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
)

// geohashAlphabet is the base-32 alphabet geohash uses ("Geohash-32"): the digits 0–9 plus the lowercase letters with `a`, `i`, `l`, `o` removed to avoid visual confusion. (This is a different exclusion set than Crockford-base32, which removes `i`, `l`, `o`, `u`.)
const geohashAlphabet = "0123456789bcdefghjkmnpqrstuvwxyz"

// ──────────────────────────────────────────────────────────────────────
// geohash_encode
// ──────────────────────────────────────────────────────────────────────

var _ function.Function = (*GeohashEncodeFunction)(nil)

type GeohashEncodeFunction struct{}

func NewGeohashEncodeFunction() function.Function { return &GeohashEncodeFunction{} }

func (f *GeohashEncodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "geohash_encode"
}

func (f *GeohashEncodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Encode (latitude, longitude) into a geohash string",
		MarkdownDescription: "Returns the [geohash](https://en.wikipedia.org/wiki/Geohash) of the given `(latitude, longitude)` at the requested `precision` (number of base-32 characters). Higher precision = smaller cell. Approximate cell side at the equator (worst case):\n\n| precision | cell side |\n| ---: | --- |\n| 1 | ~5,000 km |\n| 3 | ~156 km |\n| 5 | ~4.9 km |\n| 7 | ~152 m |\n| 9 | ~4.8 m |\n| 12 | ~3.7 cm |\n\n`precision` must be in `[1, 12]`. `latitude` must be in `[-90, 90]` and `longitude` in `[-180, 180]`.\n\n```\ngeohash_encode(37.7749, -122.4194, 7)\n→ \"9q8yyk8\"   (≈ Civic Center, San Francisco)\n```",
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
	out := geohash.EncodeWithPrecision(lat, lon, uint(precision))
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &out))
}

// ──────────────────────────────────────────────────────────────────────
// geohash_decode
// ──────────────────────────────────────────────────────────────────────

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
		Summary: "Decode a geohash into the centre point and bounding box of its cell",
		MarkdownDescription: "Parses `hash` and returns:\n\n" +
			"- `latitude` / `longitude` — the centre of the cell, in degrees.\n" +
			"- `lat_min` / `lat_max` / `lon_min` / `lon_max` — the cell's bounding box (the points the hash *might* have come from).\n\n" +
			"`hash` is case-insensitive but must use the standard geohash alphabet `0-9 b-z` minus `a i l o`. Errors on any other character.",
		Parameters: []function.Parameter{
			function.StringParameter{Name: "hash", Description: "Geohash string to decode."},
		},
		Return: function.ObjectReturn{AttributeTypes: geohashDecodeAttrs},
	}
}

func (f *GeohashDecodeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var hash string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &hash))
	if resp.Error != nil {
		return
	}
	lower := strings.ToLower(hash)
	if lower == "" {
		resp.Error = function.NewArgumentFuncError(0, "hash must not be empty")
		return
	}
	for i, r := range lower {
		if !strings.ContainsRune(geohashAlphabet, r) {
			resp.Error = function.NewArgumentFuncError(0, fmt.Sprintf("hash contains invalid character %q at position %d (allowed: %s)", r, i, geohashAlphabet))
			return
		}
	}

	lat, lon := geohash.Decode(lower)
	box := geohash.BoundingBox(lower)
	out, diags := types.ObjectValue(geohashDecodeAttrs, map[string]attr.Value{
		"latitude":  types.NumberValue(big.NewFloat(lat)),
		"longitude": types.NumberValue(big.NewFloat(lon)),
		"lat_min":   types.NumberValue(big.NewFloat(box.MinLat)),
		"lat_max":   types.NumberValue(big.NewFloat(box.MaxLat)),
		"lon_min":   types.NumberValue(big.NewFloat(box.MinLng)),
		"lon_max":   types.NumberValue(big.NewFloat(box.MaxLng)),
	})
	if diags.HasError() {
		resp.Error = function.NewFuncError("building geohash_decode result")
		return
	}
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &out))
}
