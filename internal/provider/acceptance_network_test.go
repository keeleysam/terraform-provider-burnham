package provider

import (
	"math/big"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
)

// ─── CIDR set operations ─────────────────────────────────────────

func TestAcc_CIDRMerge_AdjacentPair(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::cidr_merge(["10.0.0.0/24", "10.0.1.0/24"]) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.ListExact([]knownvalue.Check{
			knownvalue.StringExact("10.0.0.0/23"),
		})),
	)
}

func TestAcc_CIDRSubtract(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::cidr_subtract(["10.0.0.0/30"], ["10.0.0.0/31"]) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.ListExact([]knownvalue.Check{
			knownvalue.StringExact("10.0.0.2/31"),
		})),
	)
}

func TestAcc_CIDRIntersect(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::cidr_intersect(["10.0.0.0/8"], ["10.100.0.0/16"]) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.ListExact([]knownvalue.Check{
			knownvalue.StringExact("10.100.0.0/16"),
		})),
	)
}

func TestAcc_RangeToCIDRs(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::range_to_cidrs("10.0.0.1", "10.0.0.6") }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.ListExact([]knownvalue.Check{
			knownvalue.StringExact("10.0.0.1/32"),
			knownvalue.StringExact("10.0.0.2/31"),
			knownvalue.StringExact("10.0.0.4/31"),
			knownvalue.StringExact("10.0.0.6/32"),
		})),
	)
}

func TestAcc_CIDREnumerate(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::cidr_enumerate("10.0.0.0/24", 2) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.ListExact([]knownvalue.Check{
			knownvalue.StringExact("10.0.0.0/26"),
			knownvalue.StringExact("10.0.0.64/26"),
			knownvalue.StringExact("10.0.0.128/26"),
			knownvalue.StringExact("10.0.0.192/26"),
		})),
	)
}

// ─── Query / containment ────────────────────────────────────────

func TestAcc_IPInCIDR_True(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::ip_in_cidr("10.0.1.50", "10.0.1.0/24") }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.Bool(true)),
	)
}

func TestAcc_CIDROverlaps_False(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::cidr_overlaps("10.0.0.0/24", "10.0.1.0/24") }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.Bool(false)),
	)
}

func TestAcc_CIDRsAreDisjoint_True(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::cidrs_are_disjoint(["10.0.0.0/24", "10.0.1.0/24"]) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.Bool(true)),
	)
}

func TestAcc_CIDRsAreDisjoint_FalseOnContainment(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::cidrs_are_disjoint(["10.0.0.0/8", "10.0.1.0/24"]) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.Bool(false)),
	)
}

func TestAcc_IPIsPrivate_RFC1918(t *testing.T) {
	runOutputTest(t,
		`
				output "rfc1918"  { value = provider::burnham::ip_is_private("192.168.1.1") }
				output "cgnat"    { value = provider::burnham::ip_is_private("100.64.0.1") }
				output "public"   { value = provider::burnham::ip_is_private("8.8.8.8") }
			`,
		statecheck.ExpectKnownOutputValue("rfc1918", knownvalue.Bool(true)),
		statecheck.ExpectKnownOutputValue("cgnat", knownvalue.Bool(true)),
		statecheck.ExpectKnownOutputValue("public", knownvalue.Bool(false)),
	)
}

func TestAcc_CIDRsContainingIP(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::cidrs_containing_ip("10.0.1.5", ["10.0.0.0/8", "10.0.1.0/24", "192.168.0.0/16"]) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.ListExact([]knownvalue.Check{
			knownvalue.StringExact("10.0.0.0/8"),
			knownvalue.StringExact("10.0.1.0/24"),
		})),
	)
}

// ─── Decomposition / arithmetic ──────────────────────────────────

func TestAcc_CIDRFirstIP(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::cidr_first_ip("10.0.0.7/24") }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("10.0.0.0")),
	)
}

func TestAcc_CIDRHostCount(t *testing.T) {
	runOutputTest(t,
		`
				output "slash24" { value = provider::burnham::cidr_host_count("10.0.0.0/24") }
				output "usable"  { value = provider::burnham::cidr_usable_host_count("10.0.0.0/24") }
				output "p2p"     { value = provider::burnham::cidr_usable_host_count("10.0.0.0/31") }
			`,
		statecheck.ExpectKnownOutputValue("slash24", knownvalue.Int64Exact(256)),
		statecheck.ExpectKnownOutputValue("usable", knownvalue.Int64Exact(254)),
		statecheck.ExpectKnownOutputValue("p2p", knownvalue.Int64Exact(2)),
	)
}

func TestAcc_CIDRPrefixLength(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::cidr_prefix_length("10.0.0.0/23") }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.Int64Exact(23)),
	)
}

func TestAcc_IPAdd_PositiveAndNegative(t *testing.T) {
	runOutputTest(t,
		`
				output "plus"  { value = provider::burnham::ip_add("10.0.0.0", 5) }
				output "minus" { value = provider::burnham::ip_add("10.0.0.5", -3) }
			`,
		statecheck.ExpectKnownOutputValue("plus", knownvalue.StringExact("10.0.0.5")),
		statecheck.ExpectKnownOutputValue("minus", knownvalue.StringExact("10.0.0.2")),
	)
}

func TestAcc_IPSubtract(t *testing.T) {
	runOutputTest(t,
		`
				output "positive" { value = provider::burnham::ip_subtract("10.0.0.10", "10.0.0.1") }
				output "negative" { value = provider::burnham::ip_subtract("10.0.0.1", "10.0.0.10") }
				output "zero"     { value = provider::burnham::ip_subtract("10.0.0.5", "10.0.0.5") }
			`,
		statecheck.ExpectKnownOutputValue("positive", knownvalue.Int64Exact(9)),
		statecheck.ExpectKnownOutputValue("negative", knownvalue.Int64Exact(-9)),
		statecheck.ExpectKnownOutputValue("zero", knownvalue.Int64Exact(0)),
	)
}

func TestAcc_IPVersion(t *testing.T) {
	runOutputTest(t,
		`
				output "v4" { value = provider::burnham::ip_version("192.168.1.1") }
				output "v6" { value = provider::burnham::ip_version("2001:db8::1") }
			`,
		statecheck.ExpectKnownOutputValue("v4", knownvalue.Int64Exact(4)),
		statecheck.ExpectKnownOutputValue("v6", knownvalue.Int64Exact(6)),
	)
}

func TestAcc_CIDRWildcard(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::cidr_wildcard("10.0.0.0/24") }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("0.0.0.255")),
	)
}

// ─── NAT64 (RFC 6052) — covers VariadicParameter pattern ─────────

func TestAcc_NAT64Synthesize_DefaultMixedNotation(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::nat64_synthesize("192.0.2.1", "64:ff9b::/96") }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("64:ff9b::192.0.2.1")),
	)
}

func TestAcc_NAT64Synthesize_UseHexVariadic(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::nat64_synthesize("192.0.2.1", "64:ff9b::/96", true) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("64:ff9b::c000:201")),
	)
}

func TestAcc_NAT64Synthesize_InvalidPrefixLength(t *testing.T) {
	runErrorTest(t,
		`output "test" { value = provider::burnham::nat64_synthesize("192.0.2.1", "2001:db8::/44") }`,
		regexp.MustCompile(`length must be`),
	)
}

func TestAcc_NAT64Extract_NonSlash96Prefix(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::nat64_extract("2001:db8::c0:2:2100:0", "2001:db8::/64") }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("192.0.2.33")),
	)
}

func TestAcc_NAT64Extract_RoundTrip(t *testing.T) {
	runOutputTest(t,
		`
				locals {
					synthesized = provider::burnham::nat64_synthesize("192.0.2.1", "64:ff9b::/96", true)
					recovered   = provider::burnham::nat64_extract(local.synthesized)
				}
				output "test" { value = local.recovered }
			`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("192.0.2.1")),
	)
}

func TestAcc_NAT64PrefixValid(t *testing.T) {
	runOutputTest(t,
		`
				output "wkp"      { value = provider::burnham::nat64_prefix_valid("64:ff9b::/96") }
				output "rfc8215"  { value = provider::burnham::nat64_prefix_valid("64:ff9b:1::/48") }
				output "wronglen" { value = provider::burnham::nat64_prefix_valid("2001:db8::/44") }
				output "ipv4"     { value = provider::burnham::nat64_prefix_valid("10.0.0.0/24") }
			`,
		statecheck.ExpectKnownOutputValue("wkp", knownvalue.Bool(true)),
		statecheck.ExpectKnownOutputValue("rfc8215", knownvalue.Bool(true)),
		statecheck.ExpectKnownOutputValue("wronglen", knownvalue.Bool(false)),
		statecheck.ExpectKnownOutputValue("ipv4", knownvalue.Bool(false)),
	)
}

func TestAcc_NAT64SynthesizeCIDRs(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::nat64_synthesize_cidrs(["203.0.113.0/24"], "64:ff9b::/96") }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.ListExact([]knownvalue.Check{
			knownvalue.StringExact("64:ff9b::203.0.113.0/120"),
		})),
	)
}

// ─── NPTv6 (RFC 6296) ────────────────────────────────────────────

func TestAcc_NPTv6Translate_RoundTripIsIdentity(t *testing.T) {
	runOutputTest(t,
		`
				locals {
					original   = "fd00::10:0:1"
					translated = provider::burnham::nptv6_translate(local.original, "fd00::/48", "2001:db8::/48")
					reversed   = provider::burnham::nptv6_translate(local.translated, "2001:db8::/48", "fd00::/48")
				}
				output "test" { value = local.reversed }
			`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("fd00::10:0:1")),
	)
}

// ─── Mixed notation ──────────────────────────────────────────────

func TestAcc_IPToMixedNotation(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::ip_to_mixed_notation("64:ff9b::c000:201") }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("64:ff9b::192.0.2.1")),
	)
}

func TestAcc_IPv4ToIPv4Mapped(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::ipv4_to_ipv4_mapped("192.0.2.1") }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("::ffff:192.0.2.1")),
	)
}

// ─── IPAM — covers null-or-string return type ────────────────────

func TestAcc_CIDRFindFree_Available(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::cidr_find_free(["10.0.0.0/16"], ["10.0.0.0/24", "10.0.1.0/24"], 24) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("10.0.2.0/24")),
	)
}

func TestAcc_CIDRFindFree_ExhaustedReturnsNull(t *testing.T) {
	runOutputTest(t,
		`
				locals {
					result = provider::burnham::cidr_find_free(["10.0.0.0/30"], ["10.0.0.0/30"], 24)
				}
				output "is_null" { value = local.result == null }
			`,
		statecheck.ExpectKnownOutputValue("is_null", knownvalue.Bool(true)),
	)
}

// ─── Coverage fillers — wrappers not exercised above ─────────────

func TestAcc_CIDRContains(t *testing.T) {
	runOutputTest(t,
		`
				output "supernet_subnet" { value = provider::burnham::cidr_contains("10.0.0.0/8", "10.1.2.0/24") }
				output "supernet_ip"     { value = provider::burnham::cidr_contains("10.0.0.0/8", "10.1.2.3") }
				output "disjoint"        { value = provider::burnham::cidr_contains("10.0.0.0/8", "192.168.0.0/16") }
			`,
		statecheck.ExpectKnownOutputValue("supernet_subnet", knownvalue.Bool(true)),
		statecheck.ExpectKnownOutputValue("supernet_ip", knownvalue.Bool(true)),
		statecheck.ExpectKnownOutputValue("disjoint", knownvalue.Bool(false)),
	)
}

func TestAcc_CIDRExpand(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::cidr_expand("10.0.0.0/30") }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.ListExact([]knownvalue.Check{
			knownvalue.StringExact("10.0.0.0"),
			knownvalue.StringExact("10.0.0.1"),
			knownvalue.StringExact("10.0.0.2"),
			knownvalue.StringExact("10.0.0.3"),
		})),
	)
}

func TestAcc_CIDRIsPrivate(t *testing.T) {
	runOutputTest(t,
		`
				output "rfc1918" { value = provider::burnham::cidr_is_private("10.0.0.0/8") }
				output "public"  { value = provider::burnham::cidr_is_private("8.8.8.0/24") }
				output "ula"     { value = provider::burnham::cidr_is_private("fd00::/8") }
			`,
		statecheck.ExpectKnownOutputValue("rfc1918", knownvalue.Bool(true)),
		statecheck.ExpectKnownOutputValue("public", knownvalue.Bool(false)),
		statecheck.ExpectKnownOutputValue("ula", knownvalue.Bool(true)),
	)
}

func TestAcc_CIDRLastIP(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::cidr_last_ip("10.0.0.0/24") }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("10.0.0.255")),
	)
}

func TestAcc_CIDRVersion(t *testing.T) {
	runOutputTest(t,
		`
				output "v4" { value = provider::burnham::cidr_version("10.0.0.0/8") }
				output "v6" { value = provider::burnham::cidr_version("2001:db8::/32") }
			`,
		statecheck.ExpectKnownOutputValue("v4", knownvalue.Int64Exact(4)),
		statecheck.ExpectKnownOutputValue("v6", knownvalue.Int64Exact(6)),
	)
}

func TestAcc_CIDRsOverlapAny(t *testing.T) {
	runOutputTest(t,
		`
				output "overlapping" { value = provider::burnham::cidrs_overlap_any(["10.4.0.0/16"], ["10.0.0.0/16", "10.4.0.0/16"]) }
				output "disjoint"    { value = provider::burnham::cidrs_overlap_any(["10.4.0.0/16"], ["10.0.0.0/16", "10.1.0.0/16"]) }
			`,
		statecheck.ExpectKnownOutputValue("overlapping", knownvalue.Bool(true)),
		statecheck.ExpectKnownOutputValue("disjoint", knownvalue.Bool(false)),
	)
}

func TestAcc_NAT64SynthesizeCIDR_Singular(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::nat64_synthesize_cidr("192.0.2.0/24", "64:ff9b::/96") }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("64:ff9b::192.0.2.0/120")),
	)
}

// ─── Filter / version helpers ────────────────────────────────────

func TestAcc_CIDRFilterVersion(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::cidr_filter_version(["10.0.0.0/8", "172.16.0.0/12", "2001:db8::/32", "fd00::/8"], 4) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.ListExact([]knownvalue.Check{
			knownvalue.StringExact("10.0.0.0/8"),
			knownvalue.StringExact("172.16.0.0/12"),
		})),
	)
}

// ─── Malformed-input tests ───────────────────────────────────────
//
// These exist to catch the common shapes of user mistakes — passing an IP
// where a CIDR is expected, mixing IPv4 and IPv6, typos in addresses, etc.
// Not adversarial fuzzing: each scenario is something a real Terraform user
// is plausibly going to do at least once.

func TestAcc_Errors_CIDRExpectedGotIP(t *testing.T) {
	// User passes a bare IP where a CIDR is required — easy mistake when
	// pasting addresses without their /N.
	runErrorTest(t,
		`output "test" { value = provider::burnham::cidr_first_ip("10.0.0.1") }`,
		regexp.MustCompile(`invalid CIDR`),
	)
}

func TestAcc_Errors_IPExpectedGotCIDR(t *testing.T) {
	// Reverse mistake: CIDR pasted into an IP-typed argument.
	runErrorTest(t,
		`output "test" { value = provider::burnham::ip_in_cidr("10.0.0.1/24", "10.0.0.0/24") }`,
		regexp.MustCompile(`invalid IP`),
	)
}

func TestAcc_Errors_CIDRMissingPrefixLength(t *testing.T) {
	// Forgot the /N when building a CIDR list.
	runErrorTest(t,
		`output "test" { value = provider::burnham::cidr_merge(["10.0.0.0"]) }`,
		regexp.MustCompile(`invalid CIDR`),
	)
}

func TestAcc_Errors_TypoExtraDotInIP(t *testing.T) {
	runErrorTest(t,
		`output "test" { value = provider::burnham::ip_add("10.0.0.0.1", 5) }`,
		regexp.MustCompile(`invalid IP`),
	)
}

func TestAcc_Errors_RangeMixedFamilies(t *testing.T) {
	// Range where the first IP is IPv4 and the last is IPv6.
	runErrorTest(t,
		`output "test" { value = provider::burnham::range_to_cidrs("10.0.0.1", "::1") }`,
		regexp.MustCompile(`same address family`),
	)
}

func TestAcc_Errors_IPSubtractMixedFamilies(t *testing.T) {
	runErrorTest(t,
		`output "test" { value = provider::burnham::ip_subtract("10.0.0.1", "::1") }`,
		regexp.MustCompile(`same family`),
	)
}

func TestAcc_Errors_WildcardOnIPv6(t *testing.T) {
	// cidr_wildcard is IPv4-only; passing an IPv6 prefix should be a clear
	// error rather than a silent surprise.
	runErrorTest(t,
		`output "test" { value = provider::burnham::cidr_wildcard("2001:db8::/32") }`,
		regexp.MustCompile(`only defined for IPv4`),
	)
}

func TestAcc_Errors_NAT64IPv6AsIPv4Arg(t *testing.T) {
	// Argument-order mistake: the IPv4 slot received an IPv6 address.
	runErrorTest(t,
		`output "test" { value = provider::burnham::nat64_synthesize("2001:db8::1", "64:ff9b::/96") }`,
		regexp.MustCompile(`expected IPv4`),
	)
}

func TestAcc_Errors_IPAddUnderflow(t *testing.T) {
	// Subtracting from the lowest IPv4 address — should report underflow,
	// not silently wrap.
	runErrorTest(t,
		`output "test" { value = provider::burnham::ip_add("0.0.0.0", -1) }`,
		regexp.MustCompile(`underflow`),
	)
}

func TestAcc_Errors_CIDREnumerateZeroNewbits(t *testing.T) {
	// `cidr_enumerate` with newbits=0 doesn't produce subnets — surface a
	// clear error rather than returning the input prefix as a single-element
	// list (which would be misleading).
	runErrorTest(t,
		`output "test" { value = provider::burnham::cidr_enumerate("10.0.0.0/24", 0) }`,
		// Terraform may wrap "must be positive" across a newline; match the
		// contiguous prefix instead.
		regexp.MustCompile(`newbits must be`),
	)
}

// ─── pigeon_throughput (RFC 1149 / RFC 2549) ───────────────────────────

func TestAcc_PigeonThroughput_FrameFormatQuoteVerbatim(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::pigeon_throughput(1, 1, 100).frame_format }`,
		statecheck.ExpectKnownOutputValue("test",
			knownvalue.StringRegexp(regexp.MustCompile(`^The IP datagram is printed.*scroll of paper.*one leg.*\(RFC 1149 §3\)$`)),
		),
	)
}

func TestAcc_PigeonThroughput_MTUMatchesRFC(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::pigeon_throughput(10, 1024, 100).mtu_bytes }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.Int64Exact(256)),
	)
}

func TestAcc_PigeonThroughput_BirdsRequiredFragments(t *testing.T) {
	// 1024 bytes / 256 MTU = 4 birds. RFC 1149 §3 — one datagram per carrier.
	runOutputTest(t,
		`output "test" { value = provider::burnham::pigeon_throughput(10, 1024, 100).birds_required }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.Int64Exact(4)),
	)
}

func TestAcc_PigeonThroughput_BirdsRequiredCeils(t *testing.T) {
	// 257 bytes still needs 2 birds (one with 256 + one with 1).
	runOutputTest(t,
		`output "test" { value = provider::burnham::pigeon_throughput(10, 257, 100).birds_required }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.Int64Exact(2)),
	)
}

func TestAcc_PigeonThroughput_ZeroPayloadZeroBirds(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::pigeon_throughput(10, 0, 100).birds_required }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.Int64Exact(0)),
	)
}

func TestAcc_PigeonThroughput_FlightTimeFromCruiseSpeed(t *testing.T) {
	// 80 km at 80 km/h = 1 hour = 3600 seconds.
	runOutputTest(t,
		`output "test" { value = provider::burnham::pigeon_throughput(80, 256, 100).flight_time_seconds }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.NumberExact(big.NewFloat(3600))),
	)
}

func TestAcc_PigeonThroughput_LossCappedAt50Percent(t *testing.T) {
	// 1% per 100 km, capped at 50%. 10000 km would otherwise be 100% — should be 0.5.
	runOutputTest(t,
		`output "test" { value = provider::burnham::pigeon_throughput(10000, 256, 100).packet_loss_probability }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.NumberExact(big.NewFloat(0.5))),
	)
}

func TestAcc_PigeonThroughput_QoSClassLowAltitude(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::pigeon_throughput(10, 256, 25).qos_class }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("low altitude")),
	)
}

func TestAcc_PigeonThroughput_QoSClassExpress(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::pigeon_throughput(10, 256, 1000).qos_class }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("express")),
	)
}

func TestAcc_PigeonThroughput_QoSClassStratus(t *testing.T) {
	runOutputTest(t,
		`output "test" { value = provider::burnham::pigeon_throughput(10, 256, 5000).qos_class }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("stratus")),
	)
}

func TestAcc_PigeonThroughput_RFCCitationsPresent(t *testing.T) {
	// At least four citations should be returned, all referencing RFC 1149 or RFC 2549.
	runOutputTest(t,
		`output "test" { value = length(provider::burnham::pigeon_throughput(1, 1, 100).rfc_citations) }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.Int64Exact(4)),
	)
}

func TestAcc_PigeonThroughput_RejectsExcessiveAltitude(t *testing.T) {
	runErrorTest(t,
		`output "test" { value = provider::burnham::pigeon_throughput(10, 256, 15000) }`,
		regexp.MustCompile(`(?is)altitude_m\s+must\s+be\s+in\s+\[0,\s*12000\]`),
	)
}

func TestAcc_PigeonThroughput_RejectsNegativeDistance(t *testing.T) {
	runErrorTest(t,
		`output "test" { value = provider::burnham::pigeon_throughput(-1, 256, 100) }`,
		regexp.MustCompile(`(?is)distance_km\s+must\s+be\s+>=\s+0`),
	)
}

func TestAcc_PigeonThroughput_RejectsNegativePayload(t *testing.T) {
	runErrorTest(t,
		`output "test" { value = provider::burnham::pigeon_throughput(10, -1, 100) }`,
		regexp.MustCompile(`(?is)payload_bytes\s+must\s+be\s+>=\s+0`),
	)
}
