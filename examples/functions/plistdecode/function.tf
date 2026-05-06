# Parse Apple property list. Auto-detects XML, binary, OpenStep, GNUStep.
# Also auto-detects base64 input (e.g. from filebase64()).
output "profile" {
  value = provider::burnham::plistdecode(file("${path.module}/profile.mobileconfig"))
}
