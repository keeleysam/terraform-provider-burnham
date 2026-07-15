package cedar

import (
	"encoding/json"
	"fmt"
	"math/big"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// nodeToAttr converts the Go node tree produced by Decode (and Evaluate) into a Terraform attr.Value so cedardecode/cedarevaluate can return it.
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
	case float64:
		return types.NumberValue(big.NewFloat(v)), nil
	case json.Number:
		// Keep integers exact (Cedar Long is 64-bit); fall back to a wide float only for genuine fractionals.
		if i, err := v.Int64(); err == nil {
			return types.NumberValue(new(big.Float).SetInt64(i)), nil
		}
		f, _, err := big.ParseFloat(v.String(), 10, 512, big.ToNearestEven)
		if err != nil {
			return nil, fmt.Errorf("invalid number %q: %w", v.String(), err)
		}
		return types.NumberValue(f), nil
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
