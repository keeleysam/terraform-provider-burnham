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

func (c formatConfig) unparserOpts() []parser.UnparserOption {
	var o []parser.UnparserOption
	if c.wrapCol > 0 {
		o = append(o, parser.WrapOnColumn(c.wrapCol))
	}
	if c.wrapOps != nil {
		o = append(o, parser.WrapOnOperators(celOperatorIDs(c.wrapOps)...))
	}
	if c.wrapAfter != nil {
		o = append(o, parser.WrapAfterColumnLimit(*c.wrapAfter))
	}
	return o
}

// celLogicalSymbols augments operators.Find with the operators cel-go's symbol table omits: Find only knows arithmetic and comparison symbols, not the logical/ternary ones, which are the operators worth wrapping on. This lets the friendly "&&" / "||" a user writes reach WrapOnOperators as the IDs it wants.
var celLogicalSymbols = map[string]string{
	"&&": operators.LogicalAnd,
	"||": operators.LogicalOr,
	"?:": operators.Conditional,
}

// celOperatorIDs maps friendly operator symbols to the cel-go unparser's internal operator IDs. cel-go's WrapOnOperators wants the ID form ("_&&_"), but wrap_on_operators is documented to take the symbol a user actually writes ("&&"), so we translate. A value already in ID form (or any symbol cel-go doesn't know) passes through unchanged, so cel-go still reports it if it is genuinely unsupported.
func celOperatorIDs(symbols []string) []string {
	ids := make([]string, len(symbols))
	for i, s := range symbols {
		if id, ok := operators.Find(s); ok {
			ids[i] = id
			continue
		}
		if id, ok := celLogicalSymbols[s]; ok {
			ids[i] = id
			continue
		}
		ids[i] = s
	}
	return ids
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
		/*
		   Reparse the canonical seed before the final unparse so comprehension macros (map/exists/filter/optMap/…) are recognized.
		   A directly-built encoder AST carries its macros as plain member calls, and cel-go's unparser conservatively parenthesizes any 2+ argument call under a unary operator, so it would emit -(l.map(x, c)) instead of -l.map(x, c).
		   Reparsing first lets it emit precedence-minimal parentheses, which matches the pretty path below (that path already reparses) so the two canonical paths stay equivalent.
		*/
		seed, err := parser.Unparse(expr, si)
		if err != nil {
			return "", err
		}
		reAST, err := reparse(seed)
		if err != nil {
			return "", err
		}
		return parser.Unparse(reAST.Expr(), reAST.SourceInfo(), cfg.unparserOpts()...)
	}

	col := cfg.wrapCol
	if col <= 0 {
		col = 80
	}
	uo := []parser.UnparserOption{parser.WrapOnColumn(col)}
	if cfg.wrapOps != nil {
		uo = append(uo, parser.WrapOnOperators(celOperatorIDs(cfg.wrapOps)...))
	}
	if cfg.wrapAfter != nil {
		uo = append(uo, parser.WrapAfterColumnLimit(*cfg.wrapAfter))
	}
	seed, err := parser.Unparse(expr, si, uo...)
	if err != nil {
		return "", err
	}

	reAST, err := reparse(seed)
	if err != nil {
		return "", err
	}

	fo := []celfmt.FormatOption{celfmt.Pretty()}
	if cfg.indent != nil {
		fo = append(fo, celfmt.IndentString(*cfg.indent))
	}
	if cfg.alwaysComma {
		fo = append(fo, celfmt.AlwaysComma())
	}
	var b bytes.Buffer
	if err := celfmt.Format(&b, reAST, common.NewTextSource(seed), fo...); err != nil {
		return "", err
	}
	return b.String(), nil
}

// reparse parses a canonical CEL seed string under the lenient environment with macro-call tracking enabled, returning the native AST. Both formatExpr paths use it so comprehension macros are recognized (which is what yields precedence-minimal parentheses) before the final unparse or celfmt pass.
func reparse(seed string) (*ast.AST, error) {
	env, err := newParseEnv(true)
	if err != nil {
		return nil, err
	}
	parsed, iss := env.Parse(seed)
	if iss != nil && iss.Err() != nil {
		return nil, iss.Err()
	}
	return parsed.NativeRep(), nil
}
