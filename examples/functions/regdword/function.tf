// Tagged 32-bit integer for use inside a regencode payload.
// Use decimal — HCL doesn't accept 0x... hex literals.
output "build_number" {
  value = provider::burnham::regdword(66051) // 0x10203
  // → { __reg_type = "dword", value = "66051" }
}
