/*
Parse a delimited key/value string into a map(string), robustly.

Replaces the fragile HCL idiom `{ for p in split(",", s) : split("=", p)[0] => split("=", p)[1] }`, which breaks the moment a value contains the key/value separator, carries surrounding whitespace, or is quoted to protect a literal separator. `parse_kv` splits each pair on the first key/value separator only, trims whitespace, and is quote-aware by default so a separator inside `"..."` (or `'...'`) is treated as literal.
*/

package text

import (
	"context"
	_ "embed"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/keeleysam/terraform-burnham/internal/provider/optionsutil"
)

var _ function.Function = (*ParseKVFunction)(nil)

//go:embed descriptions/parse_kv.md
var parseKVDescription string

type ParseKVFunction struct{}

func NewParseKVFunction() function.Function { return &ParseKVFunction{} }

func (f *ParseKVFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "parse_kv"
}

func (f *ParseKVFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Parse a delimited key/value string into a map(string)",
		MarkdownDescription: parseKVDescription,
		Parameters: []function.Parameter{
			function.StringParameter{Name: "s", Description: "The delimited key/value string to parse."},
		},
		VariadicParameter: function.DynamicParameter{
			Name:        "options",
			Description: "Optional options object: { pair_sep = string, kv_sep = string, trim = bool, unquote = bool }. At most one allowed.",
		},
		Return: function.MapReturn{ElementType: types.StringType},
	}
}

// parseKVOpts holds the resolved options for a single parse.
type parseKVOpts struct {
	pairSep string
	kvSep   string
	trim    bool
	unquote bool
}

func parseParseKVOptions(opts []types.Dynamic) (parseKVOpts, *function.FuncError) {
	out := parseKVOpts{pairSep: ",", kvSep: "=", trim: true, unquote: true}
	attrs, ferr := optionsutil.SingleOptionsObject(opts, "{ pair_sep = \";\", kv_sep = \":\" }")
	if ferr != nil {
		return out, ferr
	}
	for k, val := range attrs {
		switch k {
		case "pair_sep":
			s, ok := val.(basetypes.StringValue)
			if !ok || s.IsNull() {
				return out, function.NewArgumentFuncError(1, "options.pair_sep must be a string")
			}
			out.pairSep = s.ValueString()
		case "kv_sep":
			s, ok := val.(basetypes.StringValue)
			if !ok || s.IsNull() {
				return out, function.NewArgumentFuncError(1, "options.kv_sep must be a string")
			}
			out.kvSep = s.ValueString()
		case "trim":
			b, ok := val.(basetypes.BoolValue)
			if !ok || b.IsNull() {
				return out, function.NewArgumentFuncError(1, "options.trim must be a boolean")
			}
			out.trim = b.ValueBool()
		case "unquote":
			b, ok := val.(basetypes.BoolValue)
			if !ok || b.IsNull() {
				return out, function.NewArgumentFuncError(1, "options.unquote must be a boolean")
			}
			out.unquote = b.ValueBool()
		default:
			return out, function.NewArgumentFuncError(1, fmt.Sprintf("unknown option key %q; supported keys are pair_sep, kv_sep, trim, unquote", k))
		}
	}
	if out.pairSep == "" {
		return out, function.NewArgumentFuncError(1, "options.pair_sep must not be empty")
	}
	if out.kvSep == "" {
		return out, function.NewArgumentFuncError(1, "options.kv_sep must not be empty")
	}
	if out.pairSep == out.kvSep {
		return out, function.NewArgumentFuncError(1, fmt.Sprintf("options.pair_sep and options.kv_sep must differ; both are %q", out.pairSep))
	}
	return out, nil
}

func (f *ParseKVFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var s string
	var optsArgs []types.Dynamic
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &s, &optsArgs))
	if resp.Error != nil {
		return
	}
	if len(s) > textMaxInputBytes {
		resp.Error = function.NewArgumentFuncError(0, fmt.Sprintf("s exceeds maximum supported length of %d bytes", textMaxInputBytes))
		return
	}
	if parseKVOptionsHaveUnknown(optsArgs) {
		// An option value is unknown at plan time; return an unknown result so the plan resolves at apply.
		resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, types.MapUnknown(types.StringType)))
		return
	}

	opts, ferr := parseParseKVOptions(optsArgs)
	if ferr != nil {
		resp.Error = ferr
		return
	}

	parsed, err := parseKV(s, opts)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, err.Error())
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, parsed))
}

// parseKV splits s into pairs on opts.pairSep, then each pair on the first opts.kvSep, applying trimming and quote-aware unwrapping per the options. It is the pure core with no framework dependency.
func parseKV(s string, opts parseKVOpts) (map[string]string, error) {
	out := make(map[string]string)
	for _, seg := range splitOnSep(s, opts.pairSep, opts.unquote) {
		pair := seg
		if opts.trim {
			pair = strings.TrimSpace(pair)
		}
		if pair == "" {
			// Skip empty segments (a trailing pair_sep, or "a=1,,b=2").
			continue
		}
		idx := indexSep(pair, opts.kvSep, opts.unquote)
		if idx < 0 {
			return nil, fmt.Errorf("pair %q has no key/value separator %q", seg, opts.kvSep)
		}
		key := pair[:idx]
		value := pair[idx+len(opts.kvSep):]
		if opts.trim {
			key = strings.TrimSpace(key)
			value = strings.TrimSpace(value)
		}
		if opts.unquote {
			key = unwrapQuotes(key)
			value = unwrapQuotes(value)
		}
		if _, dup := out[key]; dup {
			return nil, fmt.Errorf("duplicate key %q", key)
		}
		out[key] = value
	}
	return out, nil
}

// indexSep returns the byte index of the first occurrence of sep in s that is not enclosed in matching quotes, or -1. When quoteAware is false, quotes carry no meaning and this is a plain first-index search.
func indexSep(s, sep string, quoteAware bool) int {
	if sep == "" {
		return -1
	}
	var quote byte
	for i := 0; i < len(s); i++ {
		c := s[i]
		if quoteAware {
			if quote != 0 {
				if c == quote {
					quote = 0
				}
				continue
			}
			if c == '"' || c == '\'' {
				quote = c
				continue
			}
		}
		if quote == 0 && strings.HasPrefix(s[i:], sep) {
			return i
		}
	}
	return -1
}

// splitOnSep splits s on every top-level (unquoted, when quoteAware) occurrence of sep.
func splitOnSep(s, sep string, quoteAware bool) []string {
	var out []string
	for {
		idx := indexSep(s, sep, quoteAware)
		if idx < 0 {
			return append(out, s)
		}
		out = append(out, s[:idx])
		s = s[idx+len(sep):]
	}
}

// unwrapQuotes strips a single matching pair of surrounding double or single quotes when they are the first and last byte of s. Quotes count as a wrapper only in that position, so `b"c"` and a lone quote are returned unchanged.
func unwrapQuotes(s string) string {
	if len(s) >= 2 {
		q := s[0]
		if (q == '"' || q == '\'') && s[len(s)-1] == q {
			return s[1 : len(s)-1]
		}
	}
	return s
}

// parseKVOptionsHaveUnknown reports whether the variadic options object holds any unknown value, so the caller can return an unknown result rather than erroring at plan time.
func parseKVOptionsHaveUnknown(opts []types.Dynamic) bool {
	for _, o := range opts {
		if attrValueHasUnknown(o) {
			return true
		}
	}
	return false
}

func attrValueHasUnknown(v attr.Value) bool {
	if v == nil {
		return false
	}
	if v.IsUnknown() {
		return true
	}
	switch val := v.(type) {
	case basetypes.DynamicValue:
		return attrValueHasUnknown(val.UnderlyingValue())
	case basetypes.ObjectValue:
		for _, a := range val.Attributes() {
			if attrValueHasUnknown(a) {
				return true
			}
		}
	}
	return false
}
