// Package documents provides typesetting functions that turn Typst markup into
// rendered documents (PDF, PNG, SVG, HTML) at plan time. The Typst engine is
// compiled to WebAssembly and run under wazero, so the provider stays CGO-free
// and output is deterministic (except for documents that call non-deterministic
// Typst builtins such as datetime.today()).
package documents

import "github.com/hashicorp/terraform-plugin-framework/function"

// Functions returns the constructors for this family.
func Functions() []func() function.Function {
	return []func() function.Function{
		NewTypstPDFFunction,
		NewTypstPNGFunction,
		NewTypstSVGFunction,
		NewTypstHTMLFunction,
	}
}
