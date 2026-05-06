# Tagged binary value (hex string) for use inside a regencode payload.
output "blob" {
  value = provider::burnham::regbinary("48656c6c6f")  # "Hello"
}
