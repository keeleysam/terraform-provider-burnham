// Decode a PKCS #10 CSR into a structured object. Use it to assert at plan time that a CSR's CN and SANs match what the issuing CA expects.
locals {
  csr_pem = <<-EOT
    -----BEGIN CERTIFICATE REQUEST-----
    MIIBezCCAS0CAQAwQzEdMBsGA1UEAwwUdGVzdC5idXJuaGFtLmV4YW1wbGUxFTAT
    BgNVBAoMDEJ1cm5oYW0gVGVzdDELMAkGA1UEBhMCVVMwKjAFBgMrZXADIQBdRy86
    BPldM5uQC7PkxejuEo3cQV4s7VRFh4IwX/Bjz6CBtjCBswYJKoZIhvcNAQkOMYGl
    MIGiMGkGA1UdEQRiMGCCFHRlc3QuYnVybmhhbS5leGFtcGxlghh3d3cudGVzdC5i
    dXJuaGFtLmV4YW1wbGWHBMAAAgGBD29wc0BleGFtcGxlLmNvbYYXaHR0cHM6Ly9l
    eGFtcGxlLmNvbS9jc3IwCQYDVR0TBAIwADALBgNVHQ8EBAMCBaAwHQYDVR0lBBYw
    FAYIKwYBBQUHAwEGCCsGAQUFBwMCMAUGAytlcANBAK9YRCz7VOyYUdbUWsS4uSES
    Odr5Q/4/OQM+WhKq/ckVK43GHJ9YSbSnrBrbvHmKMsgvwSMt61Ljyi6fKro4XA0=
    -----END CERTIFICATE REQUEST-----
  EOT
}

output "subject" {
  value = provider::burnham::csr_inspect(local.csr_pem).subject
  // → "CN=test.burnham.example,O=Burnham Test,C=US"
}

output "requested_sans" {
  value = provider::burnham::csr_inspect(local.csr_pem).dns_names
  // → ["test.burnham.example", "www.test.burnham.example"]
}
