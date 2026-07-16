// base64decode: lenient, accepts standard or URL-safe alphabets, padded or not.
// A friction-free superset of core base64decode (which rejects URL-safe input).
output "decoded" {
  value = provider::burnham::base64decode("SGVsbG8") // unpadded, fine
  // → "Hello"
}
