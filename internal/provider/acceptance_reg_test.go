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

// ─── regdword / regqword range checks ──────────────────────────────

func TestAcc_RegDword_AcceptsUpperBound(t *testing.T) {
	// 2^32 - 1 = 4294967295 — the documented maximum.
	runOutputTest(t,
		`output "test" { value = provider::burnham::regdword(4294967295) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.NotNull()),
	)
}

func TestAcc_RegDword_RejectsNegative(t *testing.T) {
	// Regression: previously `Uint64()` silently coerced negatives to 0.
	runErrorTest(t,
		`output "test" { value = provider::burnham::regdword(-1) }`,
		regexp.MustCompile(`(?is)value\s+must\s+be\s+>=\s+0`),
	)
}

func TestAcc_RegDword_RejectsAboveMax(t *testing.T) {
	// Regression: previously values above 2^32-1 silently saturated to MaxUint32.
	runErrorTest(t,
		`output "test" { value = provider::burnham::regdword(4294967296) }`,
		regexp.MustCompile(`(?is)value\s+must\s+be\s+in\s+\[0,\s*4294967295\]`),
	)
}

func TestAcc_RegDword_RejectsFractional(t *testing.T) {
	runErrorTest(t,
		`output "test" { value = provider::burnham::regdword(1.5) }`,
		regexp.MustCompile(`(?is)value\s+must\s+be\s+a\s+whole\s+number`),
	)
}

func TestAcc_RegQword_AcceptsUpperBound(t *testing.T) {
	// 2^64 - 1 = 18446744073709551615 — Terraform's number type carries the value precisely.
	runOutputTest(t,
		`output "test" { value = provider::burnham::regqword(18446744073709551615) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.NotNull()),
	)
}

func TestAcc_RegQword_RejectsNegative(t *testing.T) {
	runErrorTest(t,
		`output "test" { value = provider::burnham::regqword(-1) }`,
		regexp.MustCompile(`(?is)value\s+must\s+be\s+>=\s+0`),
	)
}

func TestAcc_RegQword_RejectsAboveMax(t *testing.T) {
	// 2^64 = 18446744073709551616, one above the max.
	runErrorTest(t,
		`output "test" { value = provider::burnham::regqword(18446744073709551616) }`,
		regexp.MustCompile(`(?is)value\s+must\s+be\s+in\s+\[0,\s*18446744073709551615\]`),
	)
}

// ─── regencode injection / regbinary / regmulti edge cases ─────────

func TestAcc_RegEncode_RejectsBracketInPath(t *testing.T) {
	// A `]` in the registry path would close the bracket-line in the .reg output and let an attacker append arbitrary registry directives. Reject explicitly.
	runErrorTest(t,
		`output "test" {
		   value = provider::burnham::regencode({
		     "HKEY_LOCAL_MACHINE\\Bad]Path" = { "Name" = "x" }
		   })
		 }`,
		regexp.MustCompile(`(?is)forbidden\s+character`),
	)
}

func TestAcc_RegEncode_RejectsNewlineInPath(t *testing.T) {
	runErrorTest(t,
		`output "test" {
		   value = provider::burnham::regencode({
		     "HKEY_LOCAL_MACHINE\nInjected" = { "Name" = "x" }
		   })
		 }`,
		regexp.MustCompile(`(?is)forbidden\s+character`),
	)
}

func TestAcc_RegBinary_RejectsEmptyHex(t *testing.T) {
	runErrorTest(t,
		`output "test" { value = provider::burnham::regbinary("") }`,
		regexp.MustCompile(`(?is)hex\s+must\s+not\s+be\s+empty`),
	)
}

func TestAcc_RegMulti_RejectsEmptyList(t *testing.T) {
	runErrorTest(t,
		`output "test" { value = provider::burnham::regmulti([]) }`,
		regexp.MustCompile(`(?is)at\s+least\s+one\s+entry`),
	)
}
