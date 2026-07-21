package regex

import (
	"context"
	_ "embed"
	"encoding/json"
	"errors"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// engineFuncError maps a runOp error to the right function diagnostic. An EngineError is the regex engine reporting on the caller's own regex (an invalid pattern, or a runtime failure such as the backtrack limit tripping on a catastrophic pattern), so it points at the pattern argument (index 0). Any other error is an internal engine fault (a wasm trap, a missing result, a decode failure) and is reported as a general function error rather than wrongly blaming the caller's input.
func engineFuncError(err error) *function.FuncError {
	var ee *EngineError
	if errors.As(err, &ee) {
		return function.NewArgumentFuncError(0, ee.Msg)
	}
	return function.NewFuncError(err.Error())
}

// ── pcre_match ──────────────────────────────────────────────────

//go:embed descriptions/pcre_match.md
var pcreMatchDescription string

var _ function.Function = (*PCREMatchFunction)(nil)

type PCREMatchFunction struct{}

func NewPCREMatchFunction() function.Function { return &PCREMatchFunction{} }

func (f *PCREMatchFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "pcre_match"
}

func (f *PCREMatchFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Report whether a PCRE pattern matches anywhere in a string",
		MarkdownDescription: pcreMatchDescription,
		Parameters: []function.Parameter{
			function.StringParameter{Name: "pattern", Description: "The regular expression (PCRE syntax: backreferences and lookaround allowed). Use inline flags like `(?i)` for case-insensitive."},
			function.StringParameter{Name: "str", Description: "The string to test."},
		},
		Return: function.BoolReturn{},
	}
}

func (f *PCREMatchFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var pattern, str string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &pattern, &str))
	if resp.Error != nil {
		return
	}
	v, err := runOp(ctx, opMatch, pattern, str, "")
	if err != nil {
		resp.Error = engineFuncError(err)
		return
	}
	var b bool
	if err := json.Unmarshal(v, &b); err != nil {
		resp.Error = function.NewFuncError(err.Error())
		return
	}
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, b))
}

// ── pcre_captures ───────────────────────────────────────────────

//go:embed descriptions/pcre_captures.md
var pcreCapturesDescription string

var _ function.Function = (*PCRECapturesFunction)(nil)

type PCRECapturesFunction struct{}

func NewPCRECapturesFunction() function.Function { return &PCRECapturesFunction{} }

func (f *PCRECapturesFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "pcre_captures"
}

func (f *PCRECapturesFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Return the capture groups of the first PCRE match as a map",
		MarkdownDescription: pcreCapturesDescription,
		Parameters: []function.Parameter{
			function.StringParameter{Name: "pattern", Description: "The regular expression (PCRE syntax)."},
			function.StringParameter{Name: "str", Description: "The string to match against."},
		},
		Return: function.MapReturn{ElementType: types.StringType},
	}
}

func (f *PCRECapturesFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var pattern, str string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &pattern, &str))
	if resp.Error != nil {
		return
	}
	v, err := runOp(ctx, opCaptures, pattern, str, "")
	if err != nil {
		resp.Error = engineFuncError(err)
		return
	}
	// v is a JSON object of group -> matched text, or the JSON literal null when there is no match. Only groups that participated in the match are present, so numbered keys can be non-contiguous: an optional group that did not match (e.g. group 1 in `(a)?(b)` against "b") is omitted rather than being an empty string or null. runOp guarantees a non-nil v, so a bare "null" here is the no-match case and yields an empty map.
	m := map[string]string{}
	if string(v) != "null" {
		if err := json.Unmarshal(v, &m); err != nil {
			resp.Error = function.NewFuncError(err.Error())
			return
		}
	}
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, m))
}

// ── pcre_find_all ───────────────────────────────────────────────

//go:embed descriptions/pcre_find_all.md
var pcreFindAllDescription string

var _ function.Function = (*PCREFindAllFunction)(nil)

type PCREFindAllFunction struct{}

func NewPCREFindAllFunction() function.Function { return &PCREFindAllFunction{} }

func (f *PCREFindAllFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "pcre_find_all"
}

func (f *PCREFindAllFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Return every non-overlapping PCRE match in a string",
		MarkdownDescription: pcreFindAllDescription,
		Parameters: []function.Parameter{
			function.StringParameter{Name: "pattern", Description: "The regular expression (PCRE syntax)."},
			function.StringParameter{Name: "str", Description: "The string to search."},
		},
		Return: function.ListReturn{ElementType: types.StringType},
	}
}

func (f *PCREFindAllFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var pattern, str string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &pattern, &str))
	if resp.Error != nil {
		return
	}
	v, err := runOp(ctx, opFindAll, pattern, str, "")
	if err != nil {
		resp.Error = engineFuncError(err)
		return
	}
	out := []string{}
	if err := json.Unmarshal(v, &out); err != nil {
		resp.Error = function.NewFuncError(err.Error())
		return
	}
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, out))
}

// ── pcre_replace ────────────────────────────────────────────────

//go:embed descriptions/pcre_replace.md
var pcreReplaceDescription string

var _ function.Function = (*PCREReplaceFunction)(nil)

type PCREReplaceFunction struct{}

func NewPCREReplaceFunction() function.Function { return &PCREReplaceFunction{} }

func (f *PCREReplaceFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "pcre_replace"
}

func (f *PCREReplaceFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Replace every PCRE match, with backreferences in the replacement",
		MarkdownDescription: pcreReplaceDescription,
		Parameters: []function.Parameter{
			function.StringParameter{Name: "pattern", Description: "The regular expression (PCRE syntax)."},
			function.StringParameter{Name: "str", Description: "The string to transform."},
			function.StringParameter{Name: "replacement", Description: "The replacement, with `$1` / `${name}` referring to capture groups."},
		},
		Return: function.StringReturn{},
	}
}

func (f *PCREReplaceFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var pattern, str, replacement string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &pattern, &str, &replacement))
	if resp.Error != nil {
		return
	}
	v, err := runOp(ctx, opReplace, pattern, str, replacement)
	if err != nil {
		resp.Error = engineFuncError(err)
		return
	}
	var out string
	if err := json.Unmarshal(v, &out); err != nil {
		resp.Error = function.NewFuncError(err.Error())
		return
	}
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, out))
}

// ── pcre_split ──────────────────────────────────────────────────

//go:embed descriptions/pcre_split.md
var pcreSplitDescription string

var _ function.Function = (*PCRESplitFunction)(nil)

type PCRESplitFunction struct{}

func NewPCRESplitFunction() function.Function { return &PCRESplitFunction{} }

func (f *PCRESplitFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "pcre_split"
}

func (f *PCRESplitFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Split a string on a PCRE pattern",
		MarkdownDescription: pcreSplitDescription,
		Parameters: []function.Parameter{
			function.StringParameter{Name: "pattern", Description: "The regular expression (PCRE syntax) to split on."},
			function.StringParameter{Name: "str", Description: "The string to split."},
		},
		Return: function.ListReturn{ElementType: types.StringType},
	}
}

func (f *PCRESplitFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var pattern, str string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &pattern, &str))
	if resp.Error != nil {
		return
	}
	v, err := runOp(ctx, opSplit, pattern, str, "")
	if err != nil {
		resp.Error = engineFuncError(err)
		return
	}
	out := []string{}
	if err := json.Unmarshal(v, &out); err != nil {
		resp.Error = function.NewFuncError(err.Error())
		return
	}
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, out))
}
