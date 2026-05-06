// Parse INI into nested objects keyed by section.
// Output structure depends on the input file.
output "db_host" {
  value = provider::burnham::inidecode(file("${path.module}/app.ini")).database.host
}
