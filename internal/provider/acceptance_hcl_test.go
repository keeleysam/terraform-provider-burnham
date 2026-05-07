package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
)

func TestAcc_HCLDecode_Primitives(t *testing.T) {
	runOutputTest(t,
		`output "test" {
			value = provider::burnham::hcldecode("name = \"web\"\nreplicas = 3\nenabled = true")
		}`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.ObjectExact(map[string]knownvalue.Check{
			"name":     knownvalue.StringExact("web"),
			"replicas": knownvalue.Int64Exact(3),
			"enabled":  knownvalue.Bool(true),
		})),
	)
}

func TestAcc_HCLDecode_NestedObject(t *testing.T) {
	runOutputTest(t,
		`output "test" {
			value = provider::burnham::hcldecode("config = { name = \"web\", port = 8080 }")
		}`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.ObjectExact(map[string]knownvalue.Check{
			"config": knownvalue.ObjectExact(map[string]knownvalue.Check{
				"name": knownvalue.StringExact("web"),
				"port": knownvalue.Int64Exact(8080),
			}),
		})),
	)
}

func TestAcc_HCLDecode_List(t *testing.T) {
	runOutputTest(t,
		`output "test" {
			value = provider::burnham::hcldecode("tags = [\"a\", \"b\", \"c\"]")
		}`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.ObjectExact(map[string]knownvalue.Check{
			"tags": knownvalue.TupleExact([]knownvalue.Check{
				knownvalue.StringExact("a"),
				knownvalue.StringExact("b"),
				knownvalue.StringExact("c"),
			}),
		})),
	)
}

func TestAcc_HCLEncode_Basic(t *testing.T) {
	runOutputTest(t,
		`output "test" {
			value = provider::burnham::hclencode({ name = "web", replicas = 3 })
		}`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("name     = \"web\"\nreplicas = 3\n")),
	)
}

func TestAcc_HCLDecode_Malformed(t *testing.T) {
	runErrorTest(t,
		`output "test" { value = provider::burnham::hcldecode("foo = ") }`,
		regexp.MustCompile(`(?i)hcl|parse|expected`),
	)
}

func TestAcc_HCLDecode_RejectsBlocks(t *testing.T) {
	// Silent block-dropping was a real bug pre-review; this test pins the new error behavior.
	runErrorTest(t,
		`output "test" { value = provider::burnham::hcldecode("name = \"x\"\nprovisioner \"local\" { y = 1 }") }`,
		regexp.MustCompile(`(?i)block`),
	)
}

func TestAcc_HCLEncode_NestedAndList(t *testing.T) {
	runOutputTest(t,
		`output "test" {
			value = provider::burnham::hclencode({
				server = { host = "localhost", port = 8080 }
				tags   = ["prod", "east"]
			})
		}`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact(
			"server = {\n  host = \"localhost\"\n  port = 8080\n}\ntags = [\"prod\", \"east\"]\n",
		)),
	)
}
