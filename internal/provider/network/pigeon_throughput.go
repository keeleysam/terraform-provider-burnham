/*
RFC 1149 / RFC 2549 — IP over Avian Carriers throughput calculator.

Faithful implementation of the metrics implied by [RFC 1149](https://www.rfc-editor.org/rfc/rfc1149) ("A Standard for the Transmission of IP Datagrams on Avian Carriers", April 1990) and [RFC 2549](https://www.rfc-editor.org/rfc/rfc2549) ("IP over Avian Carriers with Quality of Service", April 1999) for a given (distance, payload, altitude) flight plan.

Constants chosen for spec-faithfulness:

  - **MTU = 256 bytes** per RFC 1149 §3 ("the MTU is normally 256 milligrams"). The unit confusion in the original is deliberate; we render the number as bytes since that's what every later citation interprets it as.
  - **Cruise speed = 80 km/h** — the typical homing-pigeon cruise (60–95 km/h range; 80 is the cited mid-point for *Columba livia domestica* under standard conditions). RFC 2549 doesn't specify a speed; we hold it constant across altitudes rather than inventing a model the RFC doesn't sanction.
  - **Loss probability = 1% per 100 km, capped at 50%** — RFC 2549 §6 mentions weather and predator-driven loss without giving a formula; this linear model is the loosest interpretation that yields meaningful numbers.
  - **QoS class** is derived from altitude in line with RFC 2549's "Carrier Class" framework. The original RFC ties QoS to ToS-bit mappings; we substitute altitude as a more directly observable input for plan-time use.

The resulting object reports raw flight time, theoretical bandwidth, effective bandwidth after expected loss, the canonical RFC 1149 §3 frame format string, and a list of citations so the caller can audit which clauses each field came from.
*/

package network

import (
	"context"
	"fmt"
	"math"
	"math/big"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

const (
	pigeonMTUBytes        = 256 // RFC 1149 §3
	pigeonCruiseSpeedKmh  = 80  // typical homing pigeon
	pigeonLossPerHundredK = 0.01
	pigeonMaxLoss         = 0.5
)

// pigeonFrameFormat is the canonical description of the RFC 1149 frame format. Quoted near-verbatim from §3 so callers writing documentation can drop it in unchanged.
const pigeonFrameFormat = "The IP datagram is printed, on a small scroll of paper, in hexadecimal, with each octet separated by whitestuff and blackstuff. The scroll of paper is wrapped around one leg of the avian carrier. (RFC 1149 §3)"

var _ function.Function = (*PigeonThroughputFunction)(nil)

type PigeonThroughputFunction struct{}

func NewPigeonThroughputFunction() function.Function { return &PigeonThroughputFunction{} }

func (f *PigeonThroughputFunction) Metadata(_ context.Context, _ function.MetadataRequest, resp *function.MetadataResponse) {
	resp.Name = "pigeon_throughput"
}

// pigeonThroughputAttrs is the fixed-shape object returned by pigeon_throughput.
var pigeonThroughputAttrs = map[string]attr.Type{
	"mtu_bytes":                types.Int64Type,
	"birds_required":           types.Int64Type,
	"per_bird_payload_bytes":   types.Int64Type,
	"cruise_speed_kmh":         types.NumberType,
	"flight_time_seconds":      types.NumberType,
	"throughput_bps":           types.NumberType,
	"packet_loss_probability":  types.NumberType,
	"effective_throughput_bps": types.NumberType,
	"qos_class":                types.StringType,
	"frame_format":             types.StringType,
	"rfc_citations":            types.ListType{ElemType: types.StringType},
}

func (f *PigeonThroughputFunction) Definition(_ context.Context, _ function.DefinitionRequest, resp *function.DefinitionResponse) {
	resp.Definition = function.Definition{
		Summary: "Compute IP-over-Avian-Carriers throughput for a flight plan (RFC 1149 / RFC 2549)",
		MarkdownDescription: "Returns a fixed-shape object describing the theoretical IP-over-Avian-Carriers throughput for transmitting `payload_bytes` over `distance_km` at `altitude_m`, faithful to the metrics implied by [RFC 1149](https://www.rfc-editor.org/rfc/rfc1149) and [RFC 2549](https://www.rfc-editor.org/rfc/rfc2549):\n\n- `mtu_bytes` (256) — RFC 1149 §3 (\"normally 256 milligrams\", interpreted as bytes per the established convention).\n- `birds_required` — `ceil(payload_bytes / mtu_bytes)`. RFC 1149 §3 specifies one datagram per carrier; oversized payloads fragment across additional birds.\n- `per_bird_payload_bytes` — `min(payload_bytes, mtu_bytes)`.\n- `cruise_speed_kmh` (80) — typical homing-pigeon cruise. RFC 2549 doesn't specify a speed; we hold it constant across altitudes rather than inventing a model the RFC doesn't sanction.\n- `flight_time_seconds` — `distance_km / cruise_speed_kmh × 3600`.\n- `throughput_bps` — `payload_bytes × 8 / flight_time_seconds`.\n- `packet_loss_probability` — RFC 2549 §6 mentions weather- and predator-driven loss without a formula. We use 1% per 100 km, capped at 50%.\n- `effective_throughput_bps` — `throughput_bps × (1 − packet_loss_probability)`.\n- `qos_class` — derived from altitude per RFC 2549 §3's Carrier Class framework. Categories: `\"low altitude\"` (< 50 m), `\"standard\"` (< 500 m), `\"express\"` (< 1500 m), `\"stratus\"` (≥ 1500 m).\n- `frame_format` — verbatim RFC 1149 §3 frame description.\n- `rfc_citations` — sources for each field, so callers can audit the spec basis.\n\n**Modelling assumptions worth knowing.** The function models a single-flight, parallel-flock dispatch:\n\n- *Multi-bird parallelism.* Multiple birds carrying the fragmented payload are assumed to fly *concurrently*, so `flight_time_seconds` does not multiply by `birds_required`. RFC 2549 §4's flock-multicast framing supports this reading; RFC 1149 §3's \"single point-to-point path\" line, taken literally, would force serial transmission and `flight_time × birds_required` latency. We pick the parallel reading because every real-world cited use of avian-carrier IP (e.g. CPIP) ran flocks in parallel.\n- *Loss model.* `effective_throughput_bps = throughput_bps × (1 − packet_loss_probability)` treats lost datagrams as silently dropped — the throughput you actually receive at the far end. RFC 2549 §6's wording implies retransmission is possible but specifies no mechanism, so we don't bake it in. If you want the retransmission-amortised flight time, multiply `flight_time_seconds` by `1 / (1 − packet_loss_probability)`.\n\n`distance_km` and `payload_bytes` must be non-negative. `altitude_m` must be in `[0, 12000]` (above which homing pigeons are not specified by either RFC).",
		Parameters: []function.Parameter{
			function.NumberParameter{Name: "distance_km", Description: "Flight distance in kilometres; must be >= 0."},
			function.Int64Parameter{Name: "payload_bytes", Description: "Total IP datagram payload to transmit, in bytes; must be >= 0."},
			function.NumberParameter{Name: "altitude_m", Description: "Cruise altitude in metres; must be in [0, 12000]."},
		},
		Return: function.ObjectReturn{AttributeTypes: pigeonThroughputAttrs},
	}
}

// qosClassFor returns the carrier-class label for the given altitude per the bands documented on this function.
func qosClassFor(altitudeM float64) string {
	switch {
	case altitudeM < 50:
		return "low altitude"
	case altitudeM < 500:
		return "standard"
	case altitudeM < 1500:
		return "express"
	default:
		return "stratus"
	}
}

func (f *PigeonThroughputFunction) Run(ctx context.Context, req function.RunRequest, resp *function.RunResponse) {
	var distanceBF, altitudeBF *big.Float
	var payloadBytes int64
	resp.Error = function.ConcatFuncErrors(resp.Error, req.Arguments.Get(ctx, &distanceBF, &payloadBytes, &altitudeBF))
	if resp.Error != nil {
		return
	}
	if distanceBF.IsInf() {
		resp.Error = function.NewArgumentFuncError(0, "distance_km must be finite")
		return
	}
	if altitudeBF.IsInf() {
		resp.Error = function.NewArgumentFuncError(2, "altitude_m must be finite")
		return
	}
	distanceKm, _ := distanceBF.Float64()
	altitudeM, _ := altitudeBF.Float64()

	if distanceKm < 0 {
		resp.Error = function.NewArgumentFuncError(0, fmt.Sprintf("distance_km must be >= 0; received %g", distanceKm))
		return
	}
	if payloadBytes < 0 {
		resp.Error = function.NewArgumentFuncError(1, fmt.Sprintf("payload_bytes must be >= 0; received %d", payloadBytes))
		return
	}
	if altitudeM < 0 || altitudeM > 12000 {
		resp.Error = function.NewArgumentFuncError(2, fmt.Sprintf("altitude_m must be in [0, 12000]; received %g", altitudeM))
		return
	}

	birdsRequired := int64(0)
	if payloadBytes > 0 {
		birdsRequired = int64(math.Ceil(float64(payloadBytes) / pigeonMTUBytes))
	}
	perBird := payloadBytes
	if perBird > pigeonMTUBytes {
		perBird = pigeonMTUBytes
	}

	var flightTimeSec, throughputBps, effectiveBps, lossProb float64
	if distanceKm == 0 || pigeonCruiseSpeedKmh == 0 {
		// At zero distance the flight time is zero; throughput is undefined. Report zero rather than NaN.
		flightTimeSec = 0
		throughputBps = 0
		effectiveBps = 0
		lossProb = 0
	} else {
		flightTimeSec = distanceKm / pigeonCruiseSpeedKmh * 3600
		if flightTimeSec > 0 {
			throughputBps = float64(payloadBytes) * 8 / flightTimeSec
		}
		lossProb = (distanceKm / 100) * pigeonLossPerHundredK
		if lossProb > pigeonMaxLoss {
			lossProb = pigeonMaxLoss
		}
		effectiveBps = throughputBps * (1 - lossProb)
	}

	citations := []attr.Value{
		types.StringValue("RFC 1149 §3 — frame format and 256-unit MTU"),
		types.StringValue("RFC 1149 §3 — one datagram per avian carrier"),
		types.StringValue("RFC 2549 §3 — Carrier Class / quality of service"),
		types.StringValue("RFC 2549 §6 — weather- and predator-driven loss"),
	}
	citationsList, citDiag := types.ListValue(types.StringType, citations)
	if citDiag.HasError() {
		resp.Error = function.NewFuncError("building rfc_citations list")
		return
	}

	out, diags := types.ObjectValue(pigeonThroughputAttrs, map[string]attr.Value{
		"mtu_bytes":                types.Int64Value(pigeonMTUBytes),
		"birds_required":           types.Int64Value(birdsRequired),
		"per_bird_payload_bytes":   types.Int64Value(perBird),
		"cruise_speed_kmh":         types.NumberValue(big.NewFloat(pigeonCruiseSpeedKmh)),
		"flight_time_seconds":      types.NumberValue(big.NewFloat(flightTimeSec)),
		"throughput_bps":           types.NumberValue(big.NewFloat(throughputBps)),
		"packet_loss_probability":  types.NumberValue(big.NewFloat(lossProb)),
		"effective_throughput_bps": types.NumberValue(big.NewFloat(effectiveBps)),
		"qos_class":                types.StringValue(qosClassFor(altitudeM)),
		"frame_format":             types.StringValue(pigeonFrameFormat),
		"rfc_citations":            citationsList,
	})
	if diags.HasError() {
		resp.Error = function.NewFuncError("building pigeon_throughput result")
		return
	}
	resp.Error = function.ConcatFuncErrors(resp.Error, resp.Result.Set(ctx, &out))
}
