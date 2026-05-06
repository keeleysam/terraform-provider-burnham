# Tagged 32-bit integer for use inside a regencode payload.
output "build_number" {
  value = provider::burnham::regdword(0x10203)
}
