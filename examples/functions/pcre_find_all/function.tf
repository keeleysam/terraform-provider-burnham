output "numbers" {
  value = provider::burnham::pcre_find_all("\\d+", "a1 b22 c333")
  # → ["1", "22", "333"]
}
