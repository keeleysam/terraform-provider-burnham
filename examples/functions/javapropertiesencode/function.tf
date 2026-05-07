// Encode a flat object as a Java .properties body. Keys sorted; non-ASCII characters emitted as \uXXXX for portability.
output "props" {
  value = provider::burnham::javapropertiesencode({
    "app.name"     = "frontend"
    "app.replicas" = 3
  })
}
/* →
app.name=frontend
app.replicas=3
*/
