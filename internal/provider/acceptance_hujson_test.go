package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
)

func TestAcc_HuJSONDecode_WithComments(t *testing.T) {
	runOutputTest(t,
		`
				locals {
					decoded = provider::burnham::hujsondecode("{// comment\n\"key\": \"value\",}")
				}
				output "test" { value = local.decoded.key }
			`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("value")),
	)
}

func TestAcc_HuJSONDecode_Invalid(t *testing.T) {
	runErrorTest(t,
		`output "test" { value = provider::burnham::hujsondecode("{bad}") }`,
		regexp.MustCompile(`Invalid HuJSON`),
	)
}

// ─── hujsonencode ────────────────────────────────────────────────

func TestAcc_HuJSONEncode_Basic(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::hujsonencode({ key = "value" }) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.NotNull()),
	)
}

func TestAcc_HuJSONEncode_WithComments(t *testing.T) {
	runOutputTest(t,
		`
				output "test" {
					value = provider::burnham::hujsonencode(
						{
							acls   = ["accept"]
							groups = { admin = ["alice@example.com"] }
						},
						{
							comments = {
								acls   = "Network ACLs"
								groups = "Group definitions"
							}
						}
					)
				}
			`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.NotNull()),
	)
}

func TestAcc_HuJSONRoundTrip(t *testing.T) {
	runOutputTest(t,
		`
				locals {
					input   = "{// comment\n\"key\": \"value\",\n\"num\": 42,}"
					decoded = provider::burnham::hujsondecode(local.input)
					encoded = provider::burnham::hujsonencode(local.decoded)
				}
				output "key" { value = local.decoded.key }
			`,
		statecheck.ExpectKnownOutputValue("key", knownvalue.StringExact("value")),
	)
}
