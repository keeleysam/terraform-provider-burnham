# Force a whole-number value to encode as plist <real> instead of <integer>.
output "scale" {
  value = provider::burnham::plistreal(2)
  # → { __plist_type = "real", value = "2" }
}
