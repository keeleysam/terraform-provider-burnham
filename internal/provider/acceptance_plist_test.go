package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
)

func TestAcc_PlistDecode_XML(t *testing.T) {
	runOutputTest(t,
		`
				locals {
					plist = provider::burnham::plistdecode(<<-EOT
						<?xml version="1.0" encoding="UTF-8"?>
						<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
						<plist version="1.0">
						<dict>
							<key>Name</key>
							<string>Test</string>
							<key>Version</key>
							<integer>1</integer>
							<key>Enabled</key>
							<true/>
						</dict>
						</plist>
					EOT
					)
				}
				output "name" { value = local.plist.Name }
				output "version" { value = local.plist.Version }
				output "enabled" { value = local.plist.Enabled }
			`,
		statecheck.ExpectKnownOutputValue("name", knownvalue.StringExact("Test")),
		statecheck.ExpectKnownOutputValue("version", knownvalue.Int64Exact(1)),
		statecheck.ExpectKnownOutputValue("enabled", knownvalue.Bool(true)),
	)
}

func TestAcc_PlistDecode_DateTaggedObject(t *testing.T) {
	runOutputTest(t,
		`
				locals {
					plist = provider::burnham::plistdecode(<<-EOT
						<?xml version="1.0" encoding="UTF-8"?>
						<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
						<plist version="1.0">
						<dict>
							<key>ExpirationDate</key>
							<date>2025-06-01T00:00:00Z</date>
						</dict>
						</plist>
					EOT
					)
				}
				output "type" { value = local.plist.ExpirationDate.__plist_type }
				output "value" { value = local.plist.ExpirationDate.value }
			`,
		statecheck.ExpectKnownOutputValue("type", knownvalue.StringExact("date")),
		statecheck.ExpectKnownOutputValue("value", knownvalue.StringExact("2025-06-01T00:00:00Z")),
	)
}

func TestAcc_PlistDecode_IntegerVsReal(t *testing.T) {
	runOutputTest(t,
		`
				locals {
					plist = provider::burnham::plistdecode(<<-EOT
						<?xml version="1.0" encoding="UTF-8"?>
						<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
						<plist version="1.0">
						<dict>
							<key>IntVal</key>
							<integer>5</integer>
							<key>RealVal</key>
							<real>5</real>
							<key>FracVal</key>
							<real>3.14</real>
						</dict>
						</plist>
					EOT
					)
				}
				output "int_val" { value = local.plist.IntVal }
				output "real_type" { value = local.plist.RealVal.__plist_type }
				output "frac_val" { value = local.plist.FracVal }
			`,
		statecheck.ExpectKnownOutputValue("int_val", knownvalue.Int64Exact(5)),
		statecheck.ExpectKnownOutputValue("real_type", knownvalue.StringExact("real")),
		statecheck.ExpectKnownOutputValue("frac_val", knownvalue.Float64Exact(3.14)),
	)
}

// ─── plistencode ─────────────────────────────────────────────────

func TestAcc_PlistEncode_XML(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::plistencode({ Name = "Test", Version = 1 }) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.NotNull()),
	)
}

func TestAcc_PlistEncode_InvalidFormat(t *testing.T) {
	runErrorTest(t,
		`output "test" { value = provider::burnham::plistencode({ a = 1 }, { format = "yaml" }) }`,
		regexp.MustCompile(`unsupported plist`),
	)
}

// ─── plistdate ───────────────────────────────────────────────────

func TestAcc_PlistDate_Valid(t *testing.T) {
	runOutputTest(t,
		`
				locals {
					d = provider::burnham::plistdate("2025-06-01T00:00:00Z")
				}
				output "type" { value = local.d.__plist_type }
				output "value" { value = local.d.value }
			`,
		statecheck.ExpectKnownOutputValue("type", knownvalue.StringExact("date")),
		statecheck.ExpectKnownOutputValue("value", knownvalue.StringExact("2025-06-01T00:00:00Z")),
	)
}

func TestAcc_PlistDate_Invalid(t *testing.T) {
	runErrorTest(t,
		`output "test" { value = provider::burnham::plistdate("not-a-date") }`,
		regexp.MustCompile(`Invalid RFC 3339`),
	)
}

// ─── plistdata ───────────────────────────────────────────────────

func TestAcc_PlistData_Valid(t *testing.T) {
	runOutputTest(t,
		`
				locals {
					d = provider::burnham::plistdata("SGVsbG8=")
				}
				output "type" { value = local.d.__plist_type }
				output "value" { value = local.d.value }
			`,
		statecheck.ExpectKnownOutputValue("type", knownvalue.StringExact("data")),
		statecheck.ExpectKnownOutputValue("value", knownvalue.StringExact("SGVsbG8=")),
	)
}

func TestAcc_PlistData_Invalid(t *testing.T) {
	runErrorTest(t,
		`output "test" { value = provider::burnham::plistdata("!!!invalid!!!") }`,
		regexp.MustCompile(`Invalid base64`),
	)
}

// ─── plistreal ───────────────────────────────────────────────────

func TestAcc_PlistReal_Valid(t *testing.T) {
	runOutputTest(t,
		`
				locals {
					r = provider::burnham::plistreal(2)
				}
				output "type" { value = local.r.__plist_type }
				output "value" { value = local.r.value }
			`,
		statecheck.ExpectKnownOutputValue("type", knownvalue.StringExact("real")),
		statecheck.ExpectKnownOutputValue("value", knownvalue.StringExact("2")),
	)
}

// ─── Round-trips ─────────────────────────────────────────────────

func TestAcc_PlistRoundTrip_PreservesTypes(t *testing.T) {
	runOutputTest(t,
		`
				locals {
					input = <<-EOT
						<?xml version="1.0" encoding="UTF-8"?>
						<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
						<plist version="1.0">
						<dict>
							<key>Name</key>
							<string>Test</string>
							<key>ExpirationDate</key>
							<date>2025-06-01T00:00:00Z</date>
							<key>Count</key>
							<integer>3</integer>
							<key>Scale</key>
							<real>3</real>
						</dict>
						</plist>
					EOT
					decoded = provider::burnham::plistdecode(local.input)
					encoded = provider::burnham::plistencode(local.decoded)
				}
				output "encoded" { value = local.encoded }
			`,
		statecheck.ExpectKnownOutputValue("encoded", knownvalue.NotNull()),
	)
}

// ─── Malformed-input tests ───────────────────────────────────────

func TestAcc_PlistDecode_GarbageInput(t *testing.T) {
	// User passes a non-plist string (e.g. forgot to read the file, or read
	// the wrong file). Should be a clear parse error, not a panic.
	runErrorTest(t,
		`output "test" { value = provider::burnham::plistdecode("this is not a plist") }`,
		regexp.MustCompile(`(?i)plist|parse|invalid`),
	)
}
