package provider

import (
	"context"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func runPlistDecode(t *testing.T, input string) (types.Dynamic, *function.FuncError) {
	t.Helper()
	f := &PlistDecodeFunction{}

	args := function.NewArgumentsData([]attr.Value{types.StringValue(input)})
	req := function.RunRequest{Arguments: args}
	resp := &function.RunResponse{Result: function.NewResultData(types.DynamicNull())}

	f.Run(context.Background(), req, resp)

	if resp.Error != nil {
		return types.DynamicNull(), resp.Error
	}

	result, ok := resp.Result.Value().(types.Dynamic)
	if !ok {
		t.Fatalf("expected Dynamic result, got %T", resp.Result.Value())
	}
	return result, nil
}

const testPlistXML = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Name</key>
	<string>Test Profile</string>
	<key>Version</key>
	<integer>1</integer>
	<key>Enabled</key>
	<true/>
	<key>Rating</key>
	<real>4.5</real>
	<key>Tags</key>
	<array>
		<string>a</string>
		<string>b</string>
	</array>
</dict>
</plist>`

func TestPlistDecode_XMLBasic(t *testing.T) {
	result, err := runPlistDecode(t, testPlistXML)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsNull() {
		t.Fatal("expected non-null result")
	}

	// Check it's an object with expected keys.
	obj, ok := result.UnderlyingValue().(types.Object)
	if !ok {
		t.Fatalf("expected Object, got %T", result.UnderlyingValue())
	}
	attrs := obj.Attributes()

	// Check Name.
	name, ok := attrs["Name"].(types.String)
	if !ok {
		t.Fatalf("expected Name to be String, got %T", attrs["Name"])
	}
	if name.ValueString() != "Test Profile" {
		t.Errorf("expected Name='Test Profile', got %q", name.ValueString())
	}

	// Check Version (integer).
	version, ok := attrs["Version"].(types.Number)
	if !ok {
		t.Fatalf("expected Version to be Number, got %T", attrs["Version"])
	}
	v, _ := version.ValueBigFloat().Float64()
	if v != 1 {
		t.Errorf("expected Version=1, got %f", v)
	}

	// Check Enabled.
	enabled, ok := attrs["Enabled"].(types.Bool)
	if !ok {
		t.Fatalf("expected Enabled to be Bool, got %T", attrs["Enabled"])
	}
	if !enabled.ValueBool() {
		t.Error("expected Enabled=true")
	}

	// Check Rating (real/float).
	rating, ok := attrs["Rating"].(types.Number)
	if !ok {
		t.Fatalf("expected Rating to be Number, got %T", attrs["Rating"])
	}
	r, _ := rating.ValueBigFloat().Float64()
	if r != 4.5 {
		t.Errorf("expected Rating=4.5, got %f", r)
	}

	// Check Tags (array).
	tags, ok := attrs["Tags"].(types.Tuple)
	if !ok {
		t.Fatalf("expected Tags to be Tuple, got %T", attrs["Tags"])
	}
	if len(tags.Elements()) != 2 {
		t.Errorf("expected 2 tags, got %d", len(tags.Elements()))
	}
}

const testPlistWithData = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Certificate</key>
	<data>SGVsbG8gV29ybGQ=</data>
</dict>
</plist>`

func TestPlistDecode_DataElement(t *testing.T) {
	result, err := runPlistDecode(t, testPlistWithData)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	obj := result.UnderlyingValue().(types.Object)
	cert := obj.Attributes()["Certificate"].(types.Object)
	attrs := cert.Attributes()

	typeVal := attrs[plistTypeKey].(types.String).ValueString()
	if typeVal != plistTypeData {
		t.Errorf("expected __plist_type=%q, got %q", plistTypeData, typeVal)
	}

	valueVal := attrs[plistValueKey].(types.String).ValueString()
	if valueVal != "SGVsbG8gV29ybGQ=" {
		t.Errorf("expected base64 value 'SGVsbG8gV29ybGQ=', got %q", valueVal)
	}
}

const testPlistWithDate = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>ExpirationDate</key>
	<date>2025-06-01T00:00:00Z</date>
</dict>
</plist>`

func TestPlistDecode_DateElement(t *testing.T) {
	result, err := runPlistDecode(t, testPlistWithDate)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	obj := result.UnderlyingValue().(types.Object)
	date := obj.Attributes()["ExpirationDate"].(types.Object)
	attrs := date.Attributes()

	typeVal := attrs[plistTypeKey].(types.String).ValueString()
	if typeVal != plistTypeDate {
		t.Errorf("expected __plist_type=%q, got %q", plistTypeDate, typeVal)
	}

	valueVal := attrs[plistValueKey].(types.String).ValueString()
	if valueVal != "2025-06-01T00:00:00Z" {
		t.Errorf("expected date '2025-06-01T00:00:00Z', got %q", valueVal)
	}
}

func TestPlistDecode_BinaryViaBase64(t *testing.T) {
	// Binary plist containing: {Enabled: true, Name: "Binary Test", Rating: 3.14, Version: 2}
	// Generated with howett.net/plist BinaryFormat, then base64-encoded.
	b64 := "YnBsaXN0MDDUAQIDBAUGBwhXRW5hYmxlZFROYW1lVlJhdGluZ1dWZXJzaW9uCVtCaW5hcnkgVGVzdCNACR64UeuFHxACCBEZHiUtLjpDAAAAAAAAAQEAAAAAAAAACQAAAAAAAAAAAAAAAAAAAEU="

	result, err := runPlistDecode(t, b64)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	obj, ok := result.UnderlyingValue().(types.Object)
	if !ok {
		t.Fatalf("expected Object, got %T", result.UnderlyingValue())
	}
	attrs := obj.Attributes()

	name, ok := attrs["Name"].(types.String)
	if !ok {
		t.Fatalf("expected Name to be String, got %T", attrs["Name"])
	}
	if name.ValueString() != "Binary Test" {
		t.Errorf("expected Name='Binary Test', got %q", name.ValueString())
	}

	enabled, ok := attrs["Enabled"].(types.Bool)
	if !ok {
		t.Fatalf("expected Enabled to be Bool, got %T", attrs["Enabled"])
	}
	if !enabled.ValueBool() {
		t.Error("expected Enabled=true")
	}

	version, ok := attrs["Version"].(types.Number)
	if !ok {
		t.Fatalf("expected Version to be Number, got %T", attrs["Version"])
	}
	v, _ := version.ValueBigFloat().Float64()
	if v != 2 {
		t.Errorf("expected Version=2, got %f", v)
	}

	rating, ok := attrs["Rating"].(types.Number)
	if !ok {
		t.Fatalf("expected Rating to be Number, got %T", attrs["Rating"])
	}
	r, _ := rating.ValueBigFloat().Float64()
	if r < 3.13 || r > 3.15 {
		t.Errorf("expected Rating~=3.14, got %f", r)
	}
}

const testPlistOpenStep = `{Enabled=<*BY>;Name="OpenStep Test";Rating=<*R2.5>;Version=<*I7>;}`

func TestPlistDecode_OpenStep(t *testing.T) {
	result, err := runPlistDecode(t, testPlistOpenStep)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	obj, ok := result.UnderlyingValue().(types.Object)
	if !ok {
		t.Fatalf("expected Object, got %T", result.UnderlyingValue())
	}
	attrs := obj.Attributes()

	name, ok := attrs["Name"].(types.String)
	if !ok {
		t.Fatalf("expected Name to be String, got %T", attrs["Name"])
	}
	if name.ValueString() != "OpenStep Test" {
		t.Errorf("expected Name='OpenStep Test', got %q", name.ValueString())
	}

	version, ok := attrs["Version"].(types.Number)
	if !ok {
		t.Fatalf("expected Version to be Number, got %T", attrs["Version"])
	}
	v, _ := version.ValueBigFloat().Float64()
	if v != 7 {
		t.Errorf("expected Version=7, got %f", v)
	}

	rating, ok := attrs["Rating"].(types.Number)
	if !ok {
		t.Fatalf("expected Rating to be Number, got %T", attrs["Rating"])
	}
	r, _ := rating.ValueBigFloat().Float64()
	if r != 2.5 {
		t.Errorf("expected Rating=2.5, got %f", r)
	}
}

func TestPlistDecode_IntegerVsReal(t *testing.T) {
	input := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>IntVal</key>
	<integer>2</integer>
	<key>RealWholeVal</key>
	<real>2</real>
	<key>RealFracVal</key>
	<real>3.14</real>
</dict>
</plist>`

	result, err := runPlistDecode(t, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	obj := result.UnderlyingValue().(types.Object)
	attrs := obj.Attributes()

	// <integer>2</integer> → plain number
	intVal, ok := attrs["IntVal"].(types.Number)
	if !ok {
		t.Fatalf("expected IntVal to be Number, got %T", attrs["IntVal"])
	}
	iv, _ := intVal.ValueBigFloat().Float64()
	if iv != 2 {
		t.Errorf("expected IntVal=2, got %f", iv)
	}

	// <real>2</real> → tagged real object (whole-number float)
	realWholeVal, ok := attrs["RealWholeVal"].(types.Object)
	if !ok {
		t.Fatalf("expected RealWholeVal to be Object (tagged real), got %T", attrs["RealWholeVal"])
	}
	rwAttrs := realWholeVal.Attributes()
	typeVal := rwAttrs[plistTypeKey].(types.String).ValueString()
	if typeVal != plistTypeReal {
		t.Errorf("expected __plist_type=%q, got %q", plistTypeReal, typeVal)
	}

	// <real>3.14</real> → plain number (fractional, unambiguous)
	realFracVal, ok := attrs["RealFracVal"].(types.Number)
	if !ok {
		t.Fatalf("expected RealFracVal to be Number, got %T", attrs["RealFracVal"])
	}
	rv, _ := realFracVal.ValueBigFloat().Float64()
	if rv != 3.14 {
		t.Errorf("expected RealFracVal=3.14, got %f", rv)
	}
}

func TestPlistDecode_IntegerRealRoundTrip(t *testing.T) {
	input := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>IntVal</key>
	<integer>5</integer>
	<key>RealVal</key>
	<real>5</real>
</dict>
</plist>`

	decoded, decErr := runPlistDecode(t, input)
	if decErr != nil {
		t.Fatalf("decode error: %v", decErr)
	}

	encoded, encErr := runPlistEncode(t, decoded.UnderlyingValue())
	if encErr != nil {
		t.Fatalf("encode error: %v", encErr)
	}

	if !strings.Contains(encoded, "<integer>5</integer>") {
		t.Errorf("expected <integer>5</integer> in round-trip output:\n%s", encoded)
	}
	if !strings.Contains(encoded, "<real>5</real>") {
		t.Errorf("expected <real>5</real> in round-trip output:\n%s", encoded)
	}
}

func TestPlistDecode_DoctypePlist(t *testing.T) {
	// Test that input starting with <!DOCTYPE plist is recognized.
	input := `<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>Key</key>
	<string>Value</string>
</dict>
</plist>`
	_, err := runPlistDecode(t, input)
	if err != nil {
		t.Fatalf("unexpected error for DOCTYPE-prefixed plist: %v", err)
	}
}

func TestPlistDecode_InvalidInput(t *testing.T) {
	_, err := runPlistDecode(t, `not a plist at all!!!`)
	if err == nil {
		t.Fatal("expected error for invalid input")
	}
}

func TestPlistDecode_EmptyDict(t *testing.T) {
	input := `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict/>
</plist>`
	result, err := runPlistDecode(t, input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsNull() {
		t.Fatal("expected non-null result")
	}
}

func TestPlistDecode_EmptyString(t *testing.T) {
	// Empty string base64-decodes to empty bytes, which the plist library
	// treats as an empty dict. This is valid behavior.
	result, err := runPlistDecode(t, "")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.IsNull() {
		t.Fatal("expected non-null result")
	}
}
