// hexencode: bytes to lowercase hex.
output "hex" {
  value = provider::burnham::hexencode("Hi")
  // → "4869"
}
