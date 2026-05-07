// Encode a flat object as a .env body. Keys sorted; values quoted only when they contain whitespace, quotes, or special characters.
output "env_file" {
  value = provider::burnham::dotenvencode({
    DATABASE_URL = "postgres://localhost"
    GREETING     = "hello world"
  })
}
/* →
DATABASE_URL=postgres://localhost
GREETING="hello world"
*/
