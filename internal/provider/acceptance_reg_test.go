package provider

import (
	"regexp"
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

// ─── Malformed-input tests ───────────────────────────────────────

func TestAcc_RegDecode_MissingVersionHeader(t *testing.T) {
	// A real .reg file always opens with `Windows Registry Editor Version
	// 5.00` (or REGEDIT4). Files saved from a different tool or pasted
	// without that header should produce a clear error.
	runErrorTest(t,
		`output "test" { value = provider::burnham::regdecode("[HKEY_LOCAL_MACHINE\\SOFTWARE\\Test]\r\n\"Name\"=\"Hello\"\r\n") }`,
		regexp.MustCompile(`(?i)reg|version|invalid|header`),
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
