// YAML with block style and literal scalars for multi-line strings.
output "configmap" {
  value = provider::burnham::yamlencode({
    apiVersion = "v1"
    kind       = "ConfigMap"
  })
}
/* →
apiVersion: v1
kind: ConfigMap
*/
