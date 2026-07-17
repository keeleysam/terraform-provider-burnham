// parse_kv: parse a delimited key/value string into a map(string).
// Robust where the naive split("=")/split(",") idiom is not: equals inside a
// value, surrounding whitespace, and quoted separators are all handled.
output "basic" {
  value = provider::burnham::parse_kv("a=1,b=2")
  // → { a = "1", b = "2" }
}

output "equals_in_value" {
  value = provider::burnham::parse_kv("url=https://x?a=b")
  // → { url = "https://x?a=b" }
}

output "quoted_separator" {
  value = provider::burnham::parse_kv("a=\"x,y\",b=2")
  // → { a = "x,y", b = "2" }
}

output "custom_separators" {
  value = provider::burnham::parse_kv("a:1;b:2", { pair_sep = ";", kv_sep = ":" })
  // → { a = "1", b = "2" }
}
