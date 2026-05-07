// JMESPath query — extract fields from a nested structure without long try() chains.
output "active_users" {
  value = provider::burnham::jmespath_query(
    {
      users = [
        { name = "alice", active = true },
        { name = "bob", active = false },
        { name = "carol", active = true },
      ]
    },
    "users[?active].name",
  )
  // → ["alice", "carol"]
}
