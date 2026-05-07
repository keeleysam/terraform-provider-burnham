package dataformat

import "github.com/hashicorp/terraform-plugin-framework/function"

// Functions returns the structured-data-format provider functions registered by terraform-burnham.
func Functions() []func() function.Function {
	return []func() function.Function{
		// JSON family
		NewJSONEncodeFunction,
		NewHuJSONDecodeFunction,
		NewHuJSONEncodeFunction,
		NewNDJSONDecodeFunction,
		NewNDJSONEncodeFunction,
		// Apple plist
		NewPlistDecodeFunction,
		NewPlistEncodeFunction,
		NewPlistDateFunction,
		NewPlistDataFunction,
		NewPlistRealFunction,
		// Tabular / line-oriented
		NewINIDecodeFunction,
		NewINIEncodeFunction,
		NewCSVEncodeFunction,
		NewYAMLEncodeFunction,
		// Windows registry
		NewRegDecodeFunction,
		NewRegEncodeFunction,
		NewRegDwordFunction,
		NewRegQwordFunction,
		NewRegBinaryFunction,
		NewRegMultiFunction,
		NewRegExpandSzFunction,
		// Game / niche
		NewVDFDecodeFunction,
		NewVDFEncodeFunction,
		NewKDLDecodeFunction,
		NewKDLEncodeFunction,
		// Binary
		NewMsgpackDecodeFunction,
		NewMsgpackEncodeFunction,
		NewCBORDecodeFunction,
		NewCBOREncodeFunction,
		// Config files
		NewDotenvDecodeFunction,
		NewDotenvEncodeFunction,
		NewJavaPropertiesDecodeFunction,
		NewJavaPropertiesEncodeFunction,
		NewAppleStringsDecodeFunction,
		NewAppleStringsEncodeFunction,
		NewHCLDecodeFunction,
		NewHCLEncodeFunction,
	}
}
