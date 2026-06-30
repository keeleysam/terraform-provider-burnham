package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
)

func TestAcc_JSONEncode_DefaultTabs(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::jsonencode({ name = "hello", count = 2 }) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("{\n\t\"count\": 2,\n\t\"name\": \"hello\"\n}")),
	)
}

func TestAcc_JSONEncode_CustomIndent(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::jsonencode({ a = 1 }, { indent = "  " }) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("{\n  \"a\": 1\n}")),
	)
}

func TestAcc_JSONEncode_DoesNotEscapeHTMLByDefault(t *testing.T) {
	// A pretty-printer for human review should emit <, > and & literally, not
	// as < / > / &.
	runOutputTest(t,
		`output "test" { value = provider::burnham::jsonencode({ q = "1 < 2 > 0 & ok" }) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("{\n\t\"q\": \"1 < 2 > 0 & ok\"\n}")),
	)
}

func TestAcc_JSONEncode_EscapeHTMLOptIn(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::jsonencode({ q = "a > b" }, { escape_html = true }) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("{\n\t\"q\": \"a \\u003e b\"\n}")),
	)
}
