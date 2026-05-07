// Encode a list as NDJSON (one JSON value per line, trailing newline).
output "log_lines" {
  value = provider::burnham::ndjsonencode([
    { id = 1, msg = "started" },
    { id = 2, msg = "ready" },
  ])
}
/* →
{"id":1,"msg":"started"}
{"id":2,"msg":"ready"}
*/
