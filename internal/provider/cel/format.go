package cel

import (
	"bytes"

	"github.com/elastic/celfmt"
	"github.com/google/cel-go/common"
	"github.com/google/cel-go/common/ast"
	"github.com/google/cel-go/common/operators"
	"github.com/google/cel-go/parser"
)

// formatConfig holds the resolved output-formatting options.
// Zero values mean "use the cel-go / celfmt default".
type formatConfig struct {
	wrapCol     int      // 0 = unset
	wrapOps     []string // nil = unset (cel-go default: && ||)
	wrapAfter   *bool    // nil = unset (cel-go default: true)
	pretty      bool
	indent      *string // nil = celfmt default (tab)
	alwaysComma bool
}

// FormatOption configures Encode/Validate output formatting.
// These mirror the cel-go unparser options plus celfmt's pretty-printing options, so if celfmt's features are ever upstreamed into cel-go we can swap the backend unchanged.
type FormatOption func(*formatConfig)

func WrapColumn(n int) FormatOption { return func(c *formatConfig) { c.wrapCol = n } }
func WrapOperators(s ...string) FormatOption {
	return func(c *formatConfig) { c.wrapOps = s }
}
func WrapAfter(b bool) FormatOption { return func(c *formatConfig) { c.wrapAfter = &b } }
func Pretty() FormatOption          { return func(c *formatConfig) { c.pretty = true } }
func Indent(s string) FormatOption  { return func(c *formatConfig) { c.indent = &s } }
func AlwaysComma() FormatOption     { return func(c *formatConfig) { c.alwaysComma = true } }

// logicalWrapOperatorIDs covers the operator symbols cel-go's operators.Find does not map (the logical and conditional operators), which are the defaults callers most often wrap on. Find already covers the arithmetic and comparison symbols.
var logicalWrapOperatorIDs = map[string]string{
	"&&": operators.LogicalAnd,
	"||": operators.LogicalOr,
	"?:": operators.Conditional,
}

// wrapOperatorIDs translates the human-friendly operator symbols the options accept ("&&", "==", ...) into the operator ids cel-go's parser.WrapOnOperators expects ("_&&_", "_==_", ...). A symbol cel-go does not recognize is passed through unchanged, so cel-go reports the error at unparse time.
func wrapOperatorIDs(symbols []string) []string {
	if symbols == nil {
		return nil
	}
	ids := make([]string, len(symbols))
	for i, s := range symbols {
		if id, ok := operators.Find(s); ok {
			ids[i] = id
			continue
		}
		if id, ok := logicalWrapOperatorIDs[s]; ok {
			ids[i] = id
			continue
		}
		ids[i] = s
	}
	return ids
}

func (c formatConfig) unparserOpts() []parser.UnparserOption {
	var o []parser.UnparserOption
	if c.wrapCol > 0 {
		o = append(o, parser.WrapOnColumn(c.wrapCol))
	}
	if c.wrapOps != nil {
		o = append(o, parser.WrapOnOperators(wrapOperatorIDs(c.wrapOps)...))
	}
	if c.wrapAfter != nil {
		o = append(o, parser.WrapAfterColumnLimit(*c.wrapAfter))
	}
	return o
}

// formatExpr renders expr (+ its SourceInfo) with the given options.
//
// Without pretty, it is a single cel-go unparser pass (single canonical line by default).
// With pretty, it uses a two-pass pipeline: cel-go wraps the expression at the column to introduce line breaks, the wrapped text is reparsed so the layout has source line positions, and celfmt indents from those.
// This is what makes generated (source-less) CEL indent, since celfmt's indentation is driven by source line positions.
func formatExpr(expr ast.Expr, si *ast.SourceInfo, opts ...FormatOption) (string, error) {
	var cfg formatConfig
	for _, o := range opts {
		o(&cfg)
	}

	if !cfg.pretty {
		return parser.Unparse(expr, si, cfg.unparserOpts()...)
	}

	col := cfg.wrapCol
	if col <= 0 {
		col = 80
	}
	uo := []parser.UnparserOption{parser.WrapOnColumn(col)}
	if cfg.wrapOps != nil {
		uo = append(uo, parser.WrapOnOperators(wrapOperatorIDs(cfg.wrapOps)...))
	}
	if cfg.wrapAfter != nil {
		uo = append(uo, parser.WrapAfterColumnLimit(*cfg.wrapAfter))
	}
	seed, err := parser.Unparse(expr, si, uo...)
	if err != nil {
		return "", err
	}

	env, err := newParseEnv(true)
	if err != nil {
		return "", err
	}
	reAST, iss := env.Parse(seed)
	if iss != nil && iss.Err() != nil {
		return "", iss.Err()
	}

	fo := []celfmt.FormatOption{celfmt.Pretty()}
	if cfg.indent != nil {
		fo = append(fo, celfmt.IndentString(*cfg.indent))
	}
	if cfg.alwaysComma {
		fo = append(fo, celfmt.AlwaysComma())
	}
	var b bytes.Buffer
	if err := celfmt.Format(&b, reAST.NativeRep(), common.NewTextSource(seed), fo...); err != nil {
		return "", err
	}
	return b.String(), nil
}
