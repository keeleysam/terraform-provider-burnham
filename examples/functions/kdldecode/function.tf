# Parse a KDL document (https://kdl.dev) into a list of node objects.
output "config" {
  value = provider::burnham::kdldecode(file("${path.module}/config.kdl"))
}
