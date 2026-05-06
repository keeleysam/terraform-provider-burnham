package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
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

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"burnham": providerserver.NewProtocol6WithError(New()),
}

var testAccTerraformVersionChecks = []tfversion.TerraformVersionCheck{
	tfversion.SkipBelow(tfversion.Version1_8_0),
}
