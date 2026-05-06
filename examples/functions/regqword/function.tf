# Tagged 64-bit integer for use inside a regencode payload.
output "max_size" {
  value = provider::burnham::regqword(4294967296)
}
