# Parse Valve VDF (Steam / Source engine config format).
output "appstate" {
  value = provider::burnham::vdfdecode(file("${path.module}/appmanifest_730.acf"))
}
