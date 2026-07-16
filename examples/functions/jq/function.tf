// jq: query and reshape a decoded structure with the full jq language.
// The program is a stream, so the result is always a list (one element per value produced).
output "prod_ids" {
  value = provider::burnham::jq(
    {
      items = [
        { id = 1, tier = "prod" },
        { id = 2, tier = "dev" },
        { id = 3, tier = "prod" },
      ]
    },
    ".items[] | select(.tier == $tier) | .id",
    { vars = { tier = "prod" } },
  )
  // → [1, 3]
}
