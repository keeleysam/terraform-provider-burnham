package provider

import (
	"math/big"
	"regexp"
	"testing"

	"github.com/hashicorp/terraform-plugin-testing/knownvalue"
	"github.com/hashicorp/terraform-plugin-testing/statecheck"
)

// pigeon_throughput (RFC 1149 / RFC 2549) lives in the network family but its tests are kept in their own file so the standard CIDR / IP / NAT64 / NPTv6 test surface in `acceptance_network_test.go` doesn't get diluted by the joke-RFC-faithful one. Same family, different reading speed.

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

func TestAcc_PigeonThroughput_QoSClassStandard(t *testing.T) {
	// Altitude 200 m sits in the [50, 500) standard band.
	runOutputTest(t,
		`output "test" { value = provider::burnham::pigeon_throughput(10, 256, 200).qos_class }`,
		statecheck.ExpectKnownOutputValue("test", knownvalue.StringExact("standard")),
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
