// Internal helpers for the geographic family.

package geographic

import (
	"context"
	"fmt"
	"math/big"

	"github.com/hashicorp/terraform-plugin-framework/function"
)

// bigFloatToFloat64 converts a Terraform Number (`*big.Float`) to a Go float64. Errors when the value is infinite. Lat/lon coordinates are well within float64 precision so the float64 narrowing is lossless for any meaningful input; we discard the accuracy result deliberately.
func bigFloatToFloat64(name string, v *big.Float) (float64, *function.FuncError) {
	if v.IsInf() {
		return 0, function.NewFuncError(fmt.Sprintf("%s must be a finite number; received infinity", name))
	}
	f, _ := v.Float64()
	return f, nil
}

// validateLatLon enforces the standard ±90 / ±180 ranges for geographic coordinates. Most encoders silently fold out-of-range values (geohash wraps longitude, OLC clamps), so we surface the error explicitly to catch typos that would otherwise produce a "valid but wrong" code.
func validateLatLon(lat, lon float64) *function.FuncError {
	if lat < -90 || lat > 90 {
		return function.NewFuncError(fmt.Sprintf("latitude must be in [-90, 90]; received %g", lat))
	}
	if lon < -180 || lon > 180 {
		return function.NewFuncError(fmt.Sprintf("longitude must be in [-180, 180]; received %g", lon))
	}
	return nil
}

// fetchLatLon pulls the (latitude, longitude) leading number arguments out of a function request, converts both to float64, and validates the standard ±90/±180 ranges. Centralised so geohash_encode and pluscode_encode (and any future lat/lon-keyed function) share one parsing contract.
func fetchLatLon(ctx context.Context, req function.RunRequest, extras ...any) (lat, lon float64, ferr *function.FuncError) {
	var latBF, lonBF *big.Float
	args := append([]any{&latBF, &lonBF}, extras...)
	if err := req.Arguments.Get(ctx, args...); err != nil {
		return 0, 0, err
	}
	lat, ferr = bigFloatToFloat64("latitude", latBF)
	if ferr != nil {
		return 0, 0, ferr
	}
	lon, ferr = bigFloatToFloat64("longitude", lonBF)
	if ferr != nil {
		return 0, 0, ferr
	}
	if ferr = validateLatLon(lat, lon); ferr != nil {
		return 0, 0, ferr
	}
	return lat, lon, nil
}
