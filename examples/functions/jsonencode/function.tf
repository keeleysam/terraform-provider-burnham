// Pretty-printed JSON. Defaults to tab indent; pass options.indent to override.
output "data" {
  value = provider::burnham::jsonencode({ name = "alice", count = 3 })
}
/* →
{
    "count": 3,
    "name": "alice"
}
*/
