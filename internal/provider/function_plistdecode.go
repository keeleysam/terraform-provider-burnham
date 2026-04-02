package provider

import (
	"context"
	"encoding/base64"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
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
		Summary:     "Parse an Apple property list into a Terraform value",
		Description: "Decodes an Apple property list (plist) string into a Terraform value. Auto-detects XML, binary, and OpenStep formats. For binary plists, pass the output of filebase64() — the function auto-detects base64-encoded input. NSDate values become tagged objects with __plist_type=\"date\" and an RFC 3339 value; NSData values become tagged objects with __plist_type=\"data\" and a base64 value.",
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
		resp.Error = function.ConcatFuncErrors(resp.Error, function.NewFuncError("Failed to decode plist: "+err.Error()))
		return
	}

	tfVal, err := goToTerraformValue(goVal)
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
