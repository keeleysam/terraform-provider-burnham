package network

import (
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// optionalArg returns the first element of args if any, otherwise defaultVal.
// Use this with the terraform-plugin-framework's VariadicParameter pattern,
// where an optional argument is exposed as a slice that's either empty or
// holds exactly one element.
func optionalArg[T any](args []T, defaultVal T) T {
	if len(args) == 0 {
		return defaultVal
	}
	return args[0]
}

// cidrListArg converts a list(string) argument into a []string, attributing a
// null or unknown element to the argument at argIndex instead of surfacing the
// framework's provider-blaming internal conversion error.
//
// Reading a list(string) argument straight into a []string makes
// terraform-plugin-framework reject a null element with a generic "This is
// always an error in the provider ... report to the provider developer"
// diagnostic that carries no argument index. Reading the argument into a
// types.List first lets us inspect each element and return a clear
// NewArgumentFuncError that points at the offending argument.
func cidrListArg(list types.List, argIndex int64, argName string) ([]string, *function.FuncError) {
	elems := list.Elements()
	out := make([]string, 0, len(elems))
	for i, e := range elems {
		s, ok := e.(types.String)
		if !ok {
			return nil, function.NewArgumentFuncError(argIndex, fmt.Sprintf("%s: element %d is not a string.", argName, i))
		}
		if s.IsNull() {
			return nil, function.NewArgumentFuncError(argIndex, fmt.Sprintf("%s: element %d is null; every CIDR in the list must be a non-null string.", argName, i))
		}
		if s.IsUnknown() {
			return nil, function.NewArgumentFuncError(argIndex, fmt.Sprintf("%s: element %d is unknown; every CIDR in the list must be a known string.", argName, i))
		}
		out = append(out, s.ValueString())
	}
	return out, nil
}
