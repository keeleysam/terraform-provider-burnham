package geographic

import (
	"context"
	"math/big"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// encodePluscode runs pluscode_encode in-process and returns the resulting code
// together with any function error (nil on success).
func encodePluscode(t *testing.T, lat, lon float64, length int64) (string, *function.FuncError) {
	t.Helper()
	f := &PluscodeEncodeFunction{}
	args := function.NewArgumentsData([]attr.Value{
		types.NumberValue(big.NewFloat(lat)),
		types.NumberValue(big.NewFloat(lon)),
		types.Int64Value(length),
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

// decodePluscode runs pluscode_decode in-process and returns any function error
// (nil on success).
func decodePluscode(t *testing.T, code string) *function.FuncError {
	t.Helper()
	f := &PluscodeDecodeFunction{}
	args := function.NewArgumentsData([]attr.Value{types.StringValue(code)})
	req := function.RunRequest{Arguments: args}
	resp := &function.RunResponse{Result: function.NewResultData(types.ObjectNull(pluscodeDecodeAttrs))}
	f.Run(context.Background(), req, resp)
	return resp.Error
}

// TestPluscodeEncode_AntimeridianRoundTrips guards an asymmetric wrap in the
// upstream OLC encoder: longitude +180 encodes to a decodable code, but the
// documented lower bound -180 (the same meridian) mapped to an out-of-range
// integer and produced a code that pluscode_decode rejects with "longitude
// outside range". pluscode_encode must emit a code that pluscode_decode
// accepts across the whole documented [-180, 180] range.
func TestPluscodeEncode_AntimeridianRoundTrips(t *testing.T) {
	for _, lon := range []float64{-180, 180} {
		code, ferr := encodePluscode(t, 0, lon, 10)
		if ferr != nil {
			t.Fatalf("pluscode_encode(0, %g, 10) errored: %s", lon, ferr.Text)
		}
		if derr := decodePluscode(t, code); derr != nil {
			t.Errorf("pluscode_encode(0, %g, 10) = %q, which pluscode_decode rejects: %s", lon, code, derr.Text)
		}
	}
}
