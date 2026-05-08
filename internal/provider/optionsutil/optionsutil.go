/*
Helpers shared by every provider function that takes a single optional options object via the framework's `VariadicParameter: function.DynamicParameter` pattern.

Conventions baked into the helpers:

  - Optional argument lives at parameter index 1 (one positional argument followed by `...options`). Errors returned here always tag index 1.
  - Exactly zero or one options object is allowed. More is an error.
  - When the caller passes `null` or an unknown value where an object is expected, the helper rejects it.
  - Per-key validation (allowed keys, value coercion) stays at the call site, because the schema differs per function.

These helpers do not migrate the older `dataformat/` family, which has a different "ignore unknown keys silently" contract; mixing them would change semantics. They cover the newer `identifiers/`, `text/`, and any future package adopting the same explicit schema.
*/

package optionsutil

import (
	"fmt"
	"math/big"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// SingleOptionsObject pulls the (zero-or-one) options object out of a `VariadicParameter` slice and returns its attribute map. Returns `(nil, nil)` when the caller passed no options — the caller should fall through to its defaults. `hint` is a snippet shown in the error when the caller passes something that isn't an object literal: typically a worked example like `"{ size = 10 }"`.
func SingleOptionsObject(opts []types.Dynamic, hint string) (map[string]attr.Value, *function.FuncError) {
	if len(opts) == 0 {
		return nil, nil
	}
	if len(opts) > 1 {
		return nil, function.NewArgumentFuncError(1, "at most one options argument may be provided")
	}
	obj, ok := opts[0].UnderlyingValue().(basetypes.ObjectValue)
	if !ok || obj.IsNull() || obj.IsUnknown() {
		return nil, function.NewArgumentFuncError(1, fmt.Sprintf("options must be an object literal, e.g. %s", hint))
	}
	return obj.Attributes(), nil
}

// NumberAttrToInt converts a Terraform Number attribute (carries a `*big.Float` internally) into a Go int. Errors when the value is null/unknown, non-integral, or out of int range. Lossy conversions never happen — Terraform numbers preserve the integer-ness of their input.
func NumberAttrToInt(v attr.Value) (int, error) {
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
