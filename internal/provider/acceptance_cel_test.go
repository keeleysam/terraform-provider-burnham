package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
)

// ─── celencode ──────────────────────────────────────────────────

func TestAcc_CELEncode_ComparisonWithAlias(t *testing.T) {
	runOutputTest(t,
		`output "test" {
			value = provider::burnham::celencode({
				eq = [{ ident = "device.os_type" }, { ident = "OsType.DESKTOP_MAC" }]
			})
		}`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact(`device.os_type == OsType.DESKTOP_MAC`)),
	)
}

func TestAcc_CELEncode_ComplexConjunction(t *testing.T) {
	runOutputTest(t,
		`output "test" {
			value = provider::burnham::celencode({
				"&&" = [
					{ "==" = [{ ident = "a" }, "b"] },
					{ "in" = [{ ident = "origin.region_code" }, ["US", "CA"]] },
					{ call = { target = { ident = "resource.name" }, function = "startsWith", args = ["prod-"] } },
				]
			})
		}`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact(
			`a == "b" && origin.region_code in ["US", "CA"] && resource.name.startsWith("prod-")`,
		)),
	)
}

func TestAcc_CELEncode_ExistsMacro(t *testing.T) {
	runOutputTest(t,
		`output "test" {
			value = provider::burnham::celencode({
				call = {
					target   = { ident = "user.groups" }
					function = "exists"
					args = [
						{ ident = "g" },
						{ call = { target = { ident = "g" }, function = "startsWith", args = ["admin-"] } },
					]
				}
			})
		}`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact(`user.groups.exists(g, g.startsWith("admin-"))`)),
	)
}

func TestAcc_CELEncode_BuiltFromData(t *testing.T) {
	runOutputTest(t,
		`output "test" {
			value = provider::burnham::celencode({
				or = [for r in ["US", "CA"] : { "==" = [{ ident = "origin.region_code" }, r] }]
			})
		}`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact(`origin.region_code == "US" || origin.region_code == "CA"`)),
	)
}

func TestAcc_CELEncode_ArityError(t *testing.T) {
	runErrorTest(t,
		`output "test" {
			value = provider::burnham::celencode({ "==" = [{ ident = "a" }] })
		}`,
		regexp.MustCompile(`got 1`),
	)
}

func TestAcc_CELEncode_UnknownKeyError(t *testing.T) {
	runErrorTest(t,
		`output "test" {
			value = provider::burnham::celencode({ bogus = "x" })
		}`,
		regexp.MustCompile(`unknown node key`),
	)
}

func TestAcc_CELEncode_TwoVariableComprehension(t *testing.T) {
	runOutputTest(t,
		`output "test" {
			value = provider::burnham::celencode({
				call = {
					target   = { ident = "m" }
					function = "all"
					args     = [{ ident = "k" }, { ident = "v" }, { ">" = [{ ident = "v" }, 0] }]
				}
			})
		}`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact(`m.all(k, v, v > 0)`)),
	)
}

func TestAcc_CELEncode_OptionalTypes(t *testing.T) {
	runOutputTest(t,
		`output "test" {
			value = provider::burnham::celencode({
				"&&" = [
					{ ident = "msg.?field" },
					{ "==" = [["a", { optional = { ident = "b" } }], { ident = "expected" }] },
				]
			})
		}`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact(`msg.?field && ["a", ?b] == expected`)),
	)
}

// ─── cel (evaluate) ─────────────────────────────────────────────

func TestAcc_CELEvaluate_Evaluate(t *testing.T) {
	runOutputTest(t,
		`output "test" {
			value = provider::burnham::celevaluate(
				"request.tier == \"prod\" && \"admin\" in request.roles",
				{ vars = { request = { tier = "prod", roles = ["viewer", "admin"] } } },
			)
		}`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.Bool(true)),
	)
}

func TestAcc_CELEvaluate_ComputesValue(t *testing.T) {
	runOutputTest(t,
		`output "test" {
			value = provider::burnham::celevaluate(
				"items.filter(i, i > 1).size()",
				{ vars = { items = [1, 2, 3, 4] } },
			)
		}`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.Int64Exact(3)),
	)
}

func TestAcc_CELEvaluate_UndeclaredVarError(t *testing.T) {
	runErrorTest(t,
		`output "test" {
			value = provider::burnham::celevaluate("missing + 1")
		}`,
		regexp.MustCompile(`undeclared`),
	)
}

// ─── celvalidate (bool) ─────────────────────────────────────────

func TestAcc_CELValidate_True(t *testing.T) {
	runOutputTest(t,
		`output "test" {
			value = provider::burnham::celvalidate("resource.name.startsWith('prod-') && x.exists(i, i > 0)")
		}`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.Bool(true)),
	)
}

func TestAcc_CELValidate_False(t *testing.T) {
	runOutputTest(t,
		`output "test" {
			value = provider::burnham::celvalidate("a &&")
		}`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.Bool(false)),
	)
}

// ─── celformat ──────────────────────────────────────────────────

func TestAcc_CELFormat_Normalizes(t *testing.T) {
	runOutputTest(t,
		`output "test" {
			value = provider::burnham::celformat("a   &&  b == 'x'")
		}`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact(`a && b == "x"`)),
	)
}

// ─── celdecode (round-trip) ─────────────────────────────────────

func celDecodeRoundTrip(t *testing.T, notation string) {
	runOutputTest(t,
		`output "test" {
			value = provider::burnham::celencode(provider::burnham::celdecode(
				"device.os_type == OsType.DESKTOP_MAC && origin.region_code in ['US', 'CA']",
				{ notation = "`+notation+`" },
			))
		}`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact(
			`device.os_type == OsType.DESKTOP_MAC && origin.region_code in ["US", "CA"]`,
		)),
	)
}

func TestAcc_CELDecode_RoundTripCanonical(t *testing.T) { celDecodeRoundTrip(t, "canonical") }
func TestAcc_CELDecode_RoundTripStandard(t *testing.T)  { celDecodeRoundTrip(t, "standard") }
func TestAcc_CELDecode_RoundTripAliased(t *testing.T)   { celDecodeRoundTrip(t, "aliased") }

func TestAcc_CELFormat_Wrap(t *testing.T) {
	runOutputTest(t,
		`output "test" {
			value = provider::burnham::celformat(
				"aaaaa == 1 && bbbbb == 2 && ccccc == 3",
				{ format = { wrap_on_column = 20 } },
			)
		}`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("aaaaa == 1 && bbbbb == 2 &&\nccccc == 3")),
	)
}

func TestAcc_CELFormat_WrapBeforeOperator(t *testing.T) {
	runOutputTest(t,
		`output "test" {
			value = provider::burnham::celformat(
				"aaaaa == 1 && bbbbb == 2 && ccccc == 3",
				{ format = { wrap_on_column = 20, wrap_after_column_limit = false } },
			)
		}`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("aaaaa == 1 && bbbbb == 2\n&& ccccc == 3")),
	)
}

// wrap_on_operators is documented to take friendly operator symbols ("&&", "||"), which the function must translate to the cel-go unparser's internal operator IDs ("_&&_", "_||_"). Passing the raw symbol straight through makes cel-go reject it ("Unsupported operator: &&").
func TestAcc_CELFormat_WrapOnOperatorsSymbol(t *testing.T) {
	runOutputTest(t,
		`output "test" {
			value = provider::burnham::celformat(
				"aaaaa == 1 && bbbbb == 2 && ccccc == 3",
				{ format = { wrap_on_column = 20, wrap_on_operators = ["&&"] } },
			)
		}`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("aaaaa == 1 && bbbbb == 2 &&\nccccc == 3")),
	)
}

// The cel-go internal operator ID must keep working too, so authors who pass "_&&_" are not broken by the symbol translation.
func TestAcc_CELFormat_WrapOnOperatorsID(t *testing.T) {
	runOutputTest(t,
		`output "test" {
			value = provider::burnham::celformat(
				"aaaaa == 1 && bbbbb == 2 && ccccc == 3",
				{ format = { wrap_on_column = 20, wrap_on_operators = ["_&&_"] } },
			)
		}`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("aaaaa == 1 && bbbbb == 2 &&\nccccc == 3")),
	)
}

func TestAcc_CELFormat_InvalidError(t *testing.T) {
	runErrorTest(t,
		`output "test" {
			value = provider::burnham::celformat("a &&")
		}`,
		regexp.MustCompile(`Syntax error`),
	)
}
