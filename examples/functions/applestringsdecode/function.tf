// Parse an Apple .strings localization file (UTF-8 or UTF-16 BOM auto-detected).
output "translations" {
  value = provider::burnham::applestringsdecode("\"hello\" = \"Hello\";\n\"goodbye\" = \"Goodbye\";\n")
  // → { hello = "Hello", goodbye = "Goodbye" }
}
