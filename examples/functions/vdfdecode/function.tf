// Parse Valve VDF (Steam / Source engine config format).
// Output structure depends on the input file.
output "appstate" {
  value = provider::burnham::vdfdecode(file("${path.module}/appmanifest_730.acf"))
}
