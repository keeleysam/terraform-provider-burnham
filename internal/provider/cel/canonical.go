package cel

import (
	"fmt"

	"github.com/google/cel-go/common/ast"
)

// The Path A canonical nodes are keyed by the syntax.proto expr_kind field names.
// const_expr and call_expr are handled in encode.go (call_expr shares encodeCall with the Path B `call`); the rest live here.

func (e *encoder) encodeIdentExpr(val any) (ast.Expr, error) {
	spec, ok := val.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("ident_expr must be an object with a name")
	}
	name, ok := spec["name"].(string)
	if !ok {
		return nil, fmt.Errorf("ident_expr.name must be a string")
	}
	return e.f.NewIdent(e.id(), name), nil
}

func (e *encoder) encodeSelectExpr(val any) (ast.Expr, error) {
	spec, ok := val.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("select_expr must be an object with operand and field")
	}
	operandRaw, ok := spec["operand"]
	if !ok {
		return nil, fmt.Errorf("select_expr requires an operand")
	}
	operand, err := e.encode(operandRaw)
	if err != nil {
		return nil, fmt.Errorf("select_expr.operand: %w", err)
	}
	field, ok := spec["field"].(string)
	if !ok {
		return nil, fmt.Errorf("select_expr.field must be a string")
	}
	if to, ok := spec["test_only"].(bool); ok && to {
		return e.f.NewPresenceTest(e.id(), operand, field), nil
	}
	return e.f.NewSelect(e.id(), operand, field), nil
}

func (e *encoder) encodeListExpr(val any) (ast.Expr, error) {
	spec, ok := val.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("list_expr must be an object with elements")
	}
	elems, ok := spec["elements"].([]any)
	if !ok {
		return nil, fmt.Errorf("list_expr.elements must be a list")
	}
	// optional_indices marks which elements are optional (CEL `[?x]`).
	if rawIdx, present := spec["optional_indices"]; present {
		idxList, ok := rawIdx.([]any)
		if !ok {
			return nil, fmt.Errorf("list_expr.optional_indices must be a list")
		}
		optional := make(map[int]bool, len(idxList))
		for _, v := range idxList {
			i, err := toInt64(v)
			if err != nil {
				return nil, fmt.Errorf("list_expr.optional_indices: %w", err)
			}
			optional[int(i)] = true
		}
		wrapped := make([]any, len(elems))
		for i, el := range elems {
			if optional[i] {
				wrapped[i] = map[string]any{"optional": el}
			} else {
				wrapped[i] = el
			}
		}
		return e.encodeList(wrapped)
	}
	return e.encodeList(elems)
}

func (e *encoder) encodeStructExpr(val any) (ast.Expr, error) {
	spec, ok := val.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("struct_expr must be an object")
	}
	msgName, _ := spec["message_name"].(string)
	rawEntries, ok := spec["entries"].([]any)
	if !ok {
		return nil, fmt.Errorf("struct_expr.entries must be a list")
	}
	// A CreateStruct is a message literal (field_key entries) or a map literal (map_key entries).
	// cel-go builds them with different constructors, so the entries must be homogeneous.
	entries := make([]ast.EntryExpr, 0, len(rawEntries))
	isMap := false
	for i, re := range rawEntries {
		em, ok := re.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("struct_expr entry %d must be an object", i)
		}
		optional, _ := em["optional_entry"].(bool)
		value, err := e.encode(em["value"])
		if err != nil {
			return nil, fmt.Errorf("struct_expr entry %d value: %w", i, err)
		}
		_, hasMapKey := em["map_key"]
		if fk, ok := em["field_key"].(string); ok {
			if isMap {
				return nil, fmt.Errorf("struct_expr entries must be all field_key or all map_key")
			}
			entries = append(entries, e.f.NewStructField(e.id(), fk, value, optional))
			continue
		}
		if hasMapKey {
			if i > 0 && !isMap {
				return nil, fmt.Errorf("struct_expr entries must be all field_key or all map_key")
			}
			isMap = true
			key, err := e.encode(em["map_key"])
			if err != nil {
				return nil, fmt.Errorf("struct_expr entry %d map_key: %w", i, err)
			}
			entries = append(entries, e.f.NewMapEntry(e.id(), key, value, optional))
			continue
		}
		return nil, fmt.Errorf("struct_expr entry %d needs field_key or map_key", i)
	}
	// Entry-less with no message_name is an empty map literal `{}` (a message would carry a message_name).
	if isMap || (len(entries) == 0 && msgName == "") {
		return e.f.NewMap(e.id(), entries), nil
	}
	if msgName == "" {
		return nil, fmt.Errorf("struct_expr with field_key entries requires message_name; a map literal uses map_key entries")
	}
	return e.f.NewStruct(e.id(), msgName, entries), nil
}
