// Parse a .env / dotenv body into a string-to-string object. All values are strings.
output "env" {
  value = provider::burnham::dotenvdecode("# comment\nDATABASE_URL=postgres://localhost\nLOG_LEVEL=info\n")
  // → { DATABASE_URL = "postgres://localhost", LOG_LEVEL = "info" }
}
