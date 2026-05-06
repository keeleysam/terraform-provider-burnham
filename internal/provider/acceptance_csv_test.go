package provider

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
)

func TestAcc_CSVEncode_Basic(t *testing.T) {
	runOutputTest(t,
		`
				output "test" {
					value = provider::burnham::csvencode([
						{ name = "alice", email = "alice@example.com" },
						{ name = "bob", email = "bob@example.com" },
					])
				}
			`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("email,name\nalice@example.com,alice\nbob@example.com,bob\n")),
	)
}

func TestAcc_CSVEncode_WithOptions(t *testing.T) {
	runOutputTest(t,
		`
				output "test" {
					value = provider::burnham::csvencode(
						[{ name = "alice", role = "admin" }],
						{ columns = ["name", "role"], no_header = true }
					)
				}
			`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("alice,admin\n")),
	)
}

func TestAcc_CSVEncode_TypeCoercion(t *testing.T) {
	runOutputTest(t,
		`
				output "test" {
					value = provider::burnham::csvencode(
						[{ name = "alice", count = 42, active = true }],
						{ columns = ["name", "count", "active"] }
					)
				}
			`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("name,count,active\nalice,42,true\n")),
	)
}

// ─── yamlencode ──────────────────────────────────────────────────

// ─── vdfdecode / vdfencode ───────────────────────────────────────

// ─── kdldecode / kdlencode ───────────────────────────────────────
