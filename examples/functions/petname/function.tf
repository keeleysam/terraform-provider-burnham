// Deterministic Heroku-style petname derived from a seed. Same seed → same petname. Use for human-readable resource names that stay stable across plans.
output "default_two_words" {
  value = provider::burnham::petname("env-prod")
  // → "<adjective>-<noun>", e.g. "swift-fox"
}

output "three_words" {
  value = provider::burnham::petname("env-prod", { words = 3 })
  // → "<adverb>-<adjective>-<noun>", e.g. "gently-swift-fox"
}

output "underscored" {
  value = provider::burnham::petname("env-prod", { separator = "_" })
}

output "single_noun" {
  value = provider::burnham::petname("env-prod", { words = 1 })
  // → just a noun, e.g. "fox"
}
