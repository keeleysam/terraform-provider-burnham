package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
)

func TestAcc_VDFDecode_Basic(t *testing.T) {
	runOutputTest(t,
		`
				locals {
					vdf = provider::burnham::vdfdecode("\"Config\"\n{\n\t\"key\"\t\t\"value\"\n}\n")
				}
				output "val" { value = local.vdf.Config.key }
			`,
		statecheck.ExpectKnownOutputValue("val", knownvalue.StringExact("value")),
	)
}

func TestAcc_VDFEncode_Basic(t *testing.T) {
	runOutputTest(t,
		`
				output "test" {
					value = provider::burnham::vdfencode({
						AppState = { appid = "730", name = "CS2" }
					})
				}
			`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.NotNull()),
	)
}
