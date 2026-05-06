// Build KDL output. Default is KDL v2; pass version = "v1" for legacy.
output "config" {
  value = provider::burnham::kdlencode([
    { name = "title", args = ["Hello"], props = {}, children = [] }
  ])
  // → "title \"Hello\"\n"
}
