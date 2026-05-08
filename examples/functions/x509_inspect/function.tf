/*
Decode an X.509 certificate's metadata into a structured object. Useful for plan-time assertions about expiry, SANs, and key/extended-key usage.

This function reads structure only — it does NOT verify the certificate's signature, validity period, or trust chain. Run a separate signing or trust-validation step if you need to make security decisions based on the result.
*/
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

output "expiry" {
  value = provider::burnham::x509_inspect(local.cert_pem).not_after
  // → "2036-05-05T05:44:54Z"
}

output "sans" {
  value = provider::burnham::x509_inspect(local.cert_pem).dns_names
  // → ["test.burnham.example", "www.test.burnham.example"]
}

output "is_server_auth" {
  value = contains(provider::burnham::x509_inspect(local.cert_pem).ext_key_usage, "serverAuth")
  // → true
}
