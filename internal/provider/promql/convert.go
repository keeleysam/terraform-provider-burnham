package promql

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

const (
	// promqlMaxDepth caps recursion when converting the input tree. Real expressions nest a few dozen levels at most; this guards against adversarial input.
	promqlMaxDepth = 1024
	// promqlMaxNodes caps the total node count traversed in a single conversion.
	promqlMaxNodes = 1_000_000
	// promqlMaxInputBytes caps the length of a PromQL query string argument to the string-input functions.
	promqlMaxInputBytes = 16 << 20 // 16 MiB
)

// errUnknownValue signals that a value in the input tree is unknown at plan time. A plan-time function returns an unknown result in that case rather than failing, so the value can resolve at apply.
var errUnknownValue = errors.New("value is unknown")

// hasUnknown reports whether v holds an unknown value at any depth. Terraform only auto-defers a function call when a whole argument is unknown, so a known container with an unknown nested value reaches Run; the functions check this and return an unknown result rather than erroring, so the value resolves at apply.
func hasUnknown(v attr.Value) bool {
	if v == nil {
		return false
	}
	if v.IsUnknown() {
		return true
	}
	switch val := v.(type) {
	case basetypes.DynamicValue:
		return hasUnknown(val.UnderlyingValue())
	case basetypes.TupleValue:
		return elementsHaveUnknown(val.Elements())
	case basetypes.ListValue:
		return elementsHaveUnknown(val.Elements())
	case basetypes.SetValue:
		return elementsHaveUnknown(val.Elements())
	case basetypes.ObjectValue:
		return attributesHaveUnknown(val.Attributes())
	case basetypes.MapValue:
		return attributesHaveUnknown(val.Elements())
	}
	return false
}

func elementsHaveUnknown(elems []attr.Value) bool {
	for _, e := range elems {
		if hasUnknown(e) {
			return true
		}
	}
	return false
}

func attributesHaveUnknown(attrs map[string]attr.Value) bool {
	for _, a := range attrs {
		if hasUnknown(a) {
			return true
		}
	}
	return false
}

// terraformToNode converts a Terraform attr.Value into the JSON-ish node tree the encoder consumes: nil, bool, string, json.Number, []any, map[string]any. Numbers stay json.Number so the encoder can distinguish int from double.
func terraformToNode(v attr.Value) (any, error) {
	nodes := 0
	return terraformToNodeImpl(v, 0, &nodes)
}

func terraformToNodeImpl(v attr.Value, depth int, nodes *int) (any, error) {
	if depth >= promqlMaxDepth {
		return nil, fmt.Errorf("input exceeds maximum supported nesting depth of %d", promqlMaxDepth)
	}
	*nodes++
	if *nodes > promqlMaxNodes {
		return nil, fmt.Errorf("input exceeds maximum supported node count of %d", promqlMaxNodes)
	}
	if v == nil || v.IsNull() {
		return nil, nil
	}
	if v.IsUnknown() {
		return nil, errUnknownValue
	}

	switch val := v.(type) {
	case basetypes.BoolValue:
		return val.ValueBool(), nil
	case basetypes.StringValue:
		return val.ValueString(), nil
	case basetypes.NumberValue:
		return json.Number(val.ValueBigFloat().Text('f', -1)), nil
	case basetypes.TupleValue:
		return elementsToNodes(val.Elements(), depth, nodes)
	case basetypes.ListValue:
		return elementsToNodes(val.Elements(), depth, nodes)
	case basetypes.SetValue:
		return elementsToNodes(val.Elements(), depth, nodes)
	case basetypes.ObjectValue:
		return attributesToNodes(val.Attributes(), depth, nodes)
	case basetypes.MapValue:
		return attributesToNodes(val.Elements(), depth, nodes)
	case basetypes.DynamicValue:
		return terraformToNodeImpl(val.UnderlyingValue(), depth, nodes)
	default:
		return nil, fmt.Errorf("unsupported Terraform value type %T", v)
	}
}

func elementsToNodes(elements []attr.Value, depth int, nodes *int) (any, error) {
	out := make([]any, len(elements))
	for i, elem := range elements {
		conv, err := terraformToNodeImpl(elem, depth+1, nodes)
		if err != nil {
			return nil, fmt.Errorf("index %d: %w", i, err)
		}
		out[i] = conv
	}
	return out, nil
}

func attributesToNodes(attrs map[string]attr.Value, depth int, nodes *int) (any, error) {
	out := make(map[string]any, len(attrs))
	for k, av := range attrs {
		conv, err := terraformToNodeImpl(av, depth+1, nodes)
		if err != nil {
			return nil, fmt.Errorf("key %q: %w", k, err)
		}
		out[k] = conv
	}
	return out, nil
}
