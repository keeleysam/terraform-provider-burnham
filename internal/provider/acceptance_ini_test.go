package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
)

func TestAcc_INIDecode_Basic(t *testing.T) {
	runOutputTest(t,
		`
				locals {
					ini = provider::burnham::inidecode("[database]\nhost = localhost\nport = 5432\n")
				}
				output "host" { value = local.ini.database.host }
				output "port" { value = local.ini.database.port }
			`,
		statecheck.ExpectKnownOutputValue("host", knownvalue.StringExact("localhost")),
		statecheck.ExpectKnownOutputValue("port", knownvalue.StringExact("5432")),
	)
}

// ─── iniencode ───────────────────────────────────────────────────

func TestAcc_INIEncode_Basic(t *testing.T) {
	runOutputTest(t,
		`
				output "test" {
					value = provider::burnham::iniencode({
						database = { host = "localhost", port = "5432" }
					})
				}
			`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.NotNull()),
	)
}

// ─── csvencode ───────────────────────────────────────────────────

func TestAcc_INIRoundTrip(t *testing.T) {
	runOutputTest(t,
		`
				locals {
					input   = "[db]\nhost = localhost\nport = 5432\n"
					decoded = provider::burnham::inidecode(local.input)
					encoded = provider::burnham::iniencode(local.decoded)
				}
				output "host" { value = local.decoded.db.host }
			`,
		statecheck.ExpectKnownOutputValue("host", knownvalue.StringExact("localhost")),
	)
}
