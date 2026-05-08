// Decode a full Plus code into centre + bounding box. Short codes (those that need a reference location) are not supported here — pre-extend them upstream of Terraform.
output "civic_center_decoded" {
  value = provider::burnham::pluscode_decode("849VQHFJ+X6")
  // → { latitude = …, longitude = …, lat_min/max = …, lon_min/max = …, length = 10 }
}
