/*
Unicode normalization (Unicode Annex #15, Unicode Normalization Forms).

Strings that look identical can be encoded multiple ways: `é` is either U+00E9 (precomposed, NFC) or U+0065 followed by U+0301 (decomposed, NFD). Copy-paste from rich-text editors, browsers, and macOS APIs routinely produces NFD strings; most servers and Linux tooling work with NFC; some legacy systems want compatibility-decomposed (NFKD) forms. Mismatched normalization is the cause of "looks the same, doesn't compare equal" bugs.

This function exposes the four canonical forms (NFC, NFD, NFKC, NFKD) as a parameter, and is a thin wrapper over `golang.org/x/text/unicode/norm`. That package is the canonical Unicode normalizer for Go, written by the Go team.
*/

package text

import (
	"context"
	_ "embed"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"golang.org/x/text/unicode/norm"
)

var _ function.Function = (*UnicodeNormalizeFunction)(nil)

//go:embed descriptions/unicode_normalize.md
var unicodeNormalizeDescription string

type UnicodeNormalizeFunction struct{}

func NewUnicodeNormalizeFunction() function.Function { return &UnicodeNormalizeFunction{} }

func (f *UnicodeNormalizeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "unicode_normalize"
}

func (f *UnicodeNormalizeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Normalize a string to one of the four Unicode normalization forms",
		MarkdownDescription: unicodeNormalizeDescription,
		Parameters: []function.Parameter{
			function.StringParameter{Name: "s", Description: "The string to normalize."},
			function.StringParameter{Name: "form", Description: "Normalization form: \"NFC\", \"NFD\", \"NFKC\", or \"NFKD\". Case-insensitive."},
		},
		Return: function.StringReturn{},
	}
}

func (f *UnicodeNormalizeFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var s, formName string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &s, &formName))
	if resp.Error != nil {
		return
	}
	if len(s) > textMaxInputBytes {
		resp.Error = function.NewArgumentFuncError(0, fmt.Sprintf("s exceeds maximum supported length of %d bytes", textMaxInputBytes))
		return
	}

	// Match case-insensitively so "nfc" / "Nfc" / "NFC" all work: UAX #15 itself doesn't prescribe casing for the form names, only the algorithms.
	var form norm.Form
	switch strings.ToUpper(formName) {
	case "NFC":
		form = norm.NFC
	case "NFD":
		form = norm.NFD
	case "NFKC":
		form = norm.NFKC
	case "NFKD":
		form = norm.NFKD
	default:
		resp.Error = function.NewArgumentFuncError(1, fmt.Sprintf("form must be \"NFC\", \"NFD\", \"NFKC\", or \"NFKD\" (case-insensitive); received %q", formName))
		return
	}

	out := form.String(s)
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &out))
}
