// Internal helpers shared across the text family. Kept package-private so the rest of the provider doesn't accidentally couple to text-family decisions.

package text

import (
	"fmt"
	"math/big"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// numberAttrToInt converts a Terraform Number attr.Value (carries a *big.Float internally) into a Go int. Errors when the value is null/unknown, non-integral, or out of int range.
func numberAttrToInt(v attr.Value) (int, error) {
	num, ok := v.(basetypes.NumberValue)
	if !ok {
		return 0, fmt.Errorf("expected a number, got %T", v)
	}
	if num.IsNull() || num.IsUnknown() {
		return 0, fmt.Errorf("value is null or unknown")
	}
	bi, accuracy := num.ValueBigFloat().Int(nil)
	if accuracy != big.Exact {
		return 0, fmt.Errorf("not a whole number")
	}
	if !bi.IsInt64() {
		return 0, fmt.Errorf("out of int range")
	}
	return int(bi.Int64()), nil
}
