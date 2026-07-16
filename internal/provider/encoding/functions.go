// Package encoding provides byte-codec provider-defined functions: hex and base64 encode/decode. These fill gaps in Terraform core: there is no built-in hex decode, and core's `base64encode`/`base64decode` only speak standard, padded base64. The encoders take an options object for the RFC 4648 variants; the decoders are deliberately lenient so they accept whatever an encoder on the other side produced.
package encoding

import (
	"context"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
)

// Functions returns the encoding provider-defined functions registered by terraform-burnham.
func Functions() []func() function.Function {
	return []func() function.Function{
		NewHexEncodeFunction,
		NewHexDecodeFunction,
		NewBase64EncodeFunction,
		NewBase64DecodeFunction,
		NewBase32EncodeFunction,
		NewBase32DecodeFunction,
		NewURLEncodeFunction,
		NewURLDecodeFunction,
	}
}

// hasUnknown reports whether v holds an unknown value at any depth. Terraform only auto-defers a function call when a whole argument is unknown, so a known options object with an unknown field (e.g. `{ url_safe = <unknown> }`) reaches Run.
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

// unknownStringOptionResult sets an unknown string result and returns true when any option carries an unknown value. The input is always a plain string (core defers a wholly-unknown scalar argument), so only the options object can smuggle an unknown into Run; when it does, the selected alphabet/padding is not yet known, so an encode/decode would silently use the defaults and produce a concrete plan value that changes at apply. Returning unknown lets the value resolve then.
func unknownStringOptionResult(ctx context.Context, resp *function.RunResponse, opts []types.Dynamic) bool {
	for _, o := range opts {
		if hasUnknown(o) {
			resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, types.StringUnknown()))
			return true
		}
	}
	return false
}

// stripASCIIWhitespace removes spaces, tabs, CR and LF, the leniency every
// decoder in this package shares so wrapped/pretty-printed input round-trips.
func stripASCIIWhitespace(s string) string {
	return strings.Map(func(r rune) rune {
		switch r {
		case ' ', '\t', '\r', '\n':
			return -1
		}
		return r
	}, s)
}
