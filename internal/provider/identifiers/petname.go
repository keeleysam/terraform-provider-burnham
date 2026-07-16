/*
Deterministic Heroku-style "petname": short, human-friendly identifiers like `swift-fox-42`. Useful for naming resources, dashboards, branches, or anything else where a memorable label beats a UUID.

Same shape as upstream `dustinkirkland/golang-petname`: 1 word is a noun, 2 is `adjective-noun`, 3 is `adverb-adjective-noun`, and 4+ stack additional adverbs in front (`gently-swift-amber-fox`). Where this implementation diverges is **determinism**: rather than drawing words from `math/rand`, we derive the word indices from a SHA-256 HMAC keyed by the caller-supplied `seed`. Same seed always yields the same petname. That property is the whole point at plan time: Terraform plans must not churn on re-apply.

Wordlists live in `petname_words.go` (64 entries each, sized for unbiased byte-modulo selection).
*/

package identifiers

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	_ "embed"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-framework/types/basetypes"
	"github.com/keeleysam/terraform-burnham/internal/provider/optionsutil"
)

const (
	petnameDefaultWords     = 2
	petnameDefaultSeparator = "-"
	petnameMaxWords         = 16
)

var _ function.Function = (*PetnameFunction)(nil)

type PetnameFunction struct{}

func NewPetnameFunction() function.Function { return &PetnameFunction{} }

func (f *PetnameFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "petname"
}

//go:embed descriptions/petname.md
var petnameDescription string

func (f *PetnameFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Deterministic Heroku-style petname (\"swift-fox\") derived from a seed",
		MarkdownDescription: petnameDescription,
		Parameters: []function.Parameter{
			function.StringParameter{
				Name:        "seed",
				Description: "Stable seed string. The empty string is allowed and produces a deterministic petname; for unique petnames per resource, use a per-resource seed.",
			},
		},
		VariadicParameter: function.DynamicParameter{
			Name:        "options",
			Description: "Optional options object: { words = number, separator = string }. At most one allowed.",
		},
		Return: function.StringReturn{},
	}
}

// petnameOptions parses the optional options object into (words, separator).
func petnameOptions(opts []types.Dynamic) (int, string, *function.FuncError) {
	words := petnameDefaultWords
	separator := petnameDefaultSeparator
	attrs, ferr := optionsutil.SingleOptionsObject(opts, "{ words = 3 }")
	if ferr != nil {
		return 0, "", ferr
	}
	for k, val := range attrs {
		switch k {
		case "words":
			n, err := optionsutil.NumberAttrToInt(val)
			if err != nil {
				return 0, "", function.NewArgumentFuncError(1, "options.words must be a whole number: "+err.Error())
			}
			words = n
		case "separator":
			s, ok := val.(basetypes.StringValue)
			if !ok || s.IsNull() {
				return 0, "", function.NewArgumentFuncError(1, "options.separator must be a string")
			}
			separator = s.ValueString()
		default:
			return 0, "", function.NewArgumentFuncError(1, fmt.Sprintf("unknown option key %q; supported keys are words, separator", k))
		}
	}
	return words, separator, nil
}

func (f *PetnameFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var seed string
	var optsArgs []types.Dynamic
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &seed, &optsArgs))
	if resp.Error != nil {
		return
	}
	words, separator, ferr := petnameOptions(optsArgs)
	if ferr != nil {
		resp.Error = ferr
		return
	}
	if words < 1 || words > petnameMaxWords {
		resp.Error = function.NewArgumentFuncError(1, fmt.Sprintf("words must be in [1, %d]; received %d", petnameMaxWords, words))
		return
	}

	mac := hmac.New(sha256.New, []byte(seed))
	mac.Write([]byte("burnham/petname"))
	digest := mac.Sum(nil)
	// One digest byte per word position. petnameMaxWords (16) ≤ sha256 size (32), so a single block always suffices.

	out := make([]string, 0, words)
	if words == 1 {
		out = append(out, petnameNouns[digest[0]%64])
	} else {
		// (words-2) adverbs, one adjective, one noun. For words=2 the loop runs zero times and we get adjective + noun.
		for i := 0; i < words-2; i++ {
			out = append(out, petnameAdverbs[digest[i]%64])
		}
		out = append(out, petnameAdjectives[digest[words-2]%64])
		out = append(out, petnameNouns[digest[words-1]%64])
	}

	result := strings.Join(out, separator)
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &result))
}
