// Encode a flat string-keyed object as an Apple .strings file body (UTF-8).
output "localizable_strings" {
  value = provider::burnham::applestringsencode({
    greeting = "Hello"
    farewell = "Goodbye"
  })
}
/* →
"farewell" = "Goodbye";
"greeting" = "Hello";
*/
