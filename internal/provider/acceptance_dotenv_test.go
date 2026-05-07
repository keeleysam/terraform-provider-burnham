package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
)

func TestAcc_DotenvDecode_Basic(t *testing.T) {
	runOutputTest(t,
		`output "test" {
			value = provider::burnham::dotenvdecode("# comment\nFOO=bar\nBAZ=qux\n")
		}`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.ObjectExact(map[string]knownvalue.Check{
			"FOO": knownvalue.StringExact("bar"),
			"BAZ": knownvalue.StringExact("qux"),
		})),
	)
}

func TestAcc_DotenvDecode_Quoted(t *testing.T) {
	runOutputTest(t,
		`output "test" {
			value = provider::burnham::dotenvdecode("MSG=\"hello world\"\nLITERAL='no $interp here'\n")
		}`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.ObjectExact(map[string]knownvalue.Check{
			"MSG":     knownvalue.StringExact("hello world"),
			"LITERAL": knownvalue.StringExact("no $interp here"),
		})),
	)
}

func TestAcc_DotenvEncode_QuotesValuesNeedingIt(t *testing.T) {
	runOutputTest(t,
		`output "test" {
			value = provider::burnham::dotenvencode({
				PLAIN = "foo"
				WITH_SPACE = "hello world"
			})
		}`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("PLAIN=foo\nWITH_SPACE=\"hello world\"\n")),
	)
}

func TestAcc_DotenvRoundtrip_PreservesValues(t *testing.T) {
	runOutputTest(t,
		`output "test" {
			value = provider::burnham::dotenvdecode(provider::burnham::dotenvencode({
				A = "simple"
				B = "with \"quotes\" and\nnewline"
			}))
		}`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.ObjectExact(map[string]knownvalue.Check{
			"A": knownvalue.StringExact("simple"),
			"B": knownvalue.StringExact("with \"quotes\" and\nnewline"),
		})),
	)
}
