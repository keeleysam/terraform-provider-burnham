// A green -> yellow -> red threshold ramp for dashboard gauges (5 evenly-spaced stops).
output "threshold_ramp" {
  value = provider::burnham::color_ramp(["#22c55e", "#eab308", "#ef4444"], 5)
}
