# Tagged-object representation of an Apple plist <data> (binary) element.
output "cert_blob" {
  value = provider::burnham::plistdata(filebase64("${path.module}/cert.der"))
}
