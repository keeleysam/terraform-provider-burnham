package dataformat

import "github.com/hashicorp/terraform-plugin-framework/function"

// Functions returns the structured-data-format provider functions registered
// by terraform-burnham (JSON, plist, hujson, INI, CSV, YAML, .reg, VDF, KDL).
func Functions() []func() function.Function {
	return []func() function.Function{
		NewJSONEncodeFunction,
		NewHuJSONDecodeFunction,
		NewHuJSONEncodeFunction,
		NewPlistDecodeFunction,
		NewPlistEncodeFunction,
		NewPlistDateFunction,
		NewPlistDataFunction,
		NewPlistRealFunction,
		NewINIDecodeFunction,
		NewINIEncodeFunction,
		NewCSVEncodeFunction,
		NewYAMLEncodeFunction,
		NewRegDecodeFunction,
		NewRegEncodeFunction,
		NewRegDwordFunction,
		NewRegQwordFunction,
		NewRegBinaryFunction,
		NewRegMultiFunction,
		NewRegExpandSzFunction,
		NewVDFDecodeFunction,
		NewVDFEncodeFunction,
		NewKDLDecodeFunction,
		NewKDLEncodeFunction,
	}
}
