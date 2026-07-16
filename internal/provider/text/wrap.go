/*
Word-wrap a string to a given column width.

The classic "respect existing newlines, break on whitespace, never split a word longer than the width" word-wrap, exactly the algorithm in `github.com/mitchellh/go-wordwrap`. Used here mostly for cleaning up cloud-init MOTDs and embedded shell scripts: long lines turn into something pleasant to read in a terminal.

Width counts in Unicode codepoints. East-Asian double-width characters are *not* visually doubled here; if you need terminal-cell width awareness for box-drawing or CJK layout, post-process with a width-aware library.
*/

package text

import (
	"context"
	_ "embed"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/mitchellh/go-wordwrap"
)

var _ function.Function = (*WrapFunction)(nil)

//go:embed descriptions/wrap.md
var wrapDescription string

type WrapFunction struct{}

func NewWrapFunction() function.Function { return &WrapFunction{} }

func (f *WrapFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "wrap"
}

func (f *WrapFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Word-wrap a string to a given column width",
		MarkdownDescription: wrapDescription,
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
	if len(s) > textMaxInputBytes {
		resp.Error = function.NewArgumentFuncError(0, fmt.Sprintf("s exceeds maximum supported length of %d bytes", textMaxInputBytes))
		return
	}
	out := wordwrap.WrapString(s, uint(width))
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &out))
}
