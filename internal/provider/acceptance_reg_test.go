package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
)

func TestAcc_RegDecode_Basic(t *testing.T) {
	runOutputTest(t,
		`
				locals {
					reg = provider::burnham::regdecode("Windows Registry Editor Version 5.00\r\n\r\n[HKEY_LOCAL_MACHINE\\SOFTWARE\\Test]\r\n\"Name\"=\"Hello\"\r\n")
				}
				output "name" { value = local.reg["HKEY_LOCAL_MACHINE\\SOFTWARE\\Test"].Name }
			`,
		statecheck.ExpectKnownOutputValue("name", knownvalue.StringExact("Hello")),
	)
}

func TestAcc_RegEncode_Basic(t *testing.T) {
	runOutputTest(t,
		`
				output "test" {
					value = provider::burnham::regencode({
						"HKEY_LOCAL_MACHINE\\SOFTWARE\\Test" = {
							"Name" = "Hello"
							"Count" = provider::burnham::regdword(42)
						}
					})
				}
			`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.NotNull()),
	)
}
