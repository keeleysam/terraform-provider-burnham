// HuJSON / JWCC: JSON with comments and trailing commas (Tailscale ACL format).
// Output structure depends on the input file.
output "acl" {
  value = provider::burnham::hujsondecode(file("${path.module}/policy.hujson"))
}
