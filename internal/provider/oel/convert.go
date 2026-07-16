package oel

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

const (
	// oelMaxDepth caps recursion when converting the input tree. Real expressions nest a few dozen levels at most; this guards against adversarial input.
	oelMaxDepth = 1024
	// oelMaxNodes caps the total node count traversed in a single conversion.
	oelMaxNodes = 1_000_000
	// oelMaxInputBytes caps the length of an OEL string argument to the string-input functions.
	oelMaxInputBytes = 16 << 20 // 16 MiB
	// oelMaxParseDepth bounds how deeply grouping tokens may nest in a raw OEL string before it reaches the recursive-descent parser, which recurses once per nesting level and overflows the goroutine stack (crashing the process) on pathological input well under the byte cap. Real expressions nest a few dozen levels; past this cap the string functions return a normal parse error so oelvalidate returns false rather than aborting the plan.
	oelMaxParseDepth = 2000
)

// checkNestingDepth returns an error when the grouping tokens in s nest deeper than oelMaxParseDepth. It runs before the parser so adversarial nesting yields an ordinary error instead of a stack-overflow crash. Brackets inside string literals (single- or double-quoted, with backslash escapes) do not count, matching the lexer.
func checkNestingDepth(s string) error {
	depth := 0
	for i := 0; i < len(s); i++ {
		switch c := s[i]; c {
		case '"', '\'':
			// Skip the string literal, honoring backslash escapes, so its contents never count toward nesting depth.
			quote := c
			i++
			for i < len(s) && s[i] != quote {
				if s[i] == '\\' {
					i++
				}
				i++
			}
		case '(', '[', '{':
			depth++
			if depth > oelMaxParseDepth {
				return fmt.Errorf("expression nesting exceeds maximum supported depth of %d", oelMaxParseDepth)
			}
		case ')', ']', '}':
			if depth > 0 {
				depth--
			}
		}
	}
	return nil
}

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
	if depth >= oelMaxDepth {
		return nil, fmt.Errorf("input exceeds maximum supported nesting depth of %d", oelMaxDepth)
	}
	*nodes++
	if *nodes > oelMaxNodes {
		return nil, fmt.Errorf("input exceeds maximum supported node count of %d", oelMaxNodes)
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

// toGoValue converts a Terraform attr.Value into native Go types (bool, string, int, float64, nil, []any, map[string]any) for the evaluator's context, which compares against parsed literals that are int/float64 rather than json.Number. Numbers that are integral become int, matching how the parser represents an integer literal.
func toGoValue(v attr.Value) (any, error) {
	if v == nil || v.IsNull() {
		return nil, nil
	}
	switch val := v.(type) {
	case basetypes.BoolValue:
		return val.ValueBool(), nil
	case basetypes.StringValue:
		return val.ValueString(), nil
	case basetypes.NumberValue:
		bf := val.ValueBigFloat()
		if bf.IsInt() {
			i, _ := bf.Int64()
			return int(i), nil
		}
		f, _ := bf.Float64()
		return f, nil
	case basetypes.TupleValue:
		return elementsToGo(val.Elements())
	case basetypes.ListValue:
		return elementsToGo(val.Elements())
	case basetypes.SetValue:
		return elementsToGo(val.Elements())
	case basetypes.ObjectValue:
		return attributesToGo(val.Attributes())
	case basetypes.MapValue:
		return attributesToGo(val.Elements())
	case basetypes.DynamicValue:
		return toGoValue(val.UnderlyingValue())
	default:
		return nil, fmt.Errorf("unsupported Terraform value type %T", v)
	}
}

func elementsToGo(elems []attr.Value) (any, error) {
	out := make([]any, len(elems))
	for i, e := range elems {
		conv, err := toGoValue(e)
		if err != nil {
			return nil, fmt.Errorf("index %d: %w", i, err)
		}
		out[i] = conv
	}
	return out, nil
}

func attributesToGo(attrs map[string]attr.Value) (map[string]any, error) {
	out := make(map[string]any, len(attrs))
	for k, av := range attrs {
		conv, err := toGoValue(av)
		if err != nil {
			return nil, fmt.Errorf("key %q: %w", k, err)
		}
		out[k] = conv
	}
	return out, nil
}
