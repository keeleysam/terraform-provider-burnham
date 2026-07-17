package color

import (
	"context"
	_ "embed"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/keeleysam/terraform-burnham/internal/provider/optionsutil"
)

// convertOpts controls how convertColor serializes its result.
type convertOpts struct {
	hash  bool // include a leading "#" on hex output
	upper bool // uppercase hex letters
}

func defaultConvertOpts() convertOpts { return convertOpts{hash: true, upper: false} }

// convertColor parses any CSS color and re-serializes it in the target notation. Alpha is preserved when it is below 1 (8-digit hex, rgba(), hsla(), or the oklch "/ a" slot).
func convertColor(input, target string, opts convertOpts) (string, error) {
	c, alpha, err := parseColor(input)
	if err != nil {
		return "", err
	}
	switch strings.ToLower(strings.TrimSpace(target)) {
	case "hex":
		return hexOut(c, alpha, opts.hash, opts.upper), nil
	case "rgb":
		r, g, b := c.Clamped().RGB255()
		if alpha < 1 {
			return fmt.Sprintf("rgba(%d, %d, %d, %s)", r, g, b, trimFloat(clamp01(alpha), 3)), nil
		}
		return fmt.Sprintf("rgb(%d, %d, %d)", r, g, b), nil
	case "hsl":
		h, s, l := c.Clamped().Hsl()
		hh := int(math.Round(h))
		ss := int(math.Round(s * 100))
		ll := int(math.Round(l * 100))
		if alpha < 1 {
			return fmt.Sprintf("hsla(%d, %d%%, %d%%, %s)", hh, ss, ll, trimFloat(clamp01(alpha), 3)), nil
		}
		return fmt.Sprintf("hsl(%d, %d%%, %d%%)", hh, ss, ll), nil
	case "oklch":
		l, ch, h := oklchOf(c)
		base := fmt.Sprintf("oklch(%s %s %s", trimFloat(l, 4), trimFloat(ch, 4), trimFloat(h, 2))
		if alpha < 1 {
			return base + " / " + trimFloat(clamp01(alpha), 3) + ")", nil
		}
		return base + ")", nil
	default:
		return "", fmt.Errorf("unknown target notation %q; supported: hex, rgb, hsl, oklch", target)
	}
}

// trimFloat formats f with up to prec decimals and no trailing zeros, so 0.5 stays "0.5" not "0.500".
func trimFloat(f float64, prec int) string {
	s := strconv.FormatFloat(f, 'f', prec, 64)
	if strings.Contains(s, ".") {
		s = strings.TrimRight(s, "0")
		s = strings.TrimRight(s, ".")
	}
	return s
}

func parseConvertOpts(opts []types.Dynamic) (convertOpts, *function.FuncError) {
	out := defaultConvertOpts()
	attrs, ferr := optionsutil.SingleOptionsObject(opts, `{ hash = false, uppercase = true }`)
	if ferr != nil {
		return out, ferr
	}
	for k, v := range attrs {
		b, ok := v.(basetypes.BoolValue)
		if !ok || b.IsNull() {
			return out, function.NewArgumentFuncError(2, fmt.Sprintf("options.%s must be a bool", k))
		}
		switch k {
		case "hash":
			out.hash = b.ValueBool()
		case "uppercase":
			out.upper = b.ValueBool()
		default:
			return out, function.NewArgumentFuncError(2, fmt.Sprintf("unknown option key %q; supported keys are hash, uppercase", k))
		}
	}
	return out, nil
}

//go:embed descriptions/color_convert.md
var colorConvertDescription string

var _ function.Function = (*ColorConvertFunction)(nil)

type ColorConvertFunction struct{}

func NewColorConvertFunction() function.Function { return &ColorConvertFunction{} }

func (f *ColorConvertFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "color_convert"
}

func (f *ColorConvertFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Parse any CSS color and re-serialize it in a target notation",
		MarkdownDescription: colorConvertDescription,
		Parameters: []function.Parameter{
			function.StringParameter{Name: "color", Description: "A CSS color in any notation: hex, rgb()/rgba(), hsl(), hwb(), lab()/lch(), oklab()/oklch(), or a named color."},
			function.StringParameter{Name: "target", Description: "The output notation: `hex`, `rgb`, `hsl`, or `oklch`."},
		},
		VariadicParameter: function.DynamicParameter{
			Name:        "options",
			Description: "An optional object. Keys: `hash` (bool, default true; include the leading `#` on hex output) and `uppercase` (bool, default false; uppercase hex letters).",
		},
		Return: function.StringReturn{},
	}
}

func (f *ColorConvertFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var input, target string
	var optsArgs []types.Dynamic
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &input, &target, &optsArgs))
	if resp.Error != nil {
		return
	}
	if unknownStringOptionResult(ctx, resp, optsArgs) {
		return
	}
	opts, ferr := parseConvertOpts(optsArgs)
	if ferr != nil {
		resp.Error = ferr
		return
	}
	out, err := convertColor(input, target, opts)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, err.Error())
		return
	}
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &out))
}
