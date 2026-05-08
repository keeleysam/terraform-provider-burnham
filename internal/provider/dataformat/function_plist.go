package dataformat

import (
	"context"
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"howett.net/plist"
)

var _ function.Function = (*PlistDecodeFunction)(nil)

type PlistDecodeFunction struct{}

func NewPlistDecodeFunction() function.Function {
	return &PlistDecodeFunction{}
}

func (f *PlistDecodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "plistdecode"
}

func (f *PlistDecodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Parse an Apple property list into a Terraform value",
		MarkdownDescription: "Parses an [Apple property list](https://developer.apple.com/documentation/foundation/archives_and_serialization/property_lists) string into a Terraform value. Auto-detects XML, binary, OpenStep, and GNUStep formats. For binary plists, pass the output of `filebase64()` — base64-encoded input is detected automatically.\n\n`<date>` elements decode as tagged objects of the form `{ __plist_type = \"date\", value = \"<RFC 3339 string>\" }`; `<data>` elements as `{ __plist_type = \"data\", value = \"<base64>\" }`; whole-number `<real>` elements as `{ __plist_type = \"real\", value = \"...\" }` (to distinguish from `<integer>`). All three round-trip cleanly back through `plistencode`.\n\n**Common uses:** reading Apple configuration profiles (`.mobileconfig`), `.plist` preference files, or any payload from MDM tooling where the on-disk format may be XML or binary.",
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:        "input",
				Description: "A plist string (from file()) or base64-encoded plist (from filebase64()).",
			},
		},
		Return: function.DynamicReturn{},
	}
}

func (f *PlistDecodeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var input string

	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &input))
	if resp.Error != nil {
		return
	}

	if len(input) > dataformatMaxInputBytes {
		resp.Error = function.NewArgumentFuncError(0, fmt.Sprintf("input exceeds maximum supported length of %d bytes", dataformatMaxInputBytes))
		return
	}
	data := []byte(input)

	// Auto-detect: if the input doesn't look like a raw plist, try base64 decoding.
	if !looksLikePlist(input) {
		decoded, err := base64.StdEncoding.DecodeString(input)
		if err != nil {
			resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError(
				"Input is not a recognized plist format and is not valid base64: "+err.Error()))
			return
		}
		data = decoded
	}

	var goVal interface{}
	_, err := plist.Unmarshal(data, &goVal)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, "failed to decode plist: "+err.Error())
		return
	}

	tfVal, err := goToTerraformValuePlist(goVal)
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("Failed to convert plist value: "+err.Error()))
		return
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, types.DynamicValue(tfVal)))
}

// looksLikePlist checks if the input string starts with known plist signatures.
func looksLikePlist(s string) bool {
	trimmed := strings.TrimSpace(s)
	switch {
	case strings.HasPrefix(trimmed, "<?xml"):
		return true
	case strings.HasPrefix(trimmed, "<!DOCTYPE plist"):
		return true
	case strings.HasPrefix(trimmed, "<plist"):
		return true
	case strings.HasPrefix(trimmed, "bplist"):
		return true
	case strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "("):
		// OpenStep format
		return true
	default:
		return false
	}
}

var _ function.Function = (*PlistEncodeFunction)(nil)

type PlistEncodeFunction struct{}

func NewPlistEncodeFunction() function.Function {
	return &PlistEncodeFunction{}
}

func (f *PlistEncodeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "plistencode"
}

func (f *PlistEncodeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Encode a value as an Apple property list",
		MarkdownDescription: "Encodes a Terraform value as an [Apple property list](https://developer.apple.com/documentation/foundation/archives_and_serialization/property_lists) string. Default output format is XML; pass `format = \"binary\"` for a base64-encoded binary plist or `format = \"openstep\"` for the OpenStep/GNUStep textual format.\n\nTagged objects from `plistdate()`, `plistdata()`, and `plistreal()` are converted to native `<date>`, `<data>`, and `<real>` elements. Numbers with no fractional part become `<integer>`; numbers with a fractional part become `<real>`. Pass an optional `comments` key in `options` (mirroring the data structure) to inject `<!-- -->` XML comments before specific keys.\n\n**Common uses:** generating Apple configuration profiles (`.mobileconfig`) for MDM deployment, WiFi/VPN payloads, app preference files, or anything else that downstream Apple tooling consumes.",
		Parameters: []function.Parameter{
			function.DynamicParameter{
				Name:        "value",
				Description: "The value to encode as a plist.",
			},
		},
		VariadicParameter: function.DynamicParameter{
			Name:        "options",
			Description: "An optional options object. Supported keys: \"format\" (string) — \"xml\" (default), \"binary\", or \"openstep\"; \"comments\" (object) — mirrored structure where string values become <!-- comment --> in the XML output. Pass at most one.",
		},
		Return: function.StringReturn{},
	}
}

func (f *PlistEncodeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var value types.Dynamic
	var optsArgs []types.Dynamic

	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &value, &optsArgs))
	if resp.Error != nil {
		return
	}

	formatStr := "xml"
	var comments attr.Value

	if len(optsArgs) == 1 {
		obj, ok := optsArgs[0].UnderlyingValue().(basetypes.ObjectValue)
		if !ok {
			resp.Error = function.NewArgumentFuncError(1, fmt.Sprintf("options must be an object, got %T", optsArgs[0].UnderlyingValue()))
			return
		}
		attrs := obj.Attributes()

		parsed, err := getStringOption(attrs, "format")
		if err != nil {
			resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError(err.Error()))
			return
		}
		if parsed != "" {
			formatStr = parsed
		}

		if c, ok := attrs["comments"]; ok {
			comments = c
		}
	} else if len(optsArgs) > 1 {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewArgumentFuncError(1, "At most one options argument may be provided."))
		return
	}

	plistFormat, err := parsePlistFormat(formatStr)
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError(err.Error()))
		return
	}

	goVal, err := terraformValueToGo(value.UnderlyingValue(), true)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, "failed to convert value: "+err.Error())
		return
	}

	prepared := goValueForPlistEncode(goVal)

	data, err := plist.MarshalIndent(prepared, plistFormat, "\t")
	if err != nil {
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("Failed to encode plist: "+err.Error()))
		return
	}

	var result string
	if plistFormat == plist.BinaryFormat {
		result = base64.StdEncoding.EncodeToString(data)
	} else {
		result = string(data)
	}

	// Apply XML comments (only for XML format).
	if comments != nil && plistFormat == plist.XMLFormat {
		result = applyPlistXMLComments(result, comments)
	}

	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, result))
}

func parsePlistFormat(s string) (int, error) {
	switch strings.ToLower(s) {
	case "xml":
		return plist.XMLFormat, nil
	case "binary":
		return plist.BinaryFormat, nil
	case "openstep", "gnustep":
		return plist.GNUStepFormat, nil
	default:
		return 0, fmt.Errorf("unsupported plist format %q: must be \"xml\", \"binary\", or \"openstep\"", s)
	}
}

// applyPlistXMLComments injects <!-- comment --> lines into plist XML output.
// The comments value mirrors the data structure — keys match plist <key> elements,
// and string values become XML comments above the matching <key> line.
func applyPlistXMLComments(output string, comments attr.Value) string {
	commentsObj, ok := comments.(basetypes.ObjectValue)
	if !ok {
		return output
	}

	lines := strings.Split(output, "\n")
	result := applyPlistCommentsToLines(lines, commentsObj.Attributes())
	return strings.Join(result, "\n")
}

// applyPlistCommentsToLines walks the XML lines and injects comments before
// matching <key> elements.
func applyPlistCommentsToLines(lines []string, commentsMap map[string]attr.Value) []string {
	var result []string

	for i := 0; i < len(lines); i++ {
		trimmed := strings.TrimSpace(lines[i])

		// Check if this is a <key>Name</key> line.
		if strings.HasPrefix(trimmed, "<key>") && strings.HasSuffix(trimmed, "</key>") {
			keyName := trimmed[5 : len(trimmed)-6]

			if commentVal, ok := commentsMap[keyName]; ok {
				// Detect the indentation of the <key> line.
				indent := lines[i][:len(lines[i])-len(strings.TrimLeft(lines[i], "\t "))]

				switch cv := commentVal.(type) {
				case basetypes.StringValue:
					for _, cline := range strings.Split(cv.ValueString(), "\n") {
						result = append(result, indent+"<!-- "+escapeXMLComment(cline)+" -->")
					}
				case basetypes.ObjectValue:
					// Nested comments — find the <dict> that follows the value for this key,
					// and recursively apply comments to it. We do this by scanning ahead.
					result = append(result, lines[i])
					i++
					// The value after a <key> might be a <dict> on the next line(s).
					// Collect lines until we find the matching </dict>, apply nested comments.
					result, i = applyNestedPlistComments(lines, i, result, cv.Attributes())
					continue
				}
			}
		}

		result = append(result, lines[i])
	}

	return result
}

// applyNestedPlistComments handles nested comment objects by scanning forward
// to find the <dict>...</dict> block and applying comments within it.
func applyNestedPlistComments(lines []string, startIdx int, result []string, nestedComments map[string]attr.Value) ([]string, int) {
	// Look for a <dict> to start the nested block.
	i := startIdx
	for i < len(lines) {
		trimmed := strings.TrimSpace(lines[i])
		if trimmed == "<dict>" {
			result = append(result, lines[i])
			i++
			// Collect lines inside this dict until </dict>.
			var dictLines []string
			depth := 1
			for i < len(lines) && depth > 0 {
				dt := strings.TrimSpace(lines[i])
				if dt == "<dict>" {
					depth++
				} else if dt == "</dict>" {
					depth--
					if depth == 0 {
						break
					}
				}
				dictLines = append(dictLines, lines[i])
				i++
			}
			// Apply comments to the collected dict lines.
			commented := applyPlistCommentsToLines(dictLines, nestedComments)
			result = append(result, commented...)
			// Append the closing </dict>.
			if i < len(lines) {
				result = append(result, lines[i])
			}
			return result, i
		}
		// If the value isn't a dict, just pass through.
		result = append(result, lines[i])
		if trimmed != "" {
			return result, i
		}
		i++
	}
	return result, i - 1
}

// escapeXMLComment replaces "--" with "‐‐" (Unicode hyphens) in comment text,
// since XML comments cannot contain the "--" sequence.
func escapeXMLComment(s string) string {
	return strings.ReplaceAll(s, "--", "\u2010\u2010")
}
