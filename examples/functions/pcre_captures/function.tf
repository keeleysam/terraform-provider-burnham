# Named capture groups, readable by name or by number.
output "parts" {
  value = provider::burnham::pcre_captures("(?<year>\\d{4})-(?<month>\\d{2})", "2026-07")
  # → { "0" = "2026-07", "1" = "2026", "2" = "07", "month" = "07", "year" = "2026" }
}
