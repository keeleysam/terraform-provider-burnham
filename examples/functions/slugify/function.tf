// URL-safe slugs. Transliterates non-ASCII characters into their nearest ASCII equivalent — different from corefunc's case-conversion functions.
output "english_lowercase" {
  value = provider::burnham::slugify("Hello, World!")
  // → "hello-world"
}

output "accented_to_ascii" {
  value = provider::burnham::slugify("Café au Lait")
  // → "cafe-au-lait"
}

output "underscore_separator" {
  value = provider::burnham::slugify("hello world", { separator = "_" })
  // → "hello_world"
}
