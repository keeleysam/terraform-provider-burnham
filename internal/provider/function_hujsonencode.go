package provider

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/tailscale/hujson"
)

var _ function.Function = (*HuJSONEncodeFunction)(nil)

type HuJSONEncodeFunction struct{}

func NewHuJSONEncodeFunction() function.Function {
	return &HuJSONEncodeFunction{}
}

func (f *HuJSONEncodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "hujsonencode"
}

func (f *HuJSONEncodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary: "Encode a value as a HuJSON (JWCC) string",
		Description: "Encodes a Terraform value as a HuJSON string with trailing commas and pretty-printed formatting. " +
			"By default every object member and array element is placed on its own line. " +
			"Pass an optional options object with \"indent\" to override the default tab indentation, " +
			"\"compact\" (bool) to opt in to hujson.Format's \"fit on one line if it can\" packing instead of the default expanded layout, " +
			"and \"comments\" to add comments to the output. The comments object mirrors the data structure — " +
			"each key corresponds to a key in the data, and the string value becomes a comment placed before that key. " +
			"Single-line strings become // comments, multi-line strings become /* */ comments.",
		Parameters: []function.Parameter{
			function.DynamicParameter{
				Name:        "value",
				Description: "The value to encode as HuJSON.",
			},
		},
		VariadicParameter: function.DynamicParameter{
			Name: "options",
			Description: "An optional options object. Supported keys: " +
				"\"indent\" (string) — indentation string, default \"\\t\"; " +
				"\"compact\" (bool) — when true, use hujson.Format's \"fit on one line if it can\" packing instead of the default always-expanded layout; " +
				"\"comments\" (object) — a mirrored structure where string values become comments placed before the matching key. " +
				"Pass at most one.",
		},
		Return: function.StringReturn{},
	}
}

func (f *HuJSONEncodeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var value types.Dynamic
	var optsArgs []types.Dynamic

	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &value, &optsArgs))
	if resp.Error != nil {
		return
	}

	indent := "\t"
	compact := false
	var comments attr.Value

	if len(optsArgs) == 1 {
		obj, ok := optsArgs[0].UnderlyingValue().(basetypes.ObjectValue)
		if !ok {
			resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError(fmt.Sprintf("options must be an object, got %T", optsArgs[0].UnderlyingValue())))
			return
		}
		attrs := obj.Attributes()

		parsed, err := getStringOption(attrs, "indent")
		if err != nil {
			resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError(err.Error()))
			return
		}
		if parsed != "" {
			indent = parsed
		}

		if cv, _, err := getBoolOption(attrs, "compact"); err != nil {
			resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError(err.Error()))
			return
		} else {
			compact = cv
		}

		if c, ok := attrs["comments"]; ok {
			comments = c
		}
	} else if len(optsArgs) > 1 {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewArgumentFuncError(1, "At most one options argument may be provided."))
		return
	}

	goVal, err := terraformValueToGo(value.UnderlyingValue(), false)
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("Failed to convert value: "+err.Error()))
		return
	}

	prepared := goValueForJSONEncode(goVal)

	jsonBytes, err := json.Marshal(prepared)
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("Failed to marshal JSON: "+err.Error()))
		return
	}

	if !compact {
		// Default: always-expanded layout. Round-trip the JSON through a
		// UseNumber decoder so the pretty encoder sees a uniform
		// map[string]any / []any / json.Number tree and doesn't have to
		// handle every Go type goValueForJSONEncode emits.
		var generic any
		dec := json.NewDecoder(bytes.NewReader(jsonBytes))
		dec.UseNumber()
		if err := dec.Decode(&generic); err != nil {
			resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("Failed to re-decode JSON: "+err.Error()))
			return
		}
		result := prettyEncodeHuJSON(generic, comments, indent)
		resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, result))
		return
	}

	// compact = true: defer to hujson.Format's "fit on one line if it can"
	// packing. Make the JSON non-standard so Format() will add trailing
	// commas.
	hujsonBytes := append([]byte("//\n"), jsonBytes...)

	ast, err := hujson.Parse(hujsonBytes)
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("Failed to parse as HuJSON: "+err.Error()))
		return
	}

	ast.Format()

	// Remove the injected comment.
	ast.BeforeExtra = nil

	// Apply comments from the mirrored structure.
	if comments != nil {
		if err := applyComments(&ast, comments); err != nil {
			resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("Failed to apply comments: "+err.Error()))
			return
		}
		// Re-format after adding comments so indentation is correct.
		ast.Format()
	}

	result := string(ast.Pack())

	// hujson.Format() always uses tabs. If a different indent was requested,
	// replace the leading tabs on each line.
	if indent != "\t" {
		lines := strings.Split(result, "\n")
		for i, line := range lines {
			trimmed := strings.TrimLeft(line, "\t")
			tabCount := len(line) - len(trimmed)
			if tabCount > 0 {
				lines[i] = strings.Repeat(indent, tabCount) + trimmed
			}
		}
		result = strings.Join(lines, "\n")
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, result))
}

// applyComments walks the hujson AST and the comments value in parallel,
// setting BeforeExtra on matching keys.
func applyComments(ast *hujson.Value, comments attr.Value) error {
	switch cv := comments.(type) {
	case basetypes.ObjectValue:
		return applyCommentsToValue(ast, cv.Attributes())
	default:
		return nil
	}
}

func applyCommentsToValue(ast *hujson.Value, commentsMap map[string]attr.Value) error {
	switch comp := ast.Value.(type) {
	case *hujson.Object:
		for key, commentVal := range commentsMap {
			memberIdx := findObjectMember(comp, key)
			if memberIdx < 0 {
				continue // Key not in data, silently skip.
			}

			switch cv := commentVal.(type) {
			case basetypes.StringValue:
				// Leaf: set comment on this member's name.
				comp.Members[memberIdx].Name.BeforeExtra = formatComment(cv.ValueString())

			case basetypes.ObjectValue:
				// Nested: recurse into the member's value.
				if err := applyCommentsToValue(&comp.Members[memberIdx].Value, cv.Attributes()); err != nil {
					return err
				}
			}
		}

	case *hujson.Array:
		for key, commentVal := range commentsMap {
			idx, err := strconv.Atoi(key)
			if err != nil || idx < 0 || idx >= len(comp.Elements) {
				continue // Not a valid index, silently skip.
			}

			switch cv := commentVal.(type) {
			case basetypes.StringValue:
				comp.Elements[idx].BeforeExtra = formatComment(cv.ValueString())

			case basetypes.ObjectValue:
				if err := applyCommentsToValue(&comp.Elements[idx], cv.Attributes()); err != nil {
					return err
				}
			}
		}
	}

	return nil
}

// findObjectMember returns the index of the member with the given key, or -1.
func findObjectMember(obj *hujson.Object, key string) int {
	quotedKey := `"` + key + `"`
	for i, m := range obj.Members {
		if string(m.Name.Value.(hujson.Literal)) == quotedKey {
			return i
		}
	}
	return -1
}

// formatComment converts a string to a hujson.Extra comment.
// Single-line strings become // comments, multi-line become /* */ comments.
// A leading newline is included so Format() places the comment on its own line.
func formatComment(s string) hujson.Extra {
	if strings.Contains(s, "\n") {
		// Escape */ inside the comment to prevent premature block comment close.
		escaped := strings.ReplaceAll(s, "*/", "*\\/")
		return hujson.Extra("\n/* " + escaped + " */\n")
	}
	return hujson.Extra("\n// " + s + "\n")
}
