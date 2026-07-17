// RFC 8785 canonical JSON: sorted keys, no insignificant whitespace, ES6 number formatting. The exact bytes to feed a signing or MAC primitive so the same logical value always hashes identically.
output "canonical" {
  value = provider::burnham::json_canonicalize({
    sub   = "alice"
    iat   = 1516239022
    admin = true
    roles = ["b", "a"]
  })
  // → "{\"admin\":true,\"iat\":1516239022,\"roles\":[\"b\",\"a\"],\"sub\":\"alice\"}"
}

// Numbers follow the ES6 double rules: 4.50 becomes 4.5, 1e30 becomes 1e+30.
output "numbers" {
  value = provider::burnham::json_canonicalize({ amount = 4.50, big = 1e30 })
  // → "{\"amount\":4.5,\"big\":1e+30}"
}

// The canonical bytes are what you sign: stable input to hmac / hkdf / pkcs7_sign.
output "signed_digest" {
  value = provider::burnham::hmac("sha256", "k", provider::burnham::json_canonicalize({ b = 2, a = 1 }))
  // → HMAC-SHA256 over the bytes {"a":1,"b":2}
}
