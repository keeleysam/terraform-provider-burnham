// JSON Merge Patch (RFC 7396) — overlay shape-shaped patches; null deletes keys; arrays replace wholesale.
output "prod_config" {
  value = provider::burnham::json_merge_patch(
    { replicas = 2, env = { LOG_LEVEL = "info", DEBUG = "true" } },
    { replicas = 10, env = { LOG_LEVEL = "warn", DEBUG = null } },
  )
}
/* →
{
  replicas = 10
  env = { LOG_LEVEL = "warn" }   // DEBUG removed
}
*/
