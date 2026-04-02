package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"
	"github.com/hashicorp/terraform-plugin-testing/helper/resource"
	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
	"github.com/hashicorp/terraform-plugin-testing/tfversion"
)

var testAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"burnham": providerserver.NewProtocol6WithError(New()),
}

var testAccTerraformVersionChecks = []tfversion.TerraformVersionCheck{
	tfversion.SkipBelow(tfversion.Version1_8_0),
}

// ─── jsonencode ───────────────────────────────────────────────────

func TestAcc_JSONEncode_DefaultTabs(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks:   testAccTerraformVersionChecks,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: `output "test" { value = provider::burnham::jsonencode({ name = "hello", count = 2 }) }`,
			ConfigStateChecks: []statecheck.StateCheck{
				statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("{\n\t\"count\": 2,\n\t\"name\": \"hello\"\n}")),
			},
		}},
	})
}

func TestAcc_JSONEncode_CustomIndent(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks:   testAccTerraformVersionChecks,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: `output "test" { value = provider::burnham::jsonencode({ a = 1 }, "  ") }`,
			ConfigStateChecks: []statecheck.StateCheck{
				statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("{\n  \"a\": 1\n}")),
			},
		}},
	})
}

// ─── hujsondecode ────────────────────────────────────────────────

func TestAcc_HuJSONDecode_WithComments(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks:   testAccTerraformVersionChecks,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: `
				locals {
					decoded = provider::burnham::hujsondecode("{// comment\n\"key\": \"value\",}")
				}
				output "test" { value = local.decoded.key }
			`,
			ConfigStateChecks: []statecheck.StateCheck{
				statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("value")),
			},
		}},
	})
}

func TestAcc_HuJSONDecode_Invalid(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks:   testAccTerraformVersionChecks,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config:      `output "test" { value = provider::burnham::hujsondecode("{bad}") }`,
			ExpectError: regexp.MustCompile(`Invalid HuJSON`),
		}},
	})
}

// ─── hujsonencode ────────────────────────────────────────────────

func TestAcc_HuJSONEncode_Basic(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks:   testAccTerraformVersionChecks,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: `output "test" { value = provider::burnham::hujsonencode({ key = "value" }) }`,
			ConfigStateChecks: []statecheck.StateCheck{
				statecheck.ExpectKnownOutputValue("test", knownvalue.NotNull()),
			},
		}},
	})
}

// ─── plistdecode ─────────────────────────────────────────────────

func TestAcc_PlistDecode_XML(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks:   testAccTerraformVersionChecks,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: `
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
			ConfigStateChecks: []statecheck.StateCheck{
				statecheck.ExpectKnownOutputValue("name", knownvalue.StringExact("Test")),
				statecheck.ExpectKnownOutputValue("version", knownvalue.Int64Exact(1)),
				statecheck.ExpectKnownOutputValue("enabled", knownvalue.Bool(true)),
			},
		}},
	})
}

func TestAcc_PlistDecode_DateTaggedObject(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks:   testAccTerraformVersionChecks,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: `
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
			ConfigStateChecks: []statecheck.StateCheck{
				statecheck.ExpectKnownOutputValue("type", knownvalue.StringExact("date")),
				statecheck.ExpectKnownOutputValue("value", knownvalue.StringExact("2025-06-01T00:00:00Z")),
			},
		}},
	})
}

func TestAcc_PlistDecode_IntegerVsReal(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks:   testAccTerraformVersionChecks,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: `
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
			ConfigStateChecks: []statecheck.StateCheck{
				statecheck.ExpectKnownOutputValue("int_val", knownvalue.Int64Exact(5)),
				statecheck.ExpectKnownOutputValue("real_type", knownvalue.StringExact("real")),
				statecheck.ExpectKnownOutputValue("frac_val", knownvalue.Float64Exact(3.14)),
			},
		}},
	})
}

// ─── plistencode ─────────────────────────────────────────────────

func TestAcc_PlistEncode_XML(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks:   testAccTerraformVersionChecks,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: `output "test" { value = provider::burnham::plistencode({ Name = "Test", Version = 1 }) }`,
			ConfigStateChecks: []statecheck.StateCheck{
				statecheck.ExpectKnownOutputValue("test", knownvalue.NotNull()),
			},
		}},
	})
}

func TestAcc_PlistEncode_InvalidFormat(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks:   testAccTerraformVersionChecks,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config:      `output "test" { value = provider::burnham::plistencode({ a = 1 }, "yaml") }`,
			ExpectError: regexp.MustCompile(`unsupported plist`),
		}},
	})
}

// ─── plistdate ───────────────────────────────────────────────────

func TestAcc_PlistDate_Valid(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks:   testAccTerraformVersionChecks,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: `
				locals {
					d = provider::burnham::plistdate("2025-06-01T00:00:00Z")
				}
				output "type" { value = local.d.__plist_type }
				output "value" { value = local.d.value }
			`,
			ConfigStateChecks: []statecheck.StateCheck{
				statecheck.ExpectKnownOutputValue("type", knownvalue.StringExact("date")),
				statecheck.ExpectKnownOutputValue("value", knownvalue.StringExact("2025-06-01T00:00:00Z")),
			},
		}},
	})
}

func TestAcc_PlistDate_Invalid(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks:   testAccTerraformVersionChecks,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config:      `output "test" { value = provider::burnham::plistdate("not-a-date") }`,
			ExpectError: regexp.MustCompile(`Invalid RFC 3339`),
		}},
	})
}

// ─── plistdata ───────────────────────────────────────────────────

func TestAcc_PlistData_Valid(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks:   testAccTerraformVersionChecks,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: `
				locals {
					d = provider::burnham::plistdata("SGVsbG8=")
				}
				output "type" { value = local.d.__plist_type }
				output "value" { value = local.d.value }
			`,
			ConfigStateChecks: []statecheck.StateCheck{
				statecheck.ExpectKnownOutputValue("type", knownvalue.StringExact("data")),
				statecheck.ExpectKnownOutputValue("value", knownvalue.StringExact("SGVsbG8=")),
			},
		}},
	})
}

func TestAcc_PlistData_Invalid(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks:   testAccTerraformVersionChecks,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config:      `output "test" { value = provider::burnham::plistdata("!!!invalid!!!") }`,
			ExpectError: regexp.MustCompile(`Invalid base64`),
		}},
	})
}

// ─── plistreal ───────────────────────────────────────────────────

func TestAcc_PlistReal_Valid(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks:   testAccTerraformVersionChecks,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: `
				locals {
					r = provider::burnham::plistreal(2)
				}
				output "type" { value = local.r.__plist_type }
				output "value" { value = local.r.value }
			`,
			ConfigStateChecks: []statecheck.StateCheck{
				statecheck.ExpectKnownOutputValue("type", knownvalue.StringExact("real")),
				statecheck.ExpectKnownOutputValue("value", knownvalue.StringExact("2")),
			},
		}},
	})
}

// ─── Round-trips ─────────────────────────────────────────────────

func TestAcc_PlistRoundTrip_PreservesTypes(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks:   testAccTerraformVersionChecks,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: `
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
			ConfigStateChecks: []statecheck.StateCheck{
				statecheck.ExpectKnownOutputValue("encoded", knownvalue.NotNull()),
			},
		}},
	})
}

// ─── inidecode ───────────────────────────────────────────────────

func TestAcc_INIDecode_Basic(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks:   testAccTerraformVersionChecks,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: `
				locals {
					ini = provider::burnham::inidecode("[database]\nhost = localhost\nport = 5432\n")
				}
				output "host" { value = local.ini.database.host }
				output "port" { value = local.ini.database.port }
			`,
			ConfigStateChecks: []statecheck.StateCheck{
				statecheck.ExpectKnownOutputValue("host", knownvalue.StringExact("localhost")),
				statecheck.ExpectKnownOutputValue("port", knownvalue.StringExact("5432")),
			},
		}},
	})
}

// ─── iniencode ───────────────────────────────────────────────────

func TestAcc_INIEncode_Basic(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks:   testAccTerraformVersionChecks,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: `
				output "test" {
					value = provider::burnham::iniencode({
						database = { host = "localhost", port = "5432" }
					})
				}
			`,
			ConfigStateChecks: []statecheck.StateCheck{
				statecheck.ExpectKnownOutputValue("test", knownvalue.NotNull()),
			},
		}},
	})
}

// ─── csvencode ───────────────────────────────────────────────────

func TestAcc_CSVEncode_Basic(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks:   testAccTerraformVersionChecks,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: `
				output "test" {
					value = provider::burnham::csvencode([
						{ name = "alice", email = "alice@example.com" },
						{ name = "bob", email = "bob@example.com" },
					])
				}
			`,
			ConfigStateChecks: []statecheck.StateCheck{
				statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("email,name\nalice@example.com,alice\nbob@example.com,bob\n")),
			},
		}},
	})
}

func TestAcc_CSVEncode_WithOptions(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks:   testAccTerraformVersionChecks,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: `
				output "test" {
					value = provider::burnham::csvencode(
						[{ name = "alice", role = "admin" }],
						{ columns = ["name", "role"], no_header = true }
					)
				}
			`,
			ConfigStateChecks: []statecheck.StateCheck{
				statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("alice,admin\n")),
			},
		}},
	})
}

func TestAcc_CSVEncode_TypeCoercion(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks:   testAccTerraformVersionChecks,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: `
				output "test" {
					value = provider::burnham::csvencode(
						[{ name = "alice", count = 42, active = true }],
						{ columns = ["name", "count", "active"] }
					)
				}
			`,
			ConfigStateChecks: []statecheck.StateCheck{
				statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("name,count,active\nalice,42,true\n")),
			},
		}},
	})
}

// ─── Round-trips ─────────────────────────────────────────────────

func TestAcc_INIRoundTrip(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks:   testAccTerraformVersionChecks,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: `
				locals {
					input   = "[db]\nhost = localhost\nport = 5432\n"
					decoded = provider::burnham::inidecode(local.input)
					encoded = provider::burnham::iniencode(local.decoded)
				}
				output "host" { value = local.decoded.db.host }
			`,
			ConfigStateChecks: []statecheck.StateCheck{
				statecheck.ExpectKnownOutputValue("host", knownvalue.StringExact("localhost")),
			},
		}},
	})
}

func TestAcc_HuJSONRoundTrip(t *testing.T) {
	resource.UnitTest(t, resource.TestCase{
		TerraformVersionChecks:   testAccTerraformVersionChecks,
		ProtoV6ProviderFactories: testAccProtoV6ProviderFactories,
		Steps: []resource.TestStep{{
			Config: `
				locals {
					input   = "{// comment\n\"key\": \"value\",\n\"num\": 42,}"
					decoded = provider::burnham::hujsondecode(local.input)
					encoded = provider::burnham::hujsonencode(local.decoded)
				}
				output "key" { value = local.decoded.key }
			`,
			ConfigStateChecks: []statecheck.StateCheck{
				statecheck.ExpectKnownOutputValue("key", knownvalue.StringExact("value")),
			},
		}},
	})
}
