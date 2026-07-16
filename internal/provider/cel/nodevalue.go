package cel

import (
	"fmt"
	"math"
	"math/big"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// nodeToAttr converts the Go node tree produced by Decode into a Terraform attr.Value so celdecode can return it.
// Objects become ObjectValue, lists become TupleValue (heterogeneous), and scalars map to their framework types.
func nodeToAttr(node any) (attr.Value, error) {
	switch v := node.(type) {
	case nil:
		return types.DynamicNull(), nil
	case bool:
		return types.BoolValue(v), nil
	case string:
		return types.StringValue(v), nil
	case int:
		return types.NumberValue(new(big.Float).SetInt64(int64(v))), nil
	case int64:
		return types.NumberValue(new(big.Float).SetInt64(v)), nil
	case uint64:
		return types.NumberValue(new(big.Float).SetUint64(v)), nil
	case float64:
		// big.Float (and thus a Terraform NumberValue) cannot represent non-finite doubles: big.NewFloat(NaN) panics and infinities emit an invalid plan value. Reject them cleanly.
		if math.IsNaN(v) || math.IsInf(v, 0) {
			return nil, fmt.Errorf("cannot represent non-finite number %v as a Terraform value", v)
		}
		return types.NumberValue(big.NewFloat(v)), nil
	case []any:
		elems := make([]attr.Value, len(v))
		elemTypes := make([]attr.Type, len(v))
		for i, item := range v {
			av, err := nodeToAttr(item)
			if err != nil {
				return nil, err
			}
			elems[i] = av
			elemTypes[i] = av.Type(nil)
		}
		tv, diags := types.TupleValue(elemTypes, elems)
		if diags.HasError() {
			return nil, fmt.Errorf("%s", diags.Errors()[0].Detail())
		}
		return tv, nil
	case map[string]any:
		attrs := make(map[string]attr.Value, len(v))
		attrTypes := make(map[string]attr.Type, len(v))
		for k, item := range v {
			av, err := nodeToAttr(item)
			if err != nil {
				return nil, err
			}
			attrs[k] = av
			attrTypes[k] = av.Type(nil)
		}
		ov, diags := types.ObjectValue(attrTypes, attrs)
		if diags.HasError() {
			return nil, fmt.Errorf("%s", diags.Errors()[0].Detail())
		}
		return ov, nil
	default:
		return nil, fmt.Errorf("cannot represent node of type %T", node)
	}
}
