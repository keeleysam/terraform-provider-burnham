// CSV with column ordering. Header row by default; pass no_header = true to suppress.
output "users_csv" {
  value = provider::burnham::csvencode(
    [{ name = "alice", role = "admin" }, { name = "bob", role = "viewer" }],
    { columns = ["name", "role"] },
  )
}
/* →
name,role
alice,admin
bob,viewer
*/
