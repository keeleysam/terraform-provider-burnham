package regex

import "github.com/hashicorp/terraform-plugin-framework/function"

// Functions returns the regex provider-defined functions registered by
// terraform-burnham: pcre_match, pcre_captures, pcre_find_all, pcre_replace, and
// pcre_split. All use PCRE syntax (backreferences, lookaround) via fancy-regex,
// the features Terraform core's RE2-based regex functions cannot express.
func Functions() []func() function.Function {
	return []func() function.Function{
		NewPCREMatchFunction,
		NewPCRECapturesFunction,
		NewPCREFindAllFunction,
		NewPCREReplaceFunction,
		NewPCRESplitFunction,
	}
}
