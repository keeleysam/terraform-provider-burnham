// Decode a geohash into the centre point and bounding box of its cell.
output "civic_center_decoded" {
  value = provider::burnham::geohash_decode("9q8yyk8")
  // → { latitude = 37.7749…, longitude = -122.4194…, lat_min = …, lat_max = …, lon_min = …, lon_max = … }
}

// Round-trip a coordinate through encode + decode and check the centre lands inside the cell.
output "is_centre_inside_cell" {
  value = (
    provider::burnham::geohash_decode(provider::burnham::geohash_encode(37.7749, -122.4194, 7)).lat_min < 37.7749 &&
    provider::burnham::geohash_decode(provider::burnham::geohash_encode(37.7749, -122.4194, 7)).lat_max > 37.7749
  )
}
