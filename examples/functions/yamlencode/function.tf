# YAML with block style, literal scalars for multi-line strings.
output "k8s_configmap" {
  value = provider::burnham::yamlencode({
    apiVersion = "v1"
    kind       = "ConfigMap"
    data       = { script = "#!/bin/bash\necho hello\n" }
  })
}
