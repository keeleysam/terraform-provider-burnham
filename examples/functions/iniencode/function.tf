// Render nested objects as INI. Top-level keys become sections.
output "config" {
  value = provider::burnham::iniencode({
    database = { host = "localhost", port = "5432" }
  })
}
/* →
[database]
host = localhost
port = 5432
*/
