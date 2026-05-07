// Parse a Java .properties body into a string-keyed object. Comments (# and !), =/:/whitespace separators, and \uXXXX escapes are all handled.
output "config" {
  value = provider::burnham::javapropertiesdecode("# Spring config\nserver.port=8080\nserver.name=api")
  // → { "server.port" = "8080", "server.name" = "api" }
}
