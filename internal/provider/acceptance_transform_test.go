package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
)

// ─── jmespath_query ─────────────────────────────────────────────

func TestAcc_JMESPathQuery_FieldExtract(t *testing.T) {
	runOutputTest(t,
		`output "test" {
			value = provider::burnham::jmespath_query(
				{ user = { name = "alice", age = 30 } },
				"user.name",
			)
		}`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("alice")),
	)
}

func TestAcc_JMESPathQuery_Projection(t *testing.T) {
	runOutputTest(t,
		`output "test" {
			value = provider::burnham::jmespath_query(
				{ items = [{ id = 1 }, { id = 2 }, { id = 3 }] },
				"items[*].id",
			)
		}`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.TupleExact([]knownvalue.Check{
			knownvalue.Int64Exact(1),
			knownvalue.Int64Exact(2),
			knownvalue.Int64Exact(3),
		})),
	)
}

func TestAcc_JMESPathQuery_Filter(t *testing.T) {
	runOutputTest(t,
		`output "test" {
			value = provider::burnham::jmespath_query(
				{ items = [{ name = "a", on = true }, { name = "b", on = false }, { name = "c", on = true }] },
				"items[?on].name",
			)
		}`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.TupleExact([]knownvalue.Check{
			knownvalue.StringExact("a"),
			knownvalue.StringExact("c"),
		})),
	)
}

func TestAcc_JMESPathQuery_NoMatch(t *testing.T) {
	// Wrap in coalesce because a bare-null output isn't surfaced in state for
	// ExpectKnownOutputValue to see; consume the null inside Terraform instead.
	runOutputTest(t,
		`output "test" {
			value = coalesce(provider::burnham::jmespath_query({ a = 1 }, "missing"), "absent")
		}`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("absent")),
	)
}

func TestAcc_JMESPathQuery_InvalidExpression(t *testing.T) {
	runErrorTest(t,
		`output "test" { value = provider::burnham::jmespath_query({}, "[[[") }`,
		regexp.MustCompile(`(?i)jmespath|parse|syntax`),
	)
}

// ─── jsonpath_query (RFC 9535) ──────────────────────────────────

func TestAcc_JSONPathQuery_RootFieldList(t *testing.T) {
	runOutputTest(t,
		`output "test" {
			value = provider::burnham::jsonpath_query(
				{ store = { book = [{ title = "A" }, { title = "B" }] } },
				"$.store.book[*].title",
			)
		}`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.TupleExact([]knownvalue.Check{
			knownvalue.StringExact("A"),
			knownvalue.StringExact("B"),
		})),
	)
}

func TestAcc_JSONPathQuery_DescendantSegment(t *testing.T) {
	// Single-match descendant to keep ordering deterministic; visit order across
	// object keys would otherwise depend on Go map iteration.
	runOutputTest(t,
		`output "test" {
			value = provider::burnham::jsonpath_query(
				{ a = { b = { c = 42 } } },
				"$..c",
			)
		}`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.TupleExact([]knownvalue.Check{
			knownvalue.Int64Exact(42),
		})),
	)
}

func TestAcc_JSONPathQuery_Filter(t *testing.T) {
	runOutputTest(t,
		`output "test" {
			value = provider::burnham::jsonpath_query(
				{ items = [{ p = 5 }, { p = 12 }, { p = 3 }] },
				"$.items[?@.p < 10].p",
			)
		}`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.TupleExact([]knownvalue.Check{
			knownvalue.Int64Exact(5),
			knownvalue.Int64Exact(3),
		})),
	)
}

func TestAcc_JSONPathQuery_NoMatch(t *testing.T) {
	runOutputTest(t,
		`output "test" {
			value = provider::burnham::jsonpath_query({ a = 1 }, "$.missing")
		}`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.TupleExact([]knownvalue.Check{})),
	)
}

func TestAcc_JSONPathQuery_InvalidExpression(t *testing.T) {
	runErrorTest(t,
		`output "test" { value = provider::burnham::jsonpath_query({}, "no-leading-dollar") }`,
		regexp.MustCompile(`(?i)jsonpath|invalid|parse`),
	)
}

// ─── json_patch (RFC 6902) ────────────────────────────────

func TestAcc_JSONPatch_AddReplaceRemove(t *testing.T) {
	runOutputTest(t,
		`output "test" {
			value = provider::burnham::json_patch(
				{ name = "web", replicas = 2, env = { LOG_LEVEL = "info", DEBUG = "true" } },
				[
					{ op = "replace", path = "/replicas",       value = 5 },
					{ op = "add",     path = "/env/REGION",     value = "us-east-1" },
					{ op = "remove",  path = "/env/DEBUG" },
				],
			)
		}`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.ObjectExact(map[string]knownvalue.Check{
			"name":     knownvalue.StringExact("web"),
			"replicas": knownvalue.Int64Exact(5),
			"env": knownvalue.ObjectExact(map[string]knownvalue.Check{
				"LOG_LEVEL": knownvalue.StringExact("info"),
				"REGION":    knownvalue.StringExact("us-east-1"),
			}),
		})),
	)
}

func TestAcc_JSONPatch_TestOpFailure(t *testing.T) {
	runErrorTest(t,
		`output "test" {
			value = provider::burnham::json_patch(
				{ a = 1 },
				[{ op = "test", path = "/a", value = 2 }],
			)
		}`,
		regexp.MustCompile(`(?i)patch|test`),
	)
}

func TestAcc_JSONPatch_InvalidOp(t *testing.T) {
	runErrorTest(t,
		`output "test" {
			value = provider::burnham::json_patch(
				{},
				[{ op = "frobnicate", path = "/x", value = 1 }],
			)
		}`,
		regexp.MustCompile(`(?i)patch|invalid|operation|unexpected`),
	)
}

// ─── json_merge_patch (RFC 7396) ──────────────────────────

func TestAcc_JSONMergePatch_OverrideAndDelete(t *testing.T) {
	runOutputTest(t,
		`output "test" {
			value = provider::burnham::json_merge_patch(
				{ replicas = 2, env = { LOG_LEVEL = "info", DEBUG = "true" } },
				{ replicas = 10, env = { LOG_LEVEL = "warn", DEBUG = null } },
			)
		}`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.ObjectExact(map[string]knownvalue.Check{
			"replicas": knownvalue.Int64Exact(10),
			"env": knownvalue.ObjectExact(map[string]knownvalue.Check{
				"LOG_LEVEL": knownvalue.StringExact("warn"),
			}),
		})),
	)
}

func TestAcc_JSONPatch_EmptyPatchIsIdentity(t *testing.T) {
	runOutputTest(t,
		`output "test" {
			value = provider::burnham::json_patch({ a = 1, b = 2 }, [])
		}`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.ObjectExact(map[string]knownvalue.Check{
			"a": knownvalue.Int64Exact(1),
			"b": knownvalue.Int64Exact(2),
		})),
	)
}

func TestAcc_JSONMergePatch_ScalarRootReplaces(t *testing.T) {
	// RFC 7396 §1: when the patch is a non-object value, the result is the patch — wholesale replacement.
	runOutputTest(t,
		`output "test" {
			value = provider::burnham::json_merge_patch({ a = 1 }, "replaced")
		}`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("replaced")),
	)
}

func TestAcc_JSONMergePatch_ArrayReplaced(t *testing.T) {
	// RFC 7396: arrays in the patch *replace* the target array; they aren't merged element-wise.
	runOutputTest(t,
		`output "test" {
			value = provider::burnham::json_merge_patch(
				{ tags = ["a", "b", "c"] },
				{ tags = ["x"] },
			)
		}`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.ObjectExact(map[string]knownvalue.Check{
			"tags": knownvalue.TupleExact([]knownvalue.Check{
				knownvalue.StringExact("x"),
			}),
		})),
	)
}
