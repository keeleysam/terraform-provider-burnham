// Tagged 64-bit integer for use inside a regencode payload.
output "max_size" {
  value = provider::burnham::regqword(4294967296)
  // → { __reg_type = "qword", value = "4294967296" }
}
