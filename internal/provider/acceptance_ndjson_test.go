package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
)

func TestAcc_NDJSONDecode_Basic(t *testing.T) {
	runOutputTest(t,
		"output \"test\" { value = provider::burnham::ndjsondecode(\"{\\\"a\\\":1}\\n{\\\"a\\\":2}\\n\") }",
		statecheck.ExpectKnownOutputValue("test", knownvalue.TupleExact([]knownvalue.Check{
			knownvalue.ObjectExact(map[string]knownvalue.Check{"a": knownvalue.Int64Exact(1)}),
			knownvalue.ObjectExact(map[string]knownvalue.Check{"a": knownvalue.Int64Exact(2)}),
		})),
	)
}

func TestAcc_NDJSONDecode_Empty(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::ndjsondecode("") }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.TupleExact([]knownvalue.Check{})),
	)
}

func TestAcc_NDJSONEncode_Basic(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::ndjsonencode([{ a = 1 }, { a = 2 }]) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("{\"a\":1}\n{\"a\":2}\n")),
	)
}

func TestAcc_NDJSONDecode_Malformed(t *testing.T) {
	runErrorTest(t,
		`output "test" { value = provider::burnham::ndjsondecode("{not json}") }`,
		regexp.MustCompile(`(?i)ndjson|decode|json`),
	)
}
