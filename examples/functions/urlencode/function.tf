// urlencode — percent-encode for a URL. Default mode "query" matches core urlencode.
// "path"/"component" use RFC 3986 %20 instead of the form-encoding "+".
output "segment" {
  value = provider::burnham::urlencode(var.name, { mode = "path" })
}

variable "name" { type = string }
