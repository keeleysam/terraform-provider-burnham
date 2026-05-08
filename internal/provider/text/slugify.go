/*
Unicode-aware slugification — turn `"Café au Lait №3"` into `"cafe-au-lait-3"`.

Different from Terraform's `replace()` + `lower()` and from corefunc's `str_kebab` (which is only a case-conversion): `slugify` transliterates accented and non-Latin characters into their nearest ASCII equivalent before lower-casing and joining with hyphens. That's the actual operation people want when they say "make this URL-safe".

Backed by [`github.com/gosimple/slug`](https://github.com/gosimple/slug), which keeps a comprehensive Unicode → ASCII transliteration table (handles dozens of scripts: Latin extensions, Cyrillic, Greek, CJK, Arabic, Hebrew, …).
*/

package text

import (
	"context"
	"fmt"
	"strings"
	"sync"

	gosimple "github.com/gosimple/slug"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/keeleysam/terraform-burnham/internal/provider/optionsutil"
)

// slugifyMu serializes access to gosimple/slug's package-level Lowercase global. Terraform's plan-time evaluation can call provider functions concurrently across the expression graph, and the upstream library only exposes its lowercase knob as a process-global. Holding a mutex around the swap is the cheapest fix; the alternative is forking the library or post-processing case ourselves, which loses the consistent-with-upstream guarantee.
var slugifyMu sync.Mutex

var _ function.Function = (*SlugifyFunction)(nil)

type SlugifyFunction struct{}

func NewSlugifyFunction() function.Function { return &SlugifyFunction{} }

func (f *SlugifyFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "slugify"
}

func (f *SlugifyFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Convert any string to a URL-safe slug, transliterating Unicode to ASCII",
		MarkdownDescription: "Returns a URL-safe slug derived from `s`. Lowercases the result, transliterates non-ASCII characters into their nearest ASCII equivalent (`café` → `cafe`, `Москва` → `moskva`, `北京` → `bei-jing`), strips remaining punctuation, and joins runs of word characters with hyphens.\n\n```\nslugify(\"Café au Lait №3\")  → \"cafe-au-lait-no3\"\nslugify(\"Hello, World!\")     → \"hello-world\"\n```\n\nOptions object:\n\n- `language` (string) — ISO 639-1 hint for transliteration (e.g. `\"en\"`, `\"de\"`, `\"ja\"`). The default heuristic produces good output for Latin-script input; pick a language to handle non-Latin input correctly. Library list of supported codes: see [gosimple/slug](https://github.com/gosimple/slug).\n- `separator` (string) — the joiner between words. Default `\"-\"`.\n- `lowercase` (bool) — lowercase the result. Default `true`.\n\nDifferent from Terraform's `replace()` + `lower()` and from corefunc's case-conversion functions: `slugify` does **transliteration**, not just case folding.",
		Parameters: []function.Parameter{
			function.StringParameter{Name: "s", Description: "The string to slugify."},
		},
		VariadicParameter: function.DynamicParameter{
			Name:        "options",
			Description: "Optional options object: { language = string, separator = string, lowercase = bool }. At most one allowed.",
		},
		Return: function.StringReturn{},
	}
}

type slugifyOpts struct {
	language  string
	separator string
	lowercase bool
}

func parseSlugifyOptions(opts []types.Dynamic) (slugifyOpts, *function.FuncError) {
	out := slugifyOpts{separator: "-", lowercase: true}
	attrs, ferr := optionsutil.SingleOptionsObject(opts, "{ lowercase = false }")
	if ferr != nil {
		return out, ferr
	}
	for k, val := range attrs {
		switch k {
		case "language":
			s, ok := val.(basetypes.StringValue)
			if !ok || s.IsNull() {
				return out, function.NewArgumentFuncError(1, "options.language must be a string")
			}
			out.language = s.ValueString()
		case "separator":
			s, ok := val.(basetypes.StringValue)
			if !ok || s.IsNull() {
				return out, function.NewArgumentFuncError(1, "options.separator must be a string")
			}
			out.separator = s.ValueString()
		case "lowercase":
			b, ok := val.(basetypes.BoolValue)
			if !ok || b.IsNull() {
				return out, function.NewArgumentFuncError(1, "options.lowercase must be a boolean")
			}
			out.lowercase = b.ValueBool()
		default:
			return out, function.NewArgumentFuncError(1, fmt.Sprintf("unknown option key %q; supported keys are language, separator, lowercase", k))
		}
	}
	return out, nil
}

func (f *SlugifyFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var s string
	var optsArgs []types.Dynamic
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &s, &optsArgs))
	if resp.Error != nil {
		return
	}
	opts, ferr := parseSlugifyOptions(optsArgs)
	if ferr != nil {
		resp.Error = ferr
		return
	}

	result := slugifyWith(s, opts)
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &result))
}

// slugifyWith calls gosimple/slug under a package-level lock so concurrent invocations don't race on its `Lowercase` package-global. gosimple/slug always emits "-" as the separator, so we post-process if a different one was requested.
func slugifyWith(s string, opts slugifyOpts) string {
	slugifyMu.Lock()
	prev := gosimple.Lowercase
	gosimple.Lowercase = opts.lowercase
	var result string
	if opts.language != "" {
		result = gosimple.MakeLang(s, opts.language)
	} else {
		result = gosimple.Make(s)
	}
	gosimple.Lowercase = prev
	slugifyMu.Unlock()

	if opts.separator != "-" {
		result = strings.ReplaceAll(result, "-", opts.separator)
	}
	return result
}
