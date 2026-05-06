package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
)

func TestAcc_KDLDecode_Basic(t *testing.T) {
	runOutputTest(t,
		`
				locals {
					kdl = provider::burnham::kdldecode("title \"Hello\"")
				}
				output "name" { value = local.kdl[0].name }
			`,
		statecheck.ExpectKnownOutputValue("name", knownvalue.StringExact("title")),
	)
}

func TestAcc_KDLEncode_Basic(t *testing.T) {
	runOutputTest(t,
		`
				output "test" {
					value = provider::burnham::kdlencode([
						{ name = "title", args = ["Hello"], props = {}, children = [] }
					])
				}
			`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.NotNull()),
	)
}

// ─── Malformed-input tests ───────────────────────────────────────

func TestAcc_KDLDecode_MalformedInput(t *testing.T) {
	// User passes a string with KDL-syntax errors — should be a parse error.
	runErrorTest(t,
		`output "test" { value = provider::burnham::kdldecode("node { unclosed") }`,
		regexp.MustCompile(`(?i)kdl|parse|unexpected`),
	)
}
