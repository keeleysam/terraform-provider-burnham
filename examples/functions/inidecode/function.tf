# Parse INI into nested objects keyed by section.
output "db_host" {
  value = provider::burnham::inidecode(file("${path.module}/app.ini")).database.host
}
