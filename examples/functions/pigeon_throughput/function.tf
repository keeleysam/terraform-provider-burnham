/*
RFC 1149 / RFC 2549 — IP-over-Avian-Carriers throughput calculator. Computes the spec-implied numbers (MTU, birds required, flight time, throughput, packet loss probability, QoS class) for a (distance, payload, altitude) flight plan.

Outputs a fixed-shape object plus an `rfc_citations` list so callers can audit which clause each field came from.
*/
output "short_haul" {
  value = provider::burnham::pigeon_throughput(80, 1024, 100)
  // Distance: 80 km, payload: 1024 B, altitude: 100 m (RFC 2549 "standard" QoS class).
  // 1024 B / 256 MTU = 4 birds; 80 km / 80 km/h = 3600 s flight time.
}

// At higher altitude the QoS class shifts per RFC 2549 §3 framework.
output "express_class" {
  value = provider::burnham::pigeon_throughput(50, 256, 1000).qos_class
  // → "express"
}

// Long distance triggers RFC 2549 §6 loss capped at 50 %.
output "long_haul_loss" {
  value = provider::burnham::pigeon_throughput(20000, 256, 500).packet_loss_probability
  // → 0.5 (capped)
}
