// cedarevaluate: authorize a request against a Cedar policy document and get the
// decision, for unit-testing authorization policies at plan time. Uses the
// official cedar-go engine, so the decision is authoritative.
output "decision" {
  value = provider::burnham::cedarevaluate(
    "permit (principal == User::\"alice\", action == Action::\"view\", resource in Album::\"vacation\");",
    {
      principal = { type = "User", id = "alice" }
      action    = { type = "Action", id = "view" }
      resource  = { type = "Photo", id = "sunset.jpg" }
      entities = [
        { uid = { type = "Photo", id = "sunset.jpg" }, attrs = {}, parents = [{ type = "Album", id = "vacation" }] },
      ]
    }
  )
  // → { decision = "allow", reasons = ["policy0"], errors = [] }
}
