package geographic

import (
	"context"
	"math"
	"math/big"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// encodeGeohash runs geohash_encode in-process and returns the resulting code
// together with any function error (nil on success).
func encodeGeohash(t *testing.T, lat, lon float64, precision int64) (string, *function.FuncError) {
	t.Helper()
	f := &GeohashEncodeFunction{}
	args := function.NewArgumentsData([]attr.Value{
		types.NumberValue(big.NewFloat(lat)),
		types.NumberValue(big.NewFloat(lon)),
		types.Int64Value(precision),
	})
	req := function.RunRequest{Arguments: args}
	resp := &function.RunResponse{Result: function.NewResultData(types.StringNull())}
	f.Run(context.Background(), req, resp)
	if resp.Error != nil {
		return "", resp.Error
	}
	code, _ := resp.Result.Value().(types.String)
	return code.ValueString(), nil
}

// TestGeohashEncode_RejectsNearPoleWrap guards an off-by-one-ULP bug in the
// bound check: the upstream encoder wraps not only latitude == 90 / longitude
// == 180 but also the single float one ULP below each (math.Nextafter(90, 0) /
// math.Nextafter(180, 0)), which silently encode to the opposite corner (the
// south pole / the antimeridian). geohash_encode must reject those inputs
// rather than emit a wrong cell, and the two-ULP-below value must still encode.
func TestGeohashEncode_RejectsNearPoleWrap(t *testing.T) {
	rejects := []struct {
		name     string
		lat, lon float64
	}{
		{"lat one ULP below 90", math.Nextafter(90, 0), 0},
		{"lon one ULP below 180", 0, math.Nextafter(180, 0)},
	}
	for _, tc := range rejects {
		if _, ferr := encodeGeohash(t, tc.lat, tc.lon, 12); ferr == nil {
			t.Errorf("%s: geohash_encode(%.20g, %.20g) accepted a value that wraps to the opposite corner; want rejection", tc.name, tc.lat, tc.lon)
		}
	}

	// Two ULPs below the edge is past the wrap threshold and must encode.
	accepts := []struct {
		name     string
		lat, lon float64
	}{
		{"lat two ULPs below 90", math.Nextafter(math.Nextafter(90, 0), 0), 0},
		{"lon two ULPs below 180", 0, math.Nextafter(math.Nextafter(180, 0), 0)},
	}
	for _, tc := range accepts {
		if _, ferr := encodeGeohash(t, tc.lat, tc.lon, 12); ferr != nil {
			t.Errorf("%s: geohash_encode(%.20g, %.20g) rejected a safe value: %s", tc.name, tc.lat, tc.lon, ferr.Text)
		}
	}
}

// decodeGeohash runs geohash_decode in-process and returns the reported
// centre latitude / longitude.
func decodeGeohash(t *testing.T, code string) (lat, lon float64) {
	t.Helper()
	f := &GeohashDecodeFunction{}
	args := function.NewArgumentsData([]attr.Value{types.StringValue(code)})
	req := function.RunRequest{Arguments: args}
	resp := &function.RunResponse{Result: function.NewResultData(types.ObjectNull(geohashDecodeAttrs))}
	f.Run(context.Background(), req, resp)
	if resp.Error != nil {
		t.Fatalf("geohash_decode(%q) errored: %s", code, resp.Error.Text)
	}
	obj, ok := resp.Result.Value().(*types.Object)
	if !ok {
		t.Fatalf("expected Object result, got %T", resp.Result.Value())
	}
	attrs := obj.Attributes()
	latF, _ := attrs["latitude"].(types.Number).ValueBigFloat().Float64()
	lonF, _ := attrs["longitude"].(types.Number).ValueBigFloat().Float64()
	return latF, lonF
}

// TestGeohashDecode_ReturnsCellCentre asserts that geohash_decode returns the
// geometric centre of the cell (consistent with the returned bbox and the
// documentation), not the library's representative rounded point.
func TestGeohashDecode_ReturnsCellCentre(t *testing.T) {
	cases := []struct {
		code             string
		wantLat, wantLon float64
	}{
		// Cell "s" spans lat/lon [0, 45]; its centre is (22.5, 22.5).
		{"s", 22.5, 22.5},
		// Cell "ezs42" centres on (42.60498, -5.60302).
		{"ezs42", 42.60498, -5.60302},
	}
	const tol = 1e-4
	for _, tc := range cases {
		lat, lon := decodeGeohash(t, tc.code)
		if math.Abs(lat-tc.wantLat) > tol || math.Abs(lon-tc.wantLon) > tol {
			t.Errorf("geohash_decode(%q) = (%.5f, %.5f); want centre (%.5f, %.5f)", tc.code, lat, lon, tc.wantLat, tc.wantLon)
		}
	}
}
