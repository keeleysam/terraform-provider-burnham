// base32decode: lenient, case-insensitive, padding optional, whitespace ignored.
// Pass { hex_alphabet = true } for the extended-hex alphabet (NSEC3).
output "decoded" {
  value = provider::burnham::base32decode("MZXW6YTBOI")
  // → "foobar"
}
