package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
)

// --- base64zopfli ---

func TestAcc_Base64Zopfli_DefaultEqualsExplicit(t *testing.T) {
	// The documented default is iterations = 15; passing it explicitly must produce identical output, which also pins determinism through the full Run path.
	runOutputTest(t,
		`output "test" {
			value = (
				provider::burnham::base64zopfli("aim high in hope and work") ==
				provider::burnham::base64zopfli("aim high in hope and work", { iterations = 15 })
			)
		}`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.Bool(true)),
	)
}

func TestAcc_Base64Zopfli_EmptyInput(t *testing.T) {
	// Empty input must produce a valid (non-empty, base64) gzip member, not an error.
	runOutputTest(t,
		`output "test" { value = provider::burnham::base64zopfli("") }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringRegexp(regexp.MustCompile(`^[A-Za-z0-9+/]+=*$`))),
	)
}

func TestAcc_Base64Zopfli_IterationsTooLow(t *testing.T) {
	runErrorTest(t,
		`output "test" { value = provider::burnham::base64zopfli("x", { iterations = 0 }) }`,
		regexp.MustCompile(`(?i)iterations.*\[1, 100000\]`),
	)
}

func TestAcc_Base64Zopfli_IterationsTooHigh(t *testing.T) {
	runErrorTest(t,
		`output "test" { value = provider::burnham::base64zopfli("x", { iterations = 100001 }) }`,
		regexp.MustCompile(`(?i)iterations.*\[1, 100000\]`),
	)
}

func TestAcc_Base64Zopfli_UnknownOptionKey(t *testing.T) {
	runErrorTest(t,
		`output "test" { value = provider::burnham::base64zopfli("x", { quality = 5 }) }`,
		regexp.MustCompile(`(?is)unknown option key.*iterations`),
	)
}

func TestAcc_Base64Zopfli_NonObjectOptions(t *testing.T) {
	runErrorTest(t,
		`output "test" { value = provider::burnham::base64zopfli("x", "nope") }`,
		regexp.MustCompile(`(?i)object literal`),
	)
}

// --- base64brotli ---

func TestAcc_Base64Brotli_DefaultEqualsExplicit(t *testing.T) {
	// Documented defaults are quality = 11, lgwin = 22.
	runOutputTest(t,
		`output "test" {
			value = (
				provider::burnham::base64brotli("aim high in hope and work") ==
				provider::burnham::base64brotli("aim high in hope and work", { quality = 11, lgwin = 22 })
			)
		}`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.Bool(true)),
	)
}

func TestAcc_Base64Brotli_EmptyInput(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::base64brotli("") }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringRegexp(regexp.MustCompile(`^[A-Za-z0-9+/]+=*$`))),
	)
}

func TestAcc_Base64Brotli_QualityOutOfRange(t *testing.T) {
	runErrorTest(t,
		`output "test" { value = provider::burnham::base64brotli("x", { quality = 12 }) }`,
		regexp.MustCompile(`(?i)quality.*\[0, 11\]`),
	)
}

func TestAcc_Base64Brotli_LgwinOutOfRange(t *testing.T) {
	runErrorTest(t,
		`output "test" { value = provider::burnham::base64brotli("x", { lgwin = 9 }) }`,
		regexp.MustCompile(`(?i)lgwin.*\[10, 24\]`),
	)
}

func TestAcc_Base64Brotli_UnknownOptionKey(t *testing.T) {
	// `mode` is deliberately not exposed; passing it should be a clear unknown-key error.
	runErrorTest(t,
		`output "test" { value = provider::burnham::base64brotli("x", { mode = "text" }) }`,
		regexp.MustCompile(`(?is)unknown option key.*quality, lgwin`),
	)
}
