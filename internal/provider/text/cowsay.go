/*
ASCII-art cowsay — render a string inside the speech bubble of a cow drawn in ASCII.

Self-contained implementation, no third-party cowsay dependency. We support the speech-bubble layout exactly per the original cowsay(1) (multi-line bubbles use `/ \` corners and `| |` sides; single-line uses `< >`), the standard "say" vs "think" distinction (different bubble brackets and a different connector to the cow), and the default cow figure. Customisable eyes (`oo` by default) and tongue (off by default) — same knobs as upstream `-e` and `-T`.

Custom cow shapes (the upstream `.cow` files for `tux`, `dragon`, etc.) are intentionally out of scope: the small file format would either need to be embedded statically or shipped separately, and the value-add over the default cow is mostly aesthetic.
*/

package text

import (
	"context"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/mitchellh/go-wordwrap"
)

var _ function.Function = (*CowsayFunction)(nil)

type CowsayFunction struct{}

func NewCowsayFunction() function.Function { return &CowsayFunction{} }

func (f *CowsayFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "cowsay"
}

func (f *CowsayFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Render a message inside the ASCII speech bubble of a cow",
		MarkdownDescription: "Returns `message` rendered as the original `cowsay(1)` would: a multi-line speech bubble (or thought bubble) attached to an ASCII cow figure. Useful for embedding in `/etc/motd` via cloud-init, login banners, or anywhere a generated config benefits from a recognizable greeting.\n\nOptions object:\n\n- `action` (string) — `\"say\"` (default; uses `< >` brackets and a `\\` connector) or `\"think\"` (uses `( )` brackets and `o` connectors).\n- `eyes` (string) — exactly two characters used for the cow's eyes. Default `\"oo\"`. Common alternatives: `\"==\"` (stoned), `\"@@\"` (paranoid), `\"--\"` (dead), `\"$$\"` (greedy), `\"OO\"` (surprised).\n- `tongue` (string) — exactly two characters (or empty for no tongue). Default empty. Common: `\"U \"` (sticking out), `\"V \"` (vampire).\n- `width` (number) — wrap the input message to this many columns before rendering. Default `40`. Set to `0` to disable wrapping (lines stay as you wrote them).\n\nMessage lines are word-wrapped at `width` codepoints by default, matching upstream cowsay's `-W` option.",
		Parameters: []function.Parameter{
			function.StringParameter{Name: "message", Description: "The message to render in the bubble."},
		},
		VariadicParameter: function.DynamicParameter{
			Name:        "options",
			Description: "Optional options object: { action = \"say\"|\"think\", eyes = string, tongue = string, width = number }. At most one allowed.",
		},
		Return: function.StringReturn{},
	}
}

type cowsayOpts struct {
	action string
	eyes   string
	tongue string
	width  int
}

func parseCowsayOptions(opts []types.Dynamic) (cowsayOpts, *function.FuncError) {
	out := cowsayOpts{action: "say", eyes: "oo", tongue: "", width: 40}
	if len(opts) == 0 {
		return out, nil
	}
	if len(opts) > 1 {
		return out, function.NewArgumentFuncError(1, "at most one options argument may be provided")
	}
	obj, ok := opts[0].UnderlyingValue().(basetypes.ObjectValue)
	if !ok || obj.IsNull() || obj.IsUnknown() {
		return out, function.NewArgumentFuncError(1, "options must be an object literal, e.g. { action = \"think\" }")
	}
	for k, val := range obj.Attributes() {
		switch k {
		case "action":
			s, ok := val.(basetypes.StringValue)
			if !ok || s.IsNull() {
				return out, function.NewArgumentFuncError(1, "options.action must be a string")
			}
			a := s.ValueString()
			if a != "say" && a != "think" {
				return out, function.NewArgumentFuncError(1, fmt.Sprintf("options.action must be \"say\" or \"think\"; received %q", a))
			}
			out.action = a
		case "eyes":
			s, ok := val.(basetypes.StringValue)
			if !ok || s.IsNull() {
				return out, function.NewArgumentFuncError(1, "options.eyes must be a string")
			}
			e := s.ValueString()
			if n := utf8.RuneCountInString(e); n != 2 {
				return out, function.NewArgumentFuncError(1, fmt.Sprintf("options.eyes must be exactly 2 characters; received %q (%d)", e, n))
			}
			out.eyes = e
		case "tongue":
			s, ok := val.(basetypes.StringValue)
			if !ok || s.IsNull() {
				return out, function.NewArgumentFuncError(1, "options.tongue must be a string")
			}
			tg := s.ValueString()
			if n := utf8.RuneCountInString(tg); tg != "" && n != 2 {
				return out, function.NewArgumentFuncError(1, fmt.Sprintf("options.tongue must be exactly 2 characters or empty; received %q (%d)", tg, n))
			}
			out.tongue = tg
		case "width":
			n, err := numberAttrToInt(val)
			if err != nil {
				return out, function.NewArgumentFuncError(1, "options.width must be a whole number: "+err.Error())
			}
			if n < 0 {
				return out, function.NewArgumentFuncError(1, fmt.Sprintf("options.width must be >= 0; received %d", n))
			}
			out.width = n
		default:
			return out, function.NewArgumentFuncError(1, fmt.Sprintf("unknown option key %q; supported keys are action, eyes, tongue, width", k))
		}
	}
	return out, nil
}

func (f *CowsayFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var message string
	var optsArgs []types.Dynamic
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &message, &optsArgs))
	if resp.Error != nil {
		return
	}
	opts, ferr := parseCowsayOptions(optsArgs)
	if ferr != nil {
		resp.Error = ferr
		return
	}

	// width == 0 disables wrapping; the user's existing line breaks are preserved as-is.
	wrapped := message
	if opts.width > 0 {
		wrapped = wordwrap.WrapString(message, uint(opts.width))
	}
	lines := strings.Split(wrapped, "\n")
	out := renderBubble(lines, opts.action) + renderCow(opts.action, opts.eyes, opts.tongue)
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &out))
}

// renderBubble draws the speech / thought bubble around a list of pre-wrapped message lines, returning the multi-line bubble string with a trailing newline. action == "say" → < > brackets for one-line, / \ and | | for multi-line; action == "think" → ( ) everywhere.
func renderBubble(lines []string, action string) string {
	if len(lines) == 0 {
		lines = []string{""}
	}
	widths := make([]int, len(lines))
	maxW := 0
	for i, l := range lines {
		w := utf8.RuneCountInString(l)
		widths[i] = w
		if w > maxW {
			maxW = w
		}
	}

	openSingle, closeSingle := '<', '>'
	openTop, closeTop := '/', '\\'
	openMid, closeMid := '|', '|'
	openBot, closeBot := '\\', '/'
	if action == "think" {
		openSingle, closeSingle = '(', ')'
		openTop, closeTop = '(', ')'
		openMid, closeMid = '(', ')'
		openBot, closeBot = '(', ')'
	}

	var b strings.Builder
	b.Grow((maxW + 6) * (len(lines) + 2))
	b.WriteByte(' ')
	for i := 0; i < maxW+2; i++ {
		b.WriteByte('_')
	}
	b.WriteByte('\n')

	if len(lines) == 1 {
		fmt.Fprintf(&b, "%c %s%s %c\n", openSingle, lines[0], strings.Repeat(" ", maxW-widths[0]), closeSingle)
	} else {
		for i, l := range lines {
			padded := l + strings.Repeat(" ", maxW-widths[i])
			open, closeR := openMid, closeMid
			switch i {
			case 0:
				open, closeR = openTop, closeTop
			case len(lines) - 1:
				open, closeR = openBot, closeBot
			}
			fmt.Fprintf(&b, "%c %s %c\n", open, padded, closeR)
		}
	}

	b.WriteByte(' ')
	for i := 0; i < maxW+2; i++ {
		b.WriteByte('-')
	}
	b.WriteByte('\n')

	return b.String()
}

// renderCow draws the connector (a `\` for "say", `o` for "think") and the standard cow figure with substitutable eyes and tongue.
func renderCow(action, eyes, tongue string) string {
	connector := `\`
	if action == "think" {
		connector = "o"
	}
	if tongue == "" {
		tongue = "  "
	}
	// 4-space indent before the connector matches upstream cowsay default cow.
	return fmt.Sprintf(`        %s   ^__^
         %s  (%s)\_______
            (__)\       )\/\
             %s ||----w |
                ||     ||
`, connector, connector, eyes, tongue)
}

