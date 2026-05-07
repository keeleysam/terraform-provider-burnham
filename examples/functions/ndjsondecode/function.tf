// Parse NDJSON (newline-delimited JSON / JSON Lines) into a list.
output "events" {
  value = provider::burnham::ndjsondecode("{\"id\":1,\"kind\":\"login\"}\n{\"id\":2,\"kind\":\"logout\"}\n")
}
/* →
[
  { id = 1, kind = "login" },
  { id = 2, kind = "logout" },
]
*/
