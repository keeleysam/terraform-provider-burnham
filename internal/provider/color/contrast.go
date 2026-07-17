package color

import (
	"context"
	_ "embed"
	"fmt"
	"math"
	"math/big"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/keeleysam/terraform-burnham/internal/provider/optionsutil"
	colorful "github.com/lucasb-eyer/go-colorful"
)

// relLuminance is the WCAG 2.x relative luminance of an sRGB color. It uses the exact WCAG piecewise gamma expansion (threshold 0.03928) and the Rec. 709 weights, rather than go-colorful's LinearRgb, so the number matches what accessibility checkers report.
func relLuminance(c colorful.Color) float64 {
	lin := func(ch float64) float64 {
		ch = clamp01(ch)
		if ch <= 0.03928 {
			return ch / 12.92
		}
		return math.Pow((ch+0.055)/1.055, 2.4)
	}
	return 0.2126*lin(c.R) + 0.7152*lin(c.G) + 0.0722*lin(c.B)
}

// contrastRatioColors returns the WCAG 2.x contrast ratio between two colors, in [1, 21].
func contrastRatioColors(c1, c2 colorful.Color) float64 {
	l1, l2 := relLuminance(c1), relLuminance(c2)
	if l1 < l2 {
		l1, l2 = l2, l1
	}
	return (l1 + 0.05) / (l2 + 0.05)
}

func contrastRatio(a, b string) (float64, error) {
	c1, _, err := parseColor(a)
	if err != nil {
		return 0, err
	}
	c2, _, err := parseColor(b)
	if err != nil {
		return 0, err
	}
	return contrastRatioColors(c1, c2), nil
}

// readableText returns whichever of candidates has the highest WCAG contrast against bg, returning the candidate string exactly as supplied. On a tie the earlier candidate wins.
func readableText(bg string, candidates []string) (string, error) {
	bgColor, _, err := parseColor(bg)
	if err != nil {
		return "", err
	}
	best, bestRatio := "", -1.0
	for _, cand := range candidates {
		c, _, err := parseColor(cand)
		if err != nil {
			return "", fmt.Errorf("candidate %q: could not parse", cand)
		}
		if r := contrastRatioColors(bgColor, c); r > bestRatio {
			best, bestRatio = cand, r
		}
	}
	return best, nil
}

// ── color_contrast_ratio ───────────────────────────────────────

//go:embed descriptions/color_contrast_ratio.md
var colorContrastRatioDescription string

var _ function.Function = (*ColorContrastRatioFunction)(nil)

type ColorContrastRatioFunction struct{}

func NewColorContrastRatioFunction() function.Function { return &ColorContrastRatioFunction{} }

func (f *ColorContrastRatioFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "color_contrast_ratio"
}

func (f *ColorContrastRatioFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "WCAG 2.x contrast ratio between two colors (1 to 21)",
		MarkdownDescription: colorContrastRatioDescription,
		Parameters: []function.Parameter{
			function.StringParameter{Name: "a", Description: "The first color, in any CSS notation."},
			function.StringParameter{Name: "b", Description: "The second color, in any CSS notation."},
		},
		Return: function.NumberReturn{},
	}
}

func (f *ColorContrastRatioFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var a, b string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &a, &b))
	if resp.Error != nil {
		return
	}
	ratio, err := contrastRatio(a, b)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, err.Error())
		return
	}
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, big.NewFloat(ratio)))
}

// ── color_readable_text ────────────────────────────────────────

//go:embed descriptions/color_readable_text.md
var colorReadableTextDescription string

var _ function.Function = (*ColorReadableTextFunction)(nil)

type ColorReadableTextFunction struct{}

func NewColorReadableTextFunction() function.Function { return &ColorReadableTextFunction{} }

func (f *ColorReadableTextFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "color_readable_text"
}

func (f *ColorReadableTextFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Pick the most legible text color for a background (WCAG)",
		MarkdownDescription: colorReadableTextDescription,
		Parameters: []function.Parameter{
			function.StringParameter{Name: "background", Description: "The background color, in any CSS notation."},
		},
		VariadicParameter: function.DynamicParameter{
			Name:        "options",
			Description: "An optional object. Key: `candidates` (list of colors to choose from; default `[\"#000000\", \"#ffffff\"]`). Returns the candidate with the highest contrast, exactly as written.",
		},
		Return: function.StringReturn{},
	}
}

func (f *ColorReadableTextFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var bg string
	var optsArgs []types.Dynamic
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &bg, &optsArgs))
	if resp.Error != nil {
		return
	}
	if unknownStringOptionResult(ctx, resp, optsArgs) {
		return
	}
	candidates, ferr := readableCandidates(optsArgs)
	if ferr != nil {
		resp.Error = ferr
		return
	}
	out, err := readableText(bg, candidates)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, err.Error())
		return
	}
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &out))
}

// readableCandidates parses the optional { candidates = [...] } list, defaulting to black and white.
func readableCandidates(opts []types.Dynamic) ([]string, *function.FuncError) {
	def := []string{"#000000", "#ffffff"}
	attrs, ferr := optionsutil.SingleOptionsObject(opts, `{ candidates = ["#111", "#eee"] }`)
	if ferr != nil {
		return def, ferr
	}
	if attrs == nil {
		return def, nil
	}
	for k, v := range attrs {
		if k != "candidates" {
			return def, function.NewArgumentFuncError(1, fmt.Sprintf("unknown option key %q; the only key is candidates", k))
		}
		list, err := stringListAttr(v)
		if err != nil {
			return def, function.NewArgumentFuncError(1, "options.candidates must be a list of color strings")
		}
		if len(list) == 0 {
			return def, function.NewArgumentFuncError(1, "options.candidates must not be empty")
		}
		return list, nil
	}
	return def, nil
}

// stringListAttr extracts a []string from a Terraform list/tuple attribute of strings.
func stringListAttr(v attr.Value) ([]string, error) {
	var elems []attr.Value
	switch val := v.(type) {
	case basetypes.ListValue:
		elems = val.Elements()
	case basetypes.TupleValue:
		elems = val.Elements()
	case basetypes.SetValue:
		elems = val.Elements()
	default:
		return nil, fmt.Errorf("not a list")
	}
	out := make([]string, len(elems))
	for i, e := range elems {
		s, ok := e.(basetypes.StringValue)
		if !ok || s.IsNull() {
			return nil, fmt.Errorf("element %d is not a string", i)
		}
		out[i] = s.ValueString()
	}
	return out, nil
}
