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
//
// Calls `t.Parallel()` so every acceptance test that goes through this helper
// (in practice: all of them) runs concurrently with its siblings. Safe because
// the provider is pure functions — no shared package-level state, no remote
// calls, no resources — and terraform-plugin-testing gives each test its own
// working directory, provider instance (the factory is invoked per-test), and
// terraform CLI subprocess. Concurrency is bounded by `go test -parallel N`
// (default = GOMAXPROCS), so the cap is the runner's core count.
func runOutputTest(t *testing.T, config string, checks ...statecheck.StateCheck) {
	t.Helper()
	t.Parallel()
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
// validation rules. Parallel for the same reasons as runOutputTest.
func runErrorTest(t *testing.T, config string, errPattern *regexp.Regexp) {
	t.Helper()
	t.Parallel()
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
