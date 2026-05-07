// JSONPath (RFC 9535) query — returns a list of matching nodes.
output "low_priced_titles" {
  value = provider::burnham::jsonpath_query(
    {
      store = {
        book = [
          { title = "Sword", price = 5 },
          { title = "Helm", price = 12 },
          { title = "Shield", price = 3 },
        ]
      }
    },
    "$.store.book[?@.price < 10].title",
  )
  // → ["Sword", "Shield"]
}
