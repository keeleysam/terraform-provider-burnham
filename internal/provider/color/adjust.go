package color

import (
	"context"
	_ "embed"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	colorful "github.com/lucasb-eyer/go-colorful"
)

// applyChannelOp applies one channel adjustment to cur. A bare number sets the channel absolutely; a string beginning with +, -, *, or / applies that operation relative to the current value (a string that is just a number also sets). This is the chroma.js-style grammar, so "*0.9" darkens by 10% and "+30" rotates hue.
func applyChannelOp(cur float64, v attr.Value) (float64, error) {
	switch val := v.(type) {
	case basetypes.NumberValue:
		if val.IsNull() {
			return cur, fmt.Errorf("null is not a valid adjustment")
		}
		f, _ := val.ValueBigFloat().Float64()
		return f, nil
	case basetypes.StringValue:
		if val.IsNull() {
			return cur, fmt.Errorf("null is not a valid adjustment")
		}
		s := strings.TrimSpace(val.ValueString())
		if s == "" {
			return cur, fmt.Errorf("empty adjustment")
		}
		op := s[0]
		if op == '+' || op == '-' || op == '*' || op == '/' {
			operand, err := strconv.ParseFloat(strings.TrimSpace(s[1:]), 64)
			if err != nil {
				return cur, fmt.Errorf("%q is not a valid operation (expected an operator + - * / followed by a number)", s)
			}
			switch op {
			case '+':
				return cur + operand, nil
			case '-':
				return cur - operand, nil
			case '*':
				return cur * operand, nil
			case '/':
				if operand == 0 {
					return cur, fmt.Errorf("division by zero in %q", s)
				}
				return cur / operand, nil
			}
		}
		// No operator prefix: treat as an absolute set.
		f, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return cur, fmt.Errorf("%q is not a number or a + - * / operation", s)
		}
		return f, nil
	default:
		return cur, fmt.Errorf("must be a number or a string like \"*0.9\"")
	}
}

// adjustColor nudges the OKLCh channels (and alpha) of input per the adjustments map, then serializes back to hex. Lightness and alpha are clamped to [0,1], chroma to >= 0, and hue wraps modulo 360. Working in OKLCh means "lighten"/"darken"/"saturate"/"rotate hue" are all the same operation on different channels.
func adjustColor(input string, adjustments map[string]attr.Value) (string, error) {
	c, alpha, err := parseColor(input)
	if err != nil {
		return "", err
	}
	l, chroma, hue := oklchOf(c)

	for k, v := range adjustments {
		switch k {
		case "lightness":
			l, err = applyChannelOp(l, v)
		case "chroma":
			chroma, err = applyChannelOp(chroma, v)
		case "hue":
			hue, err = applyChannelOp(hue, v)
		case "alpha":
			alpha, err = applyChannelOp(alpha, v)
		default:
			return "", fmt.Errorf("unknown channel %q; adjustable channels are lightness, chroma, hue, alpha", k)
		}
		if err != nil {
			return "", fmt.Errorf("%s: %w", k, err)
		}
	}

	l = clamp01(l)
	if chroma < 0 {
		chroma = 0
	}
	hue = math.Mod(hue, 360)
	if hue < 0 {
		hue += 360
	}
	alpha = clamp01(alpha)

	return hexOut(colorful.OkLch(l, chroma, hue), alpha, true, false), nil
}

//go:embed descriptions/color_adjust.md
var colorAdjustDescription string

var _ function.Function = (*ColorAdjustFunction)(nil)

type ColorAdjustFunction struct{}

func NewColorAdjustFunction() function.Function { return &ColorAdjustFunction{} }

func (f *ColorAdjustFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "color_adjust"
}

func (f *ColorAdjustFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Nudge a color's OKLCh channels (lighten, darken, saturate, rotate hue, fade)",
		MarkdownDescription: colorAdjustDescription,
		Parameters: []function.Parameter{
			function.StringParameter{Name: "color", Description: "The color to adjust, in any CSS notation."},
			function.DynamicParameter{
				Name:        "adjustments",
				Description: "An object of channel adjustments. Keys: `lightness` (OKLCh L, 0-1), `chroma` (OKLCh C, >= 0), `hue` (degrees), `alpha` (0-1). Each value is a number (set absolutely) or a string operation: `\"+0.1\"`, `\"-0.1\"`, `\"*0.9\"`, `\"/2\"`.",
			},
		},
		Return: function.StringReturn{},
	}
}

func (f *ColorAdjustFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var input string
	var adj types.Dynamic
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &input, &adj))
	if resp.Error != nil {
		return
	}
	if hasUnknown(adj) {
		resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, types.StringUnknown()))
		return
	}
	obj, ok := adj.UnderlyingValue().(basetypes.ObjectValue)
	if !ok || obj.IsNull() {
		resp.Error = function.NewArgumentFuncError(1, `adjustments must be an object, e.g. { lightness = "*0.9" }`)
		return
	}
	out, err := adjustColor(input, obj.Attributes())
	if err != nil {
		resp.Error = function.NewArgumentFuncError(1, err.Error())
		return
	}
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &out))
}
