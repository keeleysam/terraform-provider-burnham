Returns a fixed-shape object describing the theoretical IP-over-Avian-Carriers throughput for transmitting `payload_bytes` over `distance_km` at `altitude_m`, faithful to the metrics implied by [RFC 1149](https://www.rfc-editor.org/rfc/rfc1149) and [RFC 2549](https://www.rfc-editor.org/rfc/rfc2549).

### Output fields

- `mtu_bytes` (256): RFC 1149 §3 ("normally 256 milligrams", interpreted as bytes per the established convention).
- `birds_required`: `ceil(payload_bytes / mtu_bytes)`. RFC 1149 §3 specifies one datagram per carrier; oversized payloads fragment across additional birds.
- `per_bird_payload_bytes`: `min(payload_bytes, mtu_bytes)`.
- `cruise_speed_kmh` (80): typical homing-pigeon cruise. RFC 2549 doesn't specify a speed; we hold it constant across altitudes rather than inventing a model the RFC doesn't sanction.
- `flight_time_seconds`: `distance_km / cruise_speed_kmh × 3600`.
- `throughput_bps`: `payload_bytes × 8 / flight_time_seconds`.
- `packet_loss_probability`: RFC 2549 §6 mentions weather- and predator-driven loss without a formula. We use 1% per 100 km, capped at 50%.
- `effective_throughput_bps`: `throughput_bps × (1 − packet_loss_probability)`.
- `qos_class`: derived from altitude per RFC 2549 §3's Carrier Class framework:
  - `"low altitude"` (< 50 m)
  - `"standard"` (< 500 m)
  - `"express"` (< 1500 m)
  - `"stratus"` (≥ 1500 m)
- `frame_format`: near-verbatim RFC 1149 §3 frame description.
- `rfc_citations`: sources for each field, so callers can audit the spec basis.

-> **Note:** When `distance_km` is `0`, `flight_time_seconds` is `0`, so `throughput_bps`, `effective_throughput_bps`, and `packet_loss_probability` are reported as `0` rather than applying the formulas above (which would divide by zero).

### Modelling assumptions

The function models a single-flight, parallel-flock dispatch.

- **Multi-bird parallelism.** Multiple birds carrying the fragmented payload are assumed to fly *concurrently*, so `flight_time_seconds` does not multiply by `birds_required`. RFC 2549 §4's flock-multicast framing supports this reading; RFC 1149 §3's "single point-to-point path" line, taken literally, would force serial transmission and `flight_time × birds_required` latency. We pick the parallel reading because every real-world cited use of avian-carrier IP (e.g. CPIP) ran flocks in parallel.
- **Loss model.** `effective_throughput_bps = throughput_bps × (1 − packet_loss_probability)` treats lost datagrams as silently dropped: the throughput you actually receive at the far end. RFC 2549 §6's wording implies retransmission is possible but specifies no mechanism, so we don't bake it in.

-> **Note:** For the retransmission-amortised flight time, multiply `flight_time_seconds` by `1 / (1 − packet_loss_probability)`.

~> **Note:** `distance_km` and `payload_bytes` must be non-negative, and `altitude_m` must be in `[0, 12000]` (above which homing pigeons are not specified by either RFC). Out-of-range input fails the plan.