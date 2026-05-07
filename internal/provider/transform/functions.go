package transform

import "github.com/hashicorp/terraform-plugin-framework/function"

// Functions returns the data-transformation provider functions registered by terraform-burnham (queries and patches over decoded structures).
func Functions() []func() function.Function {
	return []func() function.Function{
		NewJMESPathQueryFunction,
		NewJSONPathQueryFunction,
		NewJSONPatchFunction,
		NewJSONMergePatchFunction,
	}
}
