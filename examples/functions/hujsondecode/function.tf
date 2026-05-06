# HuJSON / JWCC: JSON with comments and trailing commas (Tailscale ACL format).
output "acl" {
  value = provider::burnham::hujsondecode(file("${path.module}/policy.hujson"))
}
