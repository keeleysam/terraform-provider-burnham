/*
Word-wrap a string to a given column width.

The classic "respect existing newlines, break on whitespace, never split a word longer than the width" word-wrap, exactly the algorithm in `github.com/mitchellh/go-wordwrap`. Used here mostly for cleaning up cloud-init MOTDs and embedded shell scripts — long lines turn into something pleasant to read in a terminal.

Width counts in Unicode codepoints. East-Asian double-width characters are *not* visually doubled here; if you need terminal-cell width awareness for box-drawing or CJK layout, post-process with a width-aware library.
*/

package text

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/mitchellh/go-wordwrap"
)

var _ function.Function = (*WrapFunction)(nil)

type WrapFunction struct{}

func NewWrapFunction() function.Function { return &WrapFunction{} }

func (f *WrapFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "wrap"
}

func (f *WrapFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Word-wrap a string to a given column width",
		MarkdownDescription: "Returns `s` re-wrapped to lines of at most `width` columns. Whitespace is the only break point; existing newlines are preserved. Words longer than `width` are not split — they overflow on their own line, matching the standard Unix `fmt(1)` behaviour and what every editor's word-wrap mode does.\n\nWidth is counted in Unicode codepoints; this function is not aware of terminal-cell width for double-width East-Asian characters or zero-width modifiers. For terminal layout that depends on visual width, post-process with a width-aware library.\n\nBacked by [`github.com/mitchellh/go-wordwrap`](https://github.com/mitchellh/go-wordwrap).",
		Parameters: []function.Parameter{
			function.StringParameter{Name: "s", Description: "The string to wrap."},
			function.Int64Parameter{Name: "width", Description: "Maximum line width in codepoints; must be >= 1."},
		},
		Return: function.StringReturn{},
	}
}

func (f *WrapFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var s string
	var width int64
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &s, &width))
	if resp.Error != nil {
		return
	}
	if width < 1 {
		resp.Error = function.NewArgumentFuncError(1, fmt.Sprintf("width must be >= 1; received %d", width))
		return
	}
	out := wordwrap.WrapString(s, uint(width))
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &out))
}
