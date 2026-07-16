/*
Remove the common leading whitespace from every line of a string.

The classic `textwrap.dedent` operation: find the longest run of leading whitespace shared by all non-blank lines and strip it from each line, leaving relative indentation intact. Whitespace-only lines don't count toward the common margin and are normalized to empty. Useful for embedding indented heredocs (cloud-init, scripts, YAML/JSON blocks) in HCL and getting clean, left-aligned output. Terraform ships `indent` but no inverse.
*/

package text

import (
	"context"
	_ "embed"
	"fmt"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/function"
)

// dedentString strips the longest common leading-whitespace prefix shared by all
// non-blank lines. Whitespace-only lines are normalized to empty and ignored
// when computing the margin. Tabs and spaces must match exactly to be common.
func dedentString(s string) string {
	lines := strings.Split(s, "\n")
	margin := ""
	haveMargin := false
	for i, line := range lines {
		if strings.TrimLeft(line, " \t") == "" {
			// Blank or whitespace-only line: normalize to empty, don't let it
			// constrain the margin.
			lines[i] = ""
			continue
		}
		indent := line[:len(line)-len(strings.TrimLeft(line, " \t"))]
		if !haveMargin {
			margin, haveMargin = indent, true
		} else {
			margin = commonPrefix(margin, indent)
		}
	}
	if margin == "" {
		return strings.Join(lines, "\n")
	}
	for i, line := range lines {
		if line != "" {
			lines[i] = strings.TrimPrefix(line, margin)
		}
	}
	return strings.Join(lines, "\n")
}

func commonPrefix(a, b string) string {
	n := len(a)
	if len(b) < n {
		n = len(b)
	}
	i := 0
	for i < n && a[i] == b[i] {
		i++
	}
	return a[:i]
}

var _ function.Function = (*DedentFunction)(nil)

//go:embed descriptions/dedent.md
var dedentDescription string

type DedentFunction struct{}

func NewDedentFunction() function.Function { return &DedentFunction{} }

func (f *DedentFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "dedent"
}

func (f *DedentFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary:             "Remove common leading whitespace from every line",
		MarkdownDescription: dedentDescription,
		Parameters: []function.Parameter{
			function.StringParameter{Name: "s", Description: "The string to dedent."},
		},
		Return: function.StringReturn{},
	}
}

func (f *DedentFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var s string
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &s))
	if resp.Error != nil {
		return
	}
	if len(s) > textMaxInputBytes {
		resp.Error = function.NewArgumentFuncError(0, fmt.Sprintf("s exceeds maximum supported length of %d bytes", textMaxInputBytes))
		return
	}
	out := dedentString(s)
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &out))
}
