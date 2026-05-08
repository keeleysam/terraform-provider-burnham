// Encode (latitude, longitude) into a geohash. Higher precision narrows the cell ten-fold per character.
output "civic_center_sf" {
  value = provider::burnham::geohash_encode(37.7749, -122.4194, 7)
  // → "9q8yyk8"  (≈ 152 m × 152 m)
}

output "country_scale" {
  value = provider::burnham::geohash_encode(37.7749, -122.4194, 3)
  // → "9q8"      (≈ 156 km × 156 km)
}
