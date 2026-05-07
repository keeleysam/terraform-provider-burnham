// JSON Patch (RFC 6902) — apply an ordered list of add/remove/replace/move/copy/test ops.
output "patched" {
  value = provider::burnham::json_patch(
    { name = "web", replicas = 2, env = { LOG_LEVEL = "info", DEBUG = "true" } },
    [
      { op = "replace", path = "/replicas", value = 5 },
      { op = "add", path = "/env/REGION", value = "us-east-1" },
      { op = "remove", path = "/env/DEBUG" },
    ],
  )
}
/* →
{
  name = "web"
  replicas = 5
  env = { LOG_LEVEL = "info", REGION = "us-east-1" }
}
*/
