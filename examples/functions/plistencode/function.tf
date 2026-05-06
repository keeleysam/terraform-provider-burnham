# Build a configuration profile / .mobileconfig payload.
output "wifi_profile" {
  value = provider::burnham::plistencode({
    PayloadDisplayName = "WiFi"
    PayloadIdentifier  = "com.example.wifi"
    PayloadVersion     = 1
  })
}
