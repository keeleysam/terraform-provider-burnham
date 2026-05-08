package provider

import (
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
)

// ─── uuid_v5 (RFC 9562 §5.5 SHA-1 namespace UUID) ──────────────────────

func TestAcc_UUIDv5_DNSExample(t *testing.T) {
	// Canonical RFC 4122 / 9562 expected value: uuid5(NAMESPACE_DNS, "example.com").
	// Cross-checked against Python's uuid.uuid5(uuid.NAMESPACE_DNS, "example.com").
	runOutputTest(t,
		`output "test" { value = provider::burnham::uuid_v5("dns", "example.com") }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("cfbff0d1-9375-5685-968c-48ce8b15ae17")),
	)
}

func TestAcc_UUIDv5_URLNamespace(t *testing.T) {
	// uuid5(NAMESPACE_URL, "https://example.com").
	// Python: uuid.uuid5(uuid.NAMESPACE_URL, "https://example.com")
	runOutputTest(t,
		`output "test" { value = provider::burnham::uuid_v5("url", "https://example.com") }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("4fd35a71-71ef-5a55-a9d9-aa75c889a6d0")),
	)
}

func TestAcc_UUIDv5_CustomNamespaceUUID(t *testing.T) {
	// Pass a literal UUID as the namespace. Result must match uuid5(parsed UUID, name).
	// That namespace UUID is in fact uuid.NameSpaceDNS, so this test proves the literal-UUID path produces the same output as the short name.
	runOutputTest(t,
		`output "test" { value = provider::burnham::uuid_v5("6ba7b810-9dad-11d1-80b4-00c04fd430c8", "example.com") }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("cfbff0d1-9375-5685-968c-48ce8b15ae17")),
	)
}

func TestAcc_UUIDv5_Determinism(t *testing.T) {
	// Two calls with identical inputs must produce identical outputs (deterministic).
	runOutputTest(t,
		`output "test" {
		   value = provider::burnham::uuid_v5("dns", "stable") == provider::burnham::uuid_v5("dns", "stable")
		 }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.Bool(true)),
	)
}

func TestAcc_UUIDv5_RejectsBadNamespace(t *testing.T) {
	runErrorTest(t,
		`output "test" { value = provider::burnham::uuid_v5("not-a-namespace", "anything") }`,
		regexp.MustCompile(`(?is)namespace\s+must\s+be`),
	)
}

// ─── uuid_v7 (RFC 9562 §5.7 sortable Unix-time UUID) ───────────────────

func TestAcc_UUIDv7_StructuralVersionAndVariant(t *testing.T) {
	// We don't lock in a full literal here because the rand_a / rand_b bits depend on the HMAC choice; we just verify the version nibble (7) and variant nibble (RFC 4122 = 8/9/a/b).
	runOutputTest(t,
		`output "test" { value = provider::burnham::uuid_v7("2026-05-08T12:00:00Z", "stable-entropy") }`,
		statecheck.ExpectKnownOutputValue(
			"test",
			knownvalue.StringRegexp(regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-7[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)),
		),
	)
}

func TestAcc_UUIDv7_Determinism(t *testing.T) {
	// Same (timestamp, entropy) → same UUID.
	runOutputTest(t,
		`output "test" {
		   value = provider::burnham::uuid_v7("2026-05-08T12:00:00Z", "x") == provider::burnham::uuid_v7("2026-05-08T12:00:00Z", "x")
		 }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.Bool(true)),
	)
}

func TestAcc_UUIDv7_DifferentEntropyDiffers(t *testing.T) {
	runOutputTest(t,
		`output "test" {
		   value = provider::burnham::uuid_v7("2026-05-08T12:00:00Z", "a") != provider::burnham::uuid_v7("2026-05-08T12:00:00Z", "b")
		 }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.Bool(true)),
	)
}

func TestAcc_UUIDv7_TimestampEmbedded(t *testing.T) {
	// Round-trip: build a v7 with a known timestamp, inspect it, confirm unix_ts_ms matches.
	// 2026-05-08T12:00:00Z = 1_778_241_600 seconds = 1_778_241_600_000 ms since epoch.
	runOutputTest(t,
		`output "test" {
		   value = provider::burnham::uuid_inspect(provider::burnham::uuid_v7("2026-05-08T12:00:00Z", "x")).unix_ts_ms
		 }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.Int64Exact(1778241600000)),
	)
}

func TestAcc_UUIDv7_RejectsBadTimestamp(t *testing.T) {
	runErrorTest(t,
		`output "test" { value = provider::burnham::uuid_v7("not-a-timestamp", "") }`,
		regexp.MustCompile(`(?is)timestamp\s+must\s+be\s+RFC\s+3339`),
	)
}

func TestAcc_UUIDv7_RejectsPreEpochTimestamp(t *testing.T) {
	runErrorTest(t,
		`output "test" { value = provider::burnham::uuid_v7("1969-12-31T00:00:00Z", "") }`,
		regexp.MustCompile(`(?is)48-bit\s+Unix-millisecond\s+range`),
	)
}

// ─── uuid_inspect ──────────────────────────────────────────────────────

func TestAcc_UUIDInspect_V4(t *testing.T) {
	// v4: version 4, RFC 4122 variant, no timestamp.
	runOutputTest(t,
		`output "test" { value = provider::burnham::uuid_inspect("550e8400-e29b-41d4-a716-446655440000") }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.ObjectExact(map[string]knownvalue.Check{
			"version":    knownvalue.Int64Exact(4),
			"variant":    knownvalue.StringExact("RFC 4122"),
			"timestamp":  knownvalue.Null(),
			"unix_ts_ms": knownvalue.Null(),
		})),
	)
}

func TestAcc_UUIDInspect_V5_NoTimestamp(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::uuid_inspect("cfbff0d1-9375-5685-968a-48ce8b50e3a9") }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.ObjectExact(map[string]knownvalue.Check{
			"version":    knownvalue.Int64Exact(5),
			"variant":    knownvalue.StringExact("RFC 4122"),
			"timestamp":  knownvalue.Null(),
			"unix_ts_ms": knownvalue.Null(),
		})),
	)
}

func TestAcc_UUIDInspect_RejectsBadInput(t *testing.T) {
	runErrorTest(t,
		`output "test" { value = provider::burnham::uuid_inspect("definitely-not-a-uuid") }`,
		regexp.MustCompile(`(?is)not\s+a\s+valid\s+UUID`),
	)
}

// ─── nanoid ────────────────────────────────────────────────────────────

func TestAcc_Nanoid_DefaultSizeAndAlphabet(t *testing.T) {
	// Default: 21 chars from the URL-safe alphabet.
	runOutputTest(t,
		`output "test" { value = provider::burnham::nanoid("seed-default") }`,
		statecheck.ExpectKnownOutputValue(
			"test",
			knownvalue.StringRegexp(regexp.MustCompile(`^[_\-0-9A-Za-z]{21}$`)),
		),
	)
}

func TestAcc_Nanoid_Determinism(t *testing.T) {
	runOutputTest(t,
		`output "test" {
		   value = provider::burnham::nanoid("same-seed") == provider::burnham::nanoid("same-seed")
		 }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.Bool(true)),
	)
}

func TestAcc_Nanoid_DifferentSeedsDiffer(t *testing.T) {
	runOutputTest(t,
		`output "test" {
		   value = provider::burnham::nanoid("a") != provider::burnham::nanoid("b")
		 }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.Bool(true)),
	)
}

func TestAcc_Nanoid_CustomSize(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::nanoid("seed", { size = 8 }) }`,
		statecheck.ExpectKnownOutputValue(
			"test",
			knownvalue.StringRegexp(regexp.MustCompile(`^[_\-0-9A-Za-z]{8}$`)),
		),
	)
}

func TestAcc_Nanoid_CustomAlphabet(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::nanoid("seed", { alphabet = "0123456789", size = 6 }) }`,
		statecheck.ExpectKnownOutputValue(
			"test",
			knownvalue.StringRegexp(regexp.MustCompile(`^[0-9]{6}$`)),
		),
	)
}

func TestAcc_Nanoid_RejectsEmptyAlphabet(t *testing.T) {
	runErrorTest(t,
		`output "test" { value = provider::burnham::nanoid("seed", { alphabet = "" }) }`,
		regexp.MustCompile(`(?is)alphabet\s+must\s+be\s+non-empty`),
	)
}

func TestAcc_Nanoid_RejectsDuplicateRunes(t *testing.T) {
	runErrorTest(t,
		`output "test" { value = provider::burnham::nanoid("seed", { alphabet = "aabc" }) }`,
		regexp.MustCompile(`(?is)no\s+duplicate\s+characters`),
	)
}

func TestAcc_Nanoid_RejectsTooLarge(t *testing.T) {
	runErrorTest(t,
		`output "test" { value = provider::burnham::nanoid("seed", { size = 9999 }) }`,
		regexp.MustCompile(`(?is)size\s+must\s+be\s+in\s+\[1,\s*1024\]`),
	)
}

func TestAcc_Nanoid_UnknownOption(t *testing.T) {
	runErrorTest(t,
		`output "test" { value = provider::burnham::nanoid("seed", { color = "blue" }) }`,
		regexp.MustCompile(`(?is)unknown\s+option\s+key`),
	)
}

// ─── petname ───────────────────────────────────────────────────────────

func TestAcc_Petname_DefaultTwoWords(t *testing.T) {
	// Format: <adjective>-<noun>; both lowercase ASCII.
	runOutputTest(t,
		`output "test" { value = provider::burnham::petname("seed-1") }`,
		statecheck.ExpectKnownOutputValue(
			"test",
			knownvalue.StringRegexp(regexp.MustCompile(`^[a-z]+-[a-z]+$`)),
		),
	)
}

func TestAcc_Petname_Determinism(t *testing.T) {
	runOutputTest(t,
		`output "test" {
		   value = provider::burnham::petname("env-prod") == provider::burnham::petname("env-prod")
		 }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.Bool(true)),
	)
}

func TestAcc_Petname_OneWordIsNounOnly(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::petname("seed", { words = 1 }) }`,
		statecheck.ExpectKnownOutputValue(
			"test",
			knownvalue.StringRegexp(regexp.MustCompile(`^[a-z]+$`)),
		),
	)
}

func TestAcc_Petname_ThreeWordsAdverbAdjectiveNoun(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::petname("seed", { words = 3 }) }`,
		statecheck.ExpectKnownOutputValue(
			"test",
			knownvalue.StringRegexp(regexp.MustCompile(`^[a-z]+-[a-z]+-[a-z]+$`)),
		),
	)
}

func TestAcc_Petname_CustomSeparator(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::petname("seed", { separator = "_" }) }`,
		statecheck.ExpectKnownOutputValue(
			"test",
			knownvalue.StringRegexp(regexp.MustCompile(`^[a-z]+_[a-z]+$`)),
		),
	)
}

func TestAcc_Petname_RejectsTooManyWords(t *testing.T) {
	runErrorTest(t,
		`output "test" { value = provider::burnham::petname("seed", { words = 99 }) }`,
		regexp.MustCompile(`(?is)words\s+must\s+be\s+in\s+\[1,\s*16\]`),
	)
}

func TestAcc_Petname_RejectsZeroWords(t *testing.T) {
	runErrorTest(t,
		`output "test" { value = provider::burnham::petname("seed", { words = 0 }) }`,
		regexp.MustCompile(`(?is)words\s+must\s+be\s+in\s+\[1,\s*16\]`),
	)
}
