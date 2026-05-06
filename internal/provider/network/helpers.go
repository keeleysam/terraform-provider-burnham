package network

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
