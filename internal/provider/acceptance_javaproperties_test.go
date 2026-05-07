package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
)

func TestAcc_JavaPropertiesDecode_Basic(t *testing.T) {
	runOutputTest(t,
		`output "test" {
			value = provider::burnham::javapropertiesdecode("# comment\nfoo=bar\nbaz : qux\n")
		}`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.ObjectExact(map[string]knownvalue.Check{
			"foo": knownvalue.StringExact("bar"),
			"baz": knownvalue.StringExact("qux"),
		})),
	)
}

func TestAcc_JavaPropertiesDecode_UnicodeEscape(t *testing.T) {
	runOutputTest(t,
		`output "test" {
			value = provider::burnham::javapropertiesdecode("greeting=hell\u00f6")
		}`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.ObjectExact(map[string]knownvalue.Check{
			"greeting": knownvalue.StringExact("hellö"),
		})),
	)
}

func TestAcc_JavaPropertiesEncode_Basic(t *testing.T) {
	runOutputTest(t,
		`output "test" {
			value = provider::burnham::javapropertiesencode({ "app.name" = "frontend", "app.replicas" = 3 })
		}`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("app.name=frontend\napp.replicas=3\n")),
	)
}

func TestAcc_JavaPropertiesEncode_EscapesSpecials(t *testing.T) {
	runOutputTest(t,
		`output "test" {
			value = provider::burnham::javapropertiesencode({ "key=with=equals" = "value:with:colons" })
		}`,
		// '=' and ':' in the key get escaped (\=, \:); ':' in the value is left alone (only key separators need escaping).
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("key\\=with\\=equals=value:with:colons\n")),
	)
}
