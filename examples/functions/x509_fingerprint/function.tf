// Hex fingerprint of a certificate's DER bytes. Same value as `openssl x509 -fingerprint -sha256` minus the colons.
locals {
  cert_pem = <<-EOT
    -----BEGIN CERTIFICATE-----
    MIICDTCCAb+gAwIBAgIUMgVGWFtmsm94f67URPQkQej2tdswBQYDK2VwMEMxHTAb
    BgNVBAMMFHRlc3QuYnVybmhhbS5leGFtcGxlMRUwEwYDVQQKDAxCdXJuaGFtIFRl
    c3QxCzAJBgNVBAYTAlVTMB4XDTI2MDUwODA1NDQ1NFoXDTM2MDUwNTA1NDQ1NFow
    QzEdMBsGA1UEAwwUdGVzdC5idXJuaGFtLmV4YW1wbGUxFTATBgNVBAoMDEJ1cm5o
    YW0gVGVzdDELMAkGA1UEBhMCVVMwKjAFBgMrZXADIQC/PkuXOt6DOQTVrL6iEgju
    7V35EUWLG7lVCTIke/O/F6OBxDCBwTBpBgNVHREEYjBgghR0ZXN0LmJ1cm5oYW0u
    ZXhhbXBsZYIYd3d3LnRlc3QuYnVybmhhbS5leGFtcGxlhwTAAAIBgQ9vcHNAZXhh
    bXBsZS5jb22GF2h0dHBzOi8vZXhhbXBsZS5jb20vY3NyMAkGA1UdEwQCMAAwCwYD
    VR0PBAQDAgWgMB0GA1UdJQQWMBQGCCsGAQUFBwMBBggrBgEFBQcDAjAdBgNVHQ4E
    FgQU2yLvWoHejGQd/33oyjMrnr+Gq5AwBQYDK2VwA0EAfJ8cvzfTb8Y5cC/wcB8H
    MKsGUQhtUnEQyWWM8whCWRx4Y2Lfy0gkm8O0hVqtVnpY9YGaI4D5RBvWNLO+cvIH
    Bw==
    -----END CERTIFICATE-----
  EOT
}

output "sha256_pin" {
  value = provider::burnham::x509_fingerprint(local.cert_pem, "sha256")
  // → "6d2d325a319a26c8d89f417fc543d2673bdce5f9ba2e4ae2bdc6f409f0e346cc"
}
