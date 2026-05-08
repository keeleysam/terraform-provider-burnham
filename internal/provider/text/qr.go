/*
QR code rendered as ASCII / Unicode art.

Useful for putting a wifi-credentials QR code in `/etc/motd`, embedding an onboarding URL into a generated PDF source, or generally anywhere a Terraform plan should produce a scannable code without involving an image asset. The output is a fixed-pitch text block; viewers with sufficient contrast and the correct font (anything monospaced) can scan it directly from the terminal.

Backed by [`rsc.io/qr`](https://pkg.go.dev/rsc.io/qr) — a small, dependency-free QR encoder by Russ Cox. We render the resulting bitmap with Unicode block characters: the upper half-block `▀` (U+2580) lets one terminal row encode two QR module rows, halving the vertical footprint, which is what most QR-in-terminal tools do.
*/

package text

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"rsc.io/qr"
)

var _ function.Function = (*QRAsciiFunction)(nil)

type QRAsciiFunction struct{}

func NewQRAsciiFunction() function.Function { return &QRAsciiFunction{} }

func (f *QRAsciiFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "qr_ascii"
}

func (f *QRAsciiFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Render a QR code as compact Unicode-block ASCII art",
		MarkdownDescription: "Returns a multi-line string containing a QR code that encodes `payload`, rendered with Unicode half-block characters so two QR-module rows fit in one terminal row. Scannable directly from any monospaced display with adequate light/dark contrast (white-on-black terminals work; light themes need an inverted variant — see `style`).\n\nOptions object:\n\n- `error_correction` (string) — error-correction level, one of `\"L\"` (default, ~7%), `\"M\"` (~15%), `\"Q\"` (~25%), `\"H\"` (~30%). Higher levels survive more occlusion at the cost of a bigger code.\n- `quiet_zone` (number) — number of empty modules around the code. Default `4` (the [QR spec](https://en.wikipedia.org/wiki/QR_code) minimum). Set to `0` for very tight layouts.\n- `style` (string) — `\"dark_on_light\"` (default; dark modules render as `▀ █ ▄`, light as space — for white terminals) or `\"light_on_dark\"` (inverted, for black terminals).\n\nLayout is half-block: each terminal line covers two QR module rows. For payloads above ~150 characters you'll start hitting QR version limits at error_correction=`L`; bump to `H` only for short payloads where the size cost is acceptable.",
		Parameters: []function.Parameter{
			function.StringParameter{Name: "payload", Description: "The payload to encode. Most QR readers handle URLs, wifi-config strings (\"WIFI:T:WPA;S:ssid;P:pass;;\"), and arbitrary text up to a few hundred bytes."},
		},
		VariadicParameter: function.DynamicParameter{
			Name:        "options",
			Description: "Optional options object: { error_correction = \"L\"|\"M\"|\"Q\"|\"H\", quiet_zone = number, style = \"dark_on_light\"|\"light_on_dark\" }. At most one allowed.",
		},
		Return: function.StringReturn{},
	}
}

type qrOpts struct {
	level      qr.Level
	quietZone  int
	lightOnDark bool
}

func parseQROptions(opts []types.Dynamic) (qrOpts, *function.FuncError) {
	out := qrOpts{level: qr.L, quietZone: 4, lightOnDark: false}
	if len(opts) == 0 {
		return out, nil
	}
	if len(opts) > 1 {
		return out, function.NewArgumentFuncError(1, "at most one options argument may be provided")
	}
	obj, ok := opts[0].UnderlyingValue().(basetypes.ObjectValue)
	if !ok || obj.IsNull() || obj.IsUnknown() {
		return out, function.NewArgumentFuncError(1, "options must be an object literal, e.g. { error_correction = \"H\" }")
	}
	for k, val := range obj.Attributes() {
		switch k {
		case "error_correction":
			s, ok := val.(basetypes.StringValue)
			if !ok || s.IsNull() {
				return out, function.NewArgumentFuncError(1, "options.error_correction must be a string")
			}
			switch s.ValueString() {
			case "L":
				out.level = qr.L
			case "M":
				out.level = qr.M
			case "Q":
				out.level = qr.Q
			case "H":
				out.level = qr.H
			default:
				return out, function.NewArgumentFuncError(1, fmt.Sprintf("options.error_correction must be \"L\", \"M\", \"Q\", or \"H\"; received %q", s.ValueString()))
			}
		case "quiet_zone":
			n, err := numberAttrToInt(val)
			if err != nil {
				return out, function.NewArgumentFuncError(1, "options.quiet_zone must be a whole number: "+err.Error())
			}
			if n < 0 || n > 64 {
				return out, function.NewArgumentFuncError(1, fmt.Sprintf("options.quiet_zone must be in [0, 64]; received %d", n))
			}
			out.quietZone = n
		case "style":
			s, ok := val.(basetypes.StringValue)
			if !ok || s.IsNull() {
				return out, function.NewArgumentFuncError(1, "options.style must be a string")
			}
			switch s.ValueString() {
			case "dark_on_light":
				out.lightOnDark = false // dark modules render as dark blocks; for white terminals
			case "light_on_dark":
				out.lightOnDark = true // dark modules render as light blocks; for black terminals
			default:
				return out, function.NewArgumentFuncError(1, fmt.Sprintf("options.style must be \"dark_on_light\" or \"light_on_dark\"; received %q", s.ValueString()))
			}
		default:
			return out, function.NewArgumentFuncError(1, fmt.Sprintf("unknown option key %q; supported keys are error_correction, quiet_zone, style", k))
		}
	}
	return out, nil
}

func (f *QRAsciiFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var payload string
	var optsArgs []types.Dynamic
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &payload, &optsArgs))
	if resp.Error != nil {
		return
	}
	opts, ferr := parseQROptions(optsArgs)
	if ferr != nil {
		resp.Error = ferr
		return
	}

	code, err := qr.Encode(payload, opts.level)
	if err != nil {
		resp.Error = function.NewArgumentFuncError(0, fmt.Sprintf("encoding QR code: %s", err.Error()))
		return
	}

	out := renderHalfBlock(code, opts.quietZone, opts.lightOnDark)
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &out))
}

// renderHalfBlock prints a QR code as half-block Unicode art. Each terminal row encodes two QR module rows: top-half lit + bottom-half lit + neither + both = `▀` `▄` ` ` `█`. For "light_on_dark" the dark/light roles are swapped.
func renderHalfBlock(code *qr.Code, quietZone int, lightOnDark bool) string {
	size := code.Size
	full := size + 2*quietZone

	// Module accessor with quiet-zone padding. true = dark (foreground) module.
	at := func(x, y int) bool {
		if x < quietZone || y < quietZone || x >= quietZone+size || y >= quietZone+size {
			return false
		}
		return code.Black(x-quietZone, y-quietZone)
	}

	var b strings.Builder
	b.Grow(full * (full/2 + 1) * 4)
	for y := 0; y < full; y += 2 {
		for x := 0; x < full; x++ {
			top := at(x, y)
			bot := false
			if y+1 < full {
				bot = at(x, y+1)
			}
			if lightOnDark {
				top = !top
				bot = !bot
			}
			switch {
			case top && bot:
				b.WriteString("█")
			case top && !bot:
				b.WriteString("▀")
			case !top && bot:
				b.WriteString("▄")
			default:
				b.WriteByte(' ')
			}
		}
		b.WriteByte('\n')
	}
	return b.String()
}
