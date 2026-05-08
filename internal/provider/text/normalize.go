/*
Unicode normalization (Unicode Annex #15 — Unicode Normalization Forms).

Strings that look identical can be encoded multiple ways: `é` is either U+00E9 (precomposed, NFC) or U+0065 followed by U+0301 (decomposed, NFD). Copy-paste from rich-text editors, browsers, and macOS APIs routinely produces NFD strings; most servers and Linux tooling work with NFC; some legacy systems want compatibility-decomposed (NFKD) forms. Mismatched normalization is the cause of "looks the same, doesn't compare equal" bugs.

This function exposes the four canonical forms — NFC, NFD, NFKC, NFKD — as a parameter, and is a thin wrapper over `golang.org/x/text/unicode/norm`. That package is the canonical Unicode normalizer for Go, written by the Go team.
*/

package text

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"golang.org/x/text/unicode/norm"
)

var _ function.Function = (*UnicodeNormalizeFunction)(nil)

type UnicodeNormalizeFunction struct{}

func NewUnicodeNormalizeFunction() function.Function { return &UnicodeNormalizeFunction{} }

func (f *UnicodeNormalizeFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "unicode_normalize"
}

func (f *UnicodeNormalizeFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Normalize a string to one of the four Unicode normalization forms",
		MarkdownDescription: "Returns `s` re-encoded under the named [Unicode Normalization Form](https://unicode.org/reports/tr15/), one of:\n\n- `\"NFC\"` — Canonical Composition (the most common server-side form)\n- `\"NFD\"` — Canonical Decomposition\n- `\"NFKC\"` — Compatibility Composition (collapses ligatures and width variants)\n- `\"NFKD\"` — Compatibility Decomposition\n\nThis fixes \"looks the same, doesn't compare equal\" bugs caused by NFC vs NFD differences (browsers, macOS, and rich-text editors often hand you NFD; most server-side data is NFC). For most use cases the right call is `unicode_normalize(s, \"NFC\")`.\n\n**Caveat for `\"NFD\"` and `\"NFKD\"`**: Terraform's value-handling layer (cty) re-normalizes every string to NFC at expression boundaries, so a decomposed result is silently re-composed to NFC the moment it flows into another HCL expression — including `output` blocks. NFC-producing forms (`\"NFC\"` and `\"NFKC\"`) round-trip correctly; the decomposed forms are useful only for downstream consumers that ingest the *exact* function return value before Terraform sees it (e.g. when feeding into another Burnham function within the same expression, or when the byte representation is captured before cty touches it).\n\nBacked by [`golang.org/x/text/unicode/norm`](https://pkg.go.dev/golang.org/x/text/unicode/norm), the canonical Go implementation.",
		Parameters: []function.Parameter{
			function.StringParameter{Name: "s", Description: "The string to normalize."},
			function.StringParameter{Name: "form", Description: "Normalization form: \"NFC\", \"NFD\", \"NFKC\", or \"NFKD\". Case-sensitive."},
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

	// Match case-insensitively so "nfc" / "Nfc" / "NFC" all work — UAX #15 itself doesn't prescribe casing for the form names, only the algorithms.
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
