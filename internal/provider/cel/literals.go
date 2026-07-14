package cel

import (
	"encoding/json"
	"fmt"
	"math"
	"math/big"
	"sort"
	"strconv"

	"github.com/google/cel-go/common/ast"
	"github.com/google/cel-go/common/types"
)

// constantKinds are the CEL Constant oneof field names (cel/expr/syntax.proto).
var constantKinds = map[string]bool{
	"null_value": true, "bool_value": true, "int64_value": true,
	"uint64_value": true, "double_value": true, "string_value": true, "bytes_value": true,
}

func isConstantKind(k string) bool { return constantKinds[k] }

// literalize turns plain HCL data into CEL literals recursively.
// Objects become map literals (not nodes); a single-key object whose key is a CEL constant kind is a typed scalar literal.
// Map keys are sorted so output is deterministic.
func (e *encoder) literalize(val any) (ast.Expr, error) {
	switch v := val.(type) {
	case map[string]any:
		if len(v) == 1 {
			for k, inner := range v {
				if isConstantKind(k) {
					return e.typedConst(k, inner)
				}
			}
		}
		keys := make([]string, 0, len(v))
		for k := range v {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		entries := make([]ast.EntryExpr, 0, len(v))
		for _, k := range keys {
			value, err := e.literalize(v[k])
			if err != nil {
				return nil, fmt.Errorf("map key %q: %w", k, err)
			}
			key := e.f.NewLiteral(e.id(), types.String(k))
			entries = append(entries, e.f.NewMapEntry(e.id(), key, value, false))
		}
		return e.f.NewMap(e.id(), entries), nil
	case []any:
		elems := make([]ast.Expr, len(v))
		for i, item := range v {
			el, err := e.literalize(item)
			if err != nil {
				return nil, fmt.Errorf("list element %d: %w", i, err)
			}
			elems[i] = el
		}
		return e.f.NewList(e.id(), elems, nil), nil
	default:
		return e.encode(val) // scalars: nil / bool / string / number
	}
}

// typedConst builds a literal of an explicit CEL constant kind.
func (e *encoder) typedConst(kind string, v any) (ast.Expr, error) {
	switch kind {
	case "null_value":
		return e.f.NewLiteral(e.id(), types.NullValue), nil
	case "bool_value":
		b, ok := v.(bool)
		if !ok {
			return nil, fmt.Errorf("bool_value must be a bool")
		}
		return e.f.NewLiteral(e.id(), types.Bool(b)), nil
	case "string_value":
		s, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("string_value must be a string")
		}
		return e.f.NewLiteral(e.id(), types.String(s)), nil
	case "bytes_value":
		s, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("bytes_value must be a string")
		}
		return e.f.NewLiteral(e.id(), types.Bytes([]byte(s))), nil
	case "int64_value":
		i, err := toInt64(v)
		if err != nil {
			return nil, fmt.Errorf("int64_value: %w", err)
		}
		return e.f.NewLiteral(e.id(), types.Int(i)), nil
	case "uint64_value":
		u, err := toUint64(v)
		if err != nil {
			return nil, fmt.Errorf("uint64_value: %w", err)
		}
		return e.f.NewLiteral(e.id(), types.Uint(u)), nil
	case "double_value":
		f, err := toFloat64(v)
		if err != nil {
			return nil, fmt.Errorf("double_value: %w", err)
		}
		return e.f.NewLiteral(e.id(), types.Double(f)), nil
	}
	return nil, fmt.Errorf("unsupported constant kind %q", kind)
}

// encodeStruct builds a message/struct construction (Type{field: value}).
// Field values are full nodes (they may be references or expressions).
// Fields are sorted for deterministic output.
func (e *encoder) encodeStruct(val any) (ast.Expr, error) {
	spec, ok := val.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("struct must be an object with message_name and fields")
	}
	for k := range spec {
		if k != "message_name" && k != "fields" {
			return nil, fmt.Errorf("unknown struct key %q; expected message_name or fields", k)
		}
	}
	msgName, _ := spec["message_name"].(string)
	if msgName == "" {
		return nil, fmt.Errorf("struct.message_name is required; for a map literal use { const = { ... } }")
	}
	fieldsRaw, ok := spec["fields"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("struct.fields must be an object")
	}
	names := make([]string, 0, len(fieldsRaw))
	for n := range fieldsRaw {
		names = append(names, n)
	}
	sort.Strings(names)
	entries := make([]ast.EntryExpr, 0, len(fieldsRaw))
	for _, n := range names {
		value, err := e.encode(fieldsRaw[n])
		if err != nil {
			return nil, fmt.Errorf("struct field %q: %w", n, err)
		}
		entries = append(entries, e.f.NewStructField(e.id(), n, value, false))
	}
	return e.f.NewStruct(e.id(), msgName, entries), nil
}

func toInt64(v any) (int64, error) {
	switch n := v.(type) {
	case int:
		return int64(n), nil
	case int64:
		return n, nil
	case float64:
		return int64(n), nil
	case json.Number:
		return n.Int64()
	}
	return 0, fmt.Errorf("expected an integer, got %T", v)
}

func toUint64(v any) (uint64, error) {
	switch n := v.(type) {
	case int:
		if n < 0 {
			return 0, fmt.Errorf("uint value must be non-negative")
		}
		return uint64(n), nil
	case int64:
		if n < 0 {
			return 0, fmt.Errorf("uint value must be non-negative")
		}
		return uint64(n), nil
	case uint64:
		return n, nil
	case float64:
		if n < 0 || n != math.Trunc(n) {
			return 0, fmt.Errorf("uint value must be a non-negative integer")
		}
		return uint64(n), nil
	case json.Number:
		// ParseUint covers the full [0, 2^64-1] range that Int64 cannot.
		return strconv.ParseUint(n.String(), 10, 64)
	}
	return 0, fmt.Errorf("expected a non-negative integer, got %T", v)
}

func toFloat64(v any) (float64, error) {
	switch n := v.(type) {
	case int:
		return float64(n), nil
	case int64:
		return float64(n), nil
	case float64:
		return n, nil
	case json.Number:
		bf, _, err := big.ParseFloat(n.String(), 10, 512, big.ToNearestEven)
		if err != nil {
			return 0, err
		}
		f, _ := bf.Float64()
		if math.IsInf(f, 0) {
			return 0, fmt.Errorf("value %s is out of range for a CEL double", n.String())
		}
		return f, nil
	}
	return 0, fmt.Errorf("expected a number, got %T", v)
}
