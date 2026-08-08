package dataformat

import (
	_ "embed"

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
	"github.com/tidwall/pretty"
)

var _ function.Function = (*HuJSONDecodeFunction)(nil)

//go:embed descriptions/hujsondecode.md
var hujsondecodeDescription string

type HuJSONDecodeFunction struct{}

func NewHuJSONDecodeFunction() function.Function {
	return &HuJSONDecodeFunction{}
}

func (f *HuJSONDecodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "hujsondecode"
}

func (f *HuJSONDecodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Parse a HuJSON (JWCC) string into a Terraform value",
		MarkdownDescription: hujsondecodeDescription,
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:        "input",
				Description: "A HuJSON string to decode.",
			},
		},
		Return: function.DynamicReturn{},
	}
}

func (f *HuJSONDecodeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var input string

	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &input))
	if resp.Error != nil {
		return
	}

	if len(input) > dataformatMaxInputBytes {
		resp.Error = function.NewArgumentFuncError(0, fmt.Sprintf("input exceeds maximum supported length of %d bytes", dataformatMaxInputBytes))
		return
	}
	// Standardize strips comments and trailing commas, producing valid JSON.
	standardized, err := hujson.Standardize([]byte(input))
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, "invalid HuJSON: "+err.Error())
		return
	}

	// Unmarshal to Go types using json.Number for precision.
	d := json.NewDecoder(bytes.NewReader(standardized))
	d.UseNumber()
	var goVal interface{}
	if err := d.Decode(&goVal); err != nil {
		resp.Error = function.NewArgumentFuncError(0, "failed to decode JSON: "+err.Error())
		return
	}

	tfVal, err := goToTerraformValue(goVal)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, "failed to convert value: "+err.Error())
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, types.DynamicValue(tfVal)))
}

var _ function.Function = (*HuJSONEncodeFunction)(nil)

//go:embed descriptions/hujsonencode.md
var hujsonencodeDescription string

type HuJSONEncodeFunction struct{}

func NewHuJSONEncodeFunction() function.Function {
	return &HuJSONEncodeFunction{}
}

func (f *HuJSONEncodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "hujsonencode"
}

func (f *HuJSONEncodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Encode a value as a HuJSON (JWCC) string",
		MarkdownDescription: hujsonencodeDescription,
		Parameters: []function.Parameter{
			function.DynamicParameter{
				Name:        "value",
				Description: "The value to encode as HuJSON.",
			},
		},
		VariadicParameter: function.DynamicParameter{
			Name:        "options",
			Description: "An optional options object. Supported keys: \"indent\" (string): indentation string, default \"\\t\"; \"width\" (number, default 0 = one member or element per line): pack arrays of scalars onto one line when they fit within this many columns; \"compact\" (bool): use hujson.Format's \"fit on one line if it can\" packing instead, mutually exclusive with \"width\"; \"escape_html\" (bool, default false): when true, escape \"<\", \">\" and \"&\" to \\u003c / \\u003e / \\u0026; \"comments\" (object): a mirrored structure where string values become comments placed before the matching key. Pass at most one.",
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
	if unknownStringResultIfNeeded(ctx, resp, value.UnderlyingValue(), optsArgs) {
		return
	}

	indent := "\t"
	compact := false
	escapeHTML := false
	width := 0
	var comments attr.Value

	if len(optsArgs) == 1 {
		obj, ok := optsArgs[0].UnderlyingValue().(basetypes.ObjectValue)
		if !ok {
			resp.Error = function.NewArgumentFuncError(1, fmt.Sprintf("options must be an object, got %T", optsArgs[0].UnderlyingValue()))
			return
		}
		attrs := obj.Attributes()

		if err := validateOptionKeys(attrs, "indent", "compact", "escape_html", "comments", "width"); err != nil {
			resp.Error = function.ConcatFuncErrors(resp.Error, function.NewArgumentFuncError(1, err.Error()))
			return
		}

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

		if ev, _, err := getBoolOption(attrs, "escape_html"); err != nil {
			resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError(err.Error()))
			return
		} else {
			escapeHTML = ev
		}

		w, wPresent, err := getIntOption(attrs, "width")
		if err != nil {
			resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError(err.Error()))
			return
		}
		if wPresent {
			if w < 0 {
				resp.Error = function.ConcatFuncErrors(resp.Error, function.NewArgumentFuncError(1, "\"width\" must not be negative"))
				return
			}
			if compact {
				resp.Error = function.ConcatFuncErrors(resp.Error, function.NewArgumentFuncError(1, "\"width\" and \"compact\" are mutually exclusive: \"compact\" defers all line breaking to hujson.Format, \"width\" decides it up front"))
				return
			}
			width = w
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
		resp.Error = function.NewArgumentFuncError(0, "failed to convert value: "+err.Error())
		return
	}

	prepared := goValueForJSONEncode(goVal)

	jsonBytes, err := json.Marshal(prepared)
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("Failed to marshal JSON: "+err.Error()))
		return
	}

	if width == 0 && !compact {
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
		result := prettyEncodeHuJSON(generic, comments, indent, escapeHTML)
		resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, result))
		return
	}

	// Everything from here is the hujson-backed path, shared by "width" and "compact".
	//
	// Line breaking is decided first: "width" hands the job to tidwall/pretty, whose Width applies to arrays of scalars only and never to objects, so an object is always alone on its line and the "}, {" and "[{" shapes cannot occur. "compact" instead leaves it to hujson.Format's own "fit on one line if it can" packing. hujson.Format preserves line breaks it is given, so whichever decision was made survives.
	//
	// Layout always uses tabs, because hujson.Format re-indents with tabs regardless; a requested indent is applied to the finished output further down.
	laid := jsonBytes
	if width > 0 {
		laid = bytes.TrimRight(pretty.PrettyOptions(jsonBytes, &pretty.Options{Width: width, Indent: "\t"}), "\n")
	}

	// hujson.Format only emits trailing commas for a document it already considers non-standard, so the leading comment is what opts into them. hujsonencode always emits trailing commas: a caller who wants standard JSON wants jsonencode.
	ast, err := hujson.Parse(append([]byte("//\n"), laid...))
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, "failed to parse as HuJSON: "+err.Error())
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

	// hujson.Pack() terminates the document with a newline. Every other path in this file and in jsonencode returns an unterminated string, so trim it rather than leaving compact as the odd one out.
	result := strings.TrimRight(string(ast.Pack()), "\n")

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

	// hujson.Format/Pack normalizes `\uXXXX` escapes back to literal characters,
	// so escape_html=true has to re-escape `<`, `>` and `&` in the packed output.
	if escapeHTML {
		result = escapeHTMLInStrings(result)
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
