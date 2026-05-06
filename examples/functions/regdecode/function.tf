# Parse a Windows .reg export into nested key paths and typed values.
output "reg_data" {
  value = provider::burnham::regdecode(file("${path.module}/policy.reg"))
}
