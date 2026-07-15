package cedar

import "github.com/hashicorp/terraform-plugin-framework/function"

// Functions returns the Cedar provider functions registered by terraform-burnham: cedarencode (build a Cedar policy from its EST data tree), cedardecode (parse a policy into its EST), cedarvalidate (report whether a document is valid), cedarformat (canonicalize a document), and cedarevaluate (authorize a request against a document).
func Functions() []func() function.Function {
	return []func() function.Function{
		NewCedarEncodeFunction,
		NewCedarDecodeFunction,
		NewCedarValidateFunction,
		NewCedarFormatFunction,
		NewCedarEvaluateFunction,
	}
}
