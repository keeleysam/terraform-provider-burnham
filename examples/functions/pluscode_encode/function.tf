// Encode (latitude, longitude) into a Plus code (Open Location Code). Length 10 (default in upstream OLC) gives ~3.5 m precision.
output "civic_center_sf" {
  value = provider::burnham::pluscode_encode(37.7749, -122.4194, 10)
  // → e.g. "849VQHFJ+X6"
}

// Length must be even between 2 and 10, or any value 11–15.
output "high_precision" {
  value = provider::burnham::pluscode_encode(37.7749, -122.4194, 11)
}
