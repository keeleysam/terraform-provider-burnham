output "fields" {
  value = provider::burnham::pcre_split("\\s*,\\s*", "a, b ,c")
  # → ["a", "b", "c"]
}
