package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
)

// runOutputTest applies a single Terraform config under the burnham provider
// and asserts the given state checks (typically ExpectKnownOutputValue) hold
// after apply. Captures the boilerplate every output-asserting acceptance
// test would otherwise repeat.
func runOutputTest(t *testing.T, config string, checks ...statecheck.StateCheck) {
	t.Helper()
	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks:   testAccTerraformVersionChecks,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config:            config,
			ConfigStateChecks: checks,
		}},
	})
}

// runErrorTest applies a Terraform config and asserts the apply fails with an
// error matching the given pattern. Use for tests of malformed input or
// validation rules.
func runErrorTest(t *testing.T, config string, errPattern *regexp.Regexp) {
	t.Helper()
	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks:   testAccTerraformVersionChecks,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config:      config,
			ExpectError: errPattern,
		}},
	})
}
