package geographic

import (
	"context"
	"math"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

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
